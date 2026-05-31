package processor

import (
	appctx "config-man/backend/internal/context"
	"fmt"
	"strings"
	"time"

	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

const maskedValue = "******"

func (p *Processor) ListConfigs(ctx appctx.RequestContext, projectID string) (map[string]any, *model.ErrorDetail) {
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	configs := p.store.ListConfigs(project.ID)
	return map[string]any{
		"projectId": project.ID,
		"configs":   configs,
		"files":     configs,
	}, nil
}

func (p *Processor) CreateConfig(ctx appctx.RequestContext, projectID string, req model.CreateConfigRequest) (model.Config, *model.ErrorDetail) {
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		return model.Config{}, err
	}
	if !util.CanManageProjectConfigs(ctx.Actor, project.Members) {
		return model.Config{}, model.Forbidden("Only project admins or developers can create configs")
	}
	name := model.NormalizeConfigName(req.Name)
	if len(name) < 2 {
		return model.Config{}, model.InvalidInput("name is required")
	}
	if p.store.ConfigNameExists(project.ID, name) {
		return model.Config{}, model.Conflict(fmt.Sprintf("Config %q already exists", name))
	}
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = "blank"
	}
	sourceID := strings.TrimSpace(req.SourceID)
	if appErr := p.validateConfigSource(ctx, sourceType, sourceID); appErr != nil {
		return model.Config{}, appErr
	}
	now := time.Now().UTC()
	file := model.Config{
		ID:          model.ConfigID(project.ID, name),
		ProjectID:   project.ID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		SourceType:  sourceType,
		SourceID:    sourceID,
		Prefix:      model.ConfigPrefix(name),
		SortOrder:   100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	audit := newAudit(ctx.ActorName(), "config.create", "config", file.ID, project.ID, map[string]any{
		"name":       file.Name,
		"sourceType": file.SourceType,
		"sourceId":   file.SourceID,
	})
	if err := p.store.SaveConfig(file, audit); err != nil {
		return model.Config{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return file, nil
}

func (p *Processor) validateConfigSource(ctx appctx.RequestContext, sourceType, sourceID string) *model.ErrorDetail {
	switch sourceType {
	case "blank", "standard":
		return nil
	case "template":
		if sourceID == "" {
			return nil
		}
		if _, ok := p.findAccessibleTemplate(ctx, sourceID); !ok {
			return model.NotFound(fmt.Sprintf("Template %q not found", sourceID))
		}
		return nil
	case "shared-config":
		if sourceID == "" {
			return nil
		}
		item, ok := p.store.FindSharedConfig(sourceID)
		if !ok {
			return model.NotFound(fmt.Sprintf("Shared config %q not found", sourceID))
		}
		if !p.canReadSharedConfig(ctx, item) {
			return model.Forbidden("You cannot use this shared config")
		}
		return nil
	default:
		return model.InvalidInput("sourceType must be blank, template, or shared-config")
	}
}

func (p *Processor) ListConfigEntries(ctx appctx.RequestContext, projectID, environment, branch string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
	branch = util.NormalizeBranch(branch)
	logger.Config.Info("list configs requested")

	if environment == "" {
		logger.Config.Warn("list configs invalid")
		return nil, model.InvalidInput(`Query parameter "env" is required`)
	}
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		logger.Config.Warn("list configs failed")
		return nil, err
	}
	if revealSensitive && !util.CanRevealProjectSensitive(ctx.Actor, project.Members) {
		logger.Config.Warn("list configs denied")
		return nil, model.Forbidden("Role cannot reveal sensitive values")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("list configs failed")
		return nil, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("list configs failed")
		return nil, err
	}

	localEntries := p.store.ListConfigEntries(projectID, environment, branch)
	configs := p.store.ListConfigs(projectID)
	entries := p.entriesWithSharedConfigInheritance(configs, localEntries, environment)
	for index := range entries {
		if entries[index].IsSensitive && !revealSensitive {
			entries[index].Value = maskedValue
		}
		if entries[index].SharedSensitive && !revealSensitive {
			entries[index].SharedValue = maskedValue
		}
	}

	payload := map[string]any{
		"projectId":    projectID,
		"environment":  environment,
		"configs":      configsWithEntries(configs, entries),
		"entries":      entries,
		"files":        configs,
		"entryCount":   len(entries),
		"maskedValues": !revealSensitive,
	}
	logger.Config.Info("configs listed")
	return payload, nil
}

func (p *Processor) entriesWithSharedConfigInheritance(configs []model.Config, localEntries []model.ConfigEntry, environment string) []model.ConfigEntry {
	entries := make([]model.ConfigEntry, 0, len(localEntries))
	localByConfigKey := make(map[string]model.ConfigEntry, len(localEntries))
	for _, entry := range localEntries {
		localByConfigKey[configEntryIdentity(entry.ConfigID, entry.Key)] = entry
	}

	for _, config := range configs {
		if config.SourceType != "shared-config" || strings.TrimSpace(config.SourceID) == "" {
			continue
		}
		sharedConfig, ok := p.store.FindSharedConfig(config.SourceID)
		if !ok {
			continue
		}
		for _, sharedEntry := range sharedEntriesForEnvironment(sharedConfig.Entries, environment) {
			identity := configEntryIdentity(config.ID, sharedEntry.Key)
			if localEntry, ok := localByConfigKey[identity]; ok {
				localEntry.Overridden = sharedConfigEntryDiffers(localEntry, sharedEntry)
				localEntry.SourceType = config.SourceType
				localEntry.SourceID = config.SourceID
				localEntry.SharedValue = sharedEntry.Value
				localEntry.SharedSensitive = sharedEntry.IsSensitive
				localByConfigKey[identity] = localEntry
				continue
			}
			entries = append(entries, model.ConfigEntry{
				ID:              inheritedConfigEntryID(config.ID, environment, sharedEntry.Key),
				ProjectID:       config.ProjectID,
				Environment:     environment,
				Branch:          branchFromEntriesOrDefault(localEntries),
				ConfigID:        config.ID,
				Key:             sharedEntry.Key,
				Value:           sharedEntry.Value,
				ValueType:       sharedEntry.ValueType,
				IsSensitive:     sharedEntry.IsSensitive,
				UpdatedBy:       sharedConfig.UpdatedBy,
				Inherited:       true,
				SourceType:      config.SourceType,
				SourceID:        config.SourceID,
				SharedValue:     sharedEntry.Value,
				SharedSensitive: sharedEntry.IsSensitive,
				CreatedAt:       sharedConfig.CreatedAt,
				UpdatedAt:       sharedConfig.UpdatedAt,
			})
		}
	}

	for _, entry := range localEntries {
		if updated, ok := localByConfigKey[configEntryIdentity(entry.ConfigID, entry.Key)]; ok {
			entries = append(entries, updated)
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func branchFromEntriesOrDefault(entries []model.ConfigEntry) string {
	if len(entries) == 0 {
		return "default"
	}
	return util.NormalizeBranch(entries[0].Branch)
}

func sharedConfigEntryDiffers(localEntry model.ConfigEntry, sharedEntry model.SharedConfigEntry) bool {
	return localEntry.Value != sharedEntry.Value ||
		localEntry.ValueType != sharedEntry.ValueType ||
		localEntry.IsSensitive != sharedEntry.IsSensitive
}

func sharedEntriesForEnvironment(entries []model.SharedConfigEntry, environment string) []model.SharedConfigEntry {
	exact := make([]model.SharedConfigEntry, 0, len(entries))
	fallback := make([]model.SharedConfigEntry, 0, len(entries))
	fallbackSeen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.Environment == environment {
			exact = append(exact, entry)
		}
		if !fallbackSeen[entry.Key] {
			fallback = append(fallback, entry)
			fallbackSeen[entry.Key] = true
		}
	}
	if len(exact) > 0 {
		return exact
	}
	return fallback
}

func configEntryIdentity(configID, key string) string {
	return configID + "\x00" + key
}

func inheritedConfigEntryID(configID, environment, key string) string {
	value := strings.ToLower(configID + "-" + environment + "-" + key)
	replacer := strings.NewReplacer(" ", "-", ":", "-", "/", "-", ".", "-", "_", "-")
	return "inherited-" + strings.Trim(replacer.Replace(value), "-")
}

func configsWithEntries(configs []model.Config, entries []model.ConfigEntry) []model.Config {
	entriesByConfigID := make(map[string][]model.ConfigEntry, len(configs))
	for _, entry := range entries {
		entriesByConfigID[entry.ConfigID] = append(entriesByConfigID[entry.ConfigID], entry)
	}
	configsWithEntries := make([]model.Config, 0, len(configs))
	for _, config := range configs {
		config.Entries = entriesByConfigID[config.ID]
		configsWithEntries = append(configsWithEntries, config)
	}
	return configsWithEntries
}

func (p *Processor) CreateConfigEntry(ctx appctx.RequestContext, projectID string, req model.CreateConfigEntryRequest) (model.ConfigEntry, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	branch := util.NormalizeBranch(req.Branch)
	key := strings.TrimSpace(req.Key)
	logger.Config.Info("create config requested")

	if environment == "" || key == "" {
		logger.Config.Warn("create config invalid")
		return model.ConfigEntry{}, model.InvalidInput("environment and key are required")
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, environment); err != nil {
		logger.Config.Warn("create config denied")
		return model.ConfigEntry{}, err
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("create config failed")
		return model.ConfigEntry{}, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("create config failed")
		return model.ConfigEntry{}, err
	}
	if _, ok := p.store.FindConfigEntryByKey(projectID, environment, branch, key); ok {
		logger.Config.Warn("create config conflict")
		return model.ConfigEntry{}, model.Conflict(fmt.Sprintf("Config key %q already exists in %q", key, environment))
	}

	valueType := util.NormalizeValueType(req.ValueType)
	if valueType == "" {
		logger.Config.Warn("create config invalid")
		return model.ConfigEntry{}, model.InvalidInput("valueType must be string, number, boolean, json, or yaml")
	}
	configFileID, fileErr := p.resolveConfigID(projectID, strings.TrimSpace(req.ConfigID), key)
	if fileErr != nil {
		return model.ConfigEntry{}, fileErr
	}

	now := time.Now().UTC()
	entry := model.ConfigEntry{
		ID:          util.NewID("cfg"),
		ProjectID:   projectID,
		Environment: environment,
		Branch:      branch,
		ConfigID:    configFileID,
		Key:         key,
		Value:       req.Value,
		ValueType:   valueType,
		IsSensitive: req.IsSensitive,
		UpdatedBy:   ctx.ActorName(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	version := newVersion(entry.ID, nil, entry.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "create config"))
	audit := newAudit(ctx.ActorName(), "config.create", "config_entry", entry.ID, projectID, map[string]any{
		"environment": entry.Environment,
		"configId":    entry.ConfigID,
		"key":         entry.Key,
		"isSensitive": entry.IsSensitive,
	})
	if err := p.store.SaveConfigEntries([]model.ConfigEntry{entry}, []model.ConfigVersion{version}, audit); err != nil {
		logger.Config.Error("create config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config created")
	return entry, nil
}

func (p *Processor) UpdateConfigEntry(ctx appctx.RequestContext, projectID, configID string, req model.UpdateConfigEntryRequest) (model.ConfigEntry, *model.ErrorDetail) {
	logger.Config.Info("update config requested")

	if req.ConfigID == nil && req.Key == nil && req.Value == nil && req.ValueType == nil && req.IsSensitive == nil {
		logger.Config.Warn("update config invalid")
		return model.ConfigEntry{}, model.InvalidInput("No config fields provided for update")
	}

	entry, ok := p.store.FindConfigEntry(projectID, configID)
	if !ok {
		logger.Config.Warn("update config not found")
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, entry.Environment); err != nil {
		logger.Config.Warn("update config denied")
		return model.ConfigEntry{}, err
	}

	oldValue := entry.Value
	if req.ConfigID != nil {
		configFileID, fileErr := p.resolveConfigID(projectID, strings.TrimSpace(*req.ConfigID), entry.Key)
		if fileErr != nil {
			return model.ConfigEntry{}, fileErr
		}
		entry.ConfigID = configFileID
	}
	if req.Key != nil {
		nextKey := strings.TrimSpace(*req.Key)
		if nextKey == "" {
			logger.Config.Warn("update config invalid")
			return model.ConfigEntry{}, model.InvalidInput("key is required")
		}
		if nextKey != entry.Key {
			if existing, exists := p.store.FindConfigEntryByKey(projectID, entry.Environment, util.NormalizeBranch(entry.Branch), nextKey); exists && existing.ID != entry.ID {
				logger.Config.Warn("update config conflict")
				return model.ConfigEntry{}, model.Conflict(fmt.Sprintf("Config key %q already exists in %q", nextKey, entry.Environment))
			}
			entry.Key = nextKey
			if req.ConfigID == nil {
				configFileID, fileErr := p.resolveConfigID(projectID, entry.ConfigID, entry.Key)
				if fileErr != nil {
					return model.ConfigEntry{}, fileErr
				}
				entry.ConfigID = configFileID
			}
		}
	}
	if req.Value != nil {
		entry.Value = *req.Value
	}
	if req.ValueType != nil {
		valueType := util.NormalizeValueType(*req.ValueType)
		if valueType == "" {
			logger.Config.Warn("update config invalid")
			return model.ConfigEntry{}, model.InvalidInput("valueType must be string, number, boolean, json, or yaml")
		}
		entry.ValueType = valueType
	}
	if req.IsSensitive != nil {
		entry.IsSensitive = *req.IsSensitive
	}
	entry.UpdatedBy = ctx.ActorName()
	entry.UpdatedAt = time.Now().UTC()

	version := newVersion(entry.ID, &oldValue, entry.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "update config"))
	audit := newAudit(ctx.ActorName(), "config.update", "config_entry", entry.ID, projectID, map[string]any{
		"environment": entry.Environment,
		"configId":    entry.ConfigID,
		"key":         entry.Key,
		"valueType":   entry.ValueType,
		"isSensitive": entry.IsSensitive,
	})
	if err := p.store.SaveConfigEntries([]model.ConfigEntry{entry}, []model.ConfigVersion{version}, audit); err != nil {
		logger.Config.Error("update config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config updated")
	return entry, nil
}

func (p *Processor) resolveConfigID(projectID, requestedConfigID, key string) (string, *model.ErrorDetail) {
	if requestedConfigID == "" {
		return "", model.InvalidInput("configId is required")
	}
	if _, ok := p.store.FindConfig(projectID, requestedConfigID); !ok {
		return "", model.NotFound(fmt.Sprintf("Config %q not found", requestedConfigID))
	}
	return requestedConfigID, nil
}

func (p *Processor) DeleteConfigEntry(ctx appctx.RequestContext, projectID, configID string) (map[string]any, *model.ErrorDetail) {
	logger.Config.Info("delete config requested")

	entry, ok := p.store.FindConfigEntry(projectID, configID)
	if !ok {
		logger.Config.Warn("delete config not found")
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, entry.Environment); err != nil {
		logger.Config.Warn("delete config denied")
		return nil, err
	}

	audit := newAudit(ctx.ActorName(), "config.delete", "config_entry", configID, projectID, map[string]any{
		"environment": entry.Environment,
		"key":         entry.Key,
	})
	deleted, ok, err := p.store.DeleteConfigEntry(projectID, configID, audit)
	if err != nil {
		logger.Config.Error("delete config persistence failed")
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}
	if !ok {
		logger.Config.Warn("delete config disappeared before delete")
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}

	logger.Config.Info("config deleted")
	return map[string]any{"deleted": true, "config": deleted}, nil
}

func (p *Processor) ListConfigEntryVersions(ctx appctx.RequestContext, projectID, configID string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	logger.Config.Info("list config versions requested")

	entry, ok := p.store.FindConfigEntry(projectID, configID)
	if !ok {
		logger.Config.Warn("list config versions not found")
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		logger.Config.Warn("list config versions denied")
		return nil, err
	}
	if revealSensitive && !util.CanRevealProjectSensitive(ctx.Actor, project.Members) {
		logger.Config.Warn("list config versions denied")
		return nil, model.Forbidden("Role cannot reveal sensitive values")
	}

	versions := p.store.ListConfigVersions(configID)
	if entry.IsSensitive && !revealSensitive {
		maskVersions(versions)
	}

	payload := map[string]any{
		"projectId":    projectID,
		"configId":     configID,
		"environment":  entry.Environment,
		"key":          entry.Key,
		"isSensitive":  entry.IsSensitive,
		"versions":     versions,
		"versionCount": len(versions),
		"maskedValues": entry.IsSensitive && !revealSensitive,
	}
	logger.Config.Info("config versions listed")
	return payload, nil
}

func (p *Processor) RollbackConfigEntry(ctx appctx.RequestContext, projectID, configID string, req model.RollbackConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	logger.Config.Info("rollback config requested")

	entry, ok := p.store.FindConfigEntry(projectID, configID)
	if !ok {
		logger.Config.Warn("rollback config not found")
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	project, err := p.requireWritableProjectEnvironment(ctx, projectID, entry.Environment)
	if err != nil {
		logger.Config.Warn("rollback config denied")
		return model.ConfigEntry{}, err
	}

	versions := p.store.ListConfigVersions(configID)
	version, ok := selectRollbackVersion(versions, strings.TrimSpace(req.VersionID))
	if !ok {
		logger.Config.Warn("rollback config version not found")
		return model.ConfigEntry{}, model.NotFound("Rollback version not found")
	}
	if version.OldValue == nil {
		logger.Config.Warn("rollback config unavailable")
		return model.ConfigEntry{}, model.InvalidInput("Selected version does not have a previous value")
	}

	oldValue := entry.Value
	entry.Value = *version.OldValue
	entry.UpdatedBy = ctx.ActorName()
	entry.UpdatedAt = time.Now().UTC()

	changeReason := util.Fallback(req.ChangeReason, "rollback config")
	nextVersion := newVersion(entry.ID, &oldValue, entry.Value, ctx.ActorName(), changeReason)
	audit := newAudit(ctx.ActorName(), "config.rollback", "config_entry", entry.ID, projectID, map[string]any{
		"environment":       entry.Environment,
		"key":               entry.Key,
		"rollbackVersionId": version.ID,
	})
	if err := p.store.SaveConfigEntries([]model.ConfigEntry{entry}, []model.ConfigVersion{nextVersion}, audit); err != nil {
		logger.Config.Error("rollback config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	_ = p.notifyProjectRollback(ctx, project, entry.Environment, entry.Key)

	logger.Config.Info("config rolled back")
	return entry, nil
}

func (p *Processor) ListConfigHistory(ctx appctx.RequestContext, projectID, environment, branch string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
	branch = util.NormalizeBranch(branch)
	logger.Config.Info("list config history requested")

	if environment == "" {
		logger.Config.Warn("list config history invalid")
		return nil, model.InvalidInput(`Query parameter "env" is required`)
	}
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		logger.Config.Warn("list config history failed")
		return nil, err
	}
	if revealSensitive && !util.CanRevealProjectSensitive(ctx.Actor, project.Members) {
		logger.Config.Warn("list config history denied")
		return nil, model.Forbidden("Role cannot reveal sensitive values")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("list config history failed")
		return nil, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("list config history failed")
		return nil, err
	}

	revisions := p.store.ListConfigRevisions(projectID, environment, branch)
	if !revealSensitive {
		maskRevisionValues(revisions)
	}

	payload := map[string]any{
		"projectId":     projectID,
		"environment":   environment,
		"branch":        branch,
		"revisions":     revisions,
		"revisionCount": len(revisions),
		"maskedValues":  !revealSensitive,
	}
	logger.Config.Info("config history listed")
	return payload, nil
}

func (p *Processor) RollbackConfigRevision(ctx appctx.RequestContext, projectID string, req model.RollbackConfigRevisionRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	branch := util.NormalizeBranch(req.Branch)
	revisionID := strings.TrimSpace(req.RevisionID)
	logger.Config.Info("rollback config revision requested")

	if environment == "" || revisionID == "" {
		logger.Config.Warn("rollback config revision invalid")
		return nil, model.InvalidInput("environment and revisionId are required")
	}
	project, err := p.requireWritableProjectEnvironment(ctx, projectID, environment)
	if err != nil {
		logger.Config.Warn("rollback config revision denied")
		return nil, err
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("rollback config revision failed")
		return nil, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("rollback config revision failed")
		return nil, err
	}

	revision, ok := selectRevision(p.store.ListConfigRevisions(projectID, environment, branch), revisionID)
	if !ok {
		logger.Config.Warn("rollback config revision not found")
		return nil, model.NotFound("Config revision not found")
	}

	changeReason := util.Fallback(req.ChangeReason, "rollback config revision")
	audit := newAudit(ctx.ActorName(), "config.rollback_revision", "config_revision", revision.ID, projectID, map[string]any{
		"environment": environment,
		"branch":      branch,
		"revisionId":  revision.ID,
		"entryCount":  len(revision.Entries),
	})
	if err := p.store.RestoreConfigRevision(projectID, environment, revision, ctx.ActorName(), changeReason, audit); err != nil {
		logger.Config.Error("rollback config revision persistence failed")
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}

	_ = p.notifyProjectRollback(ctx, project, environment, "config revision")

	logger.Config.Info("config revision rolled back")
	return map[string]any{
		"restored":    true,
		"projectId":   projectID,
		"environment": environment,
		"branch":      branch,
		"revisionId":  revision.ID,
		"entryCount":  len(revision.Entries),
	}, nil
}

func (p *Processor) notifyProjectRollback(ctx appctx.RequestContext, project model.Project, environment, target string) error {
	now := time.Now().UTC()
	notifications := make([]model.Notification, 0, len(project.Members))
	for _, member := range project.Members {
		if member.ID == "" || member.ID == ctx.Actor.ID {
			continue
		}
		notifications = append(notifications, model.Notification{
			ID:        util.NewID("not"),
			UserID:    member.ID,
			Title:     "Config rollback applied",
			Message:   fmt.Sprintf("%s %s %s was rolled back by %s", project.Name, environment, target, ctx.ActorName()),
			CreatedAt: now,
		})
	}
	if len(notifications) == 0 {
		return nil
	}
	return p.store.SaveNotifications(notifications)
}

func maskRevisionValues(revisions []model.ConfigRevision) {
	for revisionIndex := range revisions {
		for entryIndex := range revisions[revisionIndex].Entries {
			if revisions[revisionIndex].Entries[entryIndex].IsSensitive {
				revisions[revisionIndex].Entries[entryIndex].Value = maskedValue
			}
		}
	}
}

func selectRevision(revisions []model.ConfigRevision, revisionID string) (model.ConfigRevision, bool) {
	for _, revision := range revisions {
		if revision.ID == revisionID {
			return revision, true
		}
	}
	return model.ConfigRevision{}, false
}

func maskVersions(versions []model.ConfigVersion) {
	for index := range versions {
		versions[index].NewValue = maskedValue
		if versions[index].OldValue != nil {
			oldValue := maskedValue
			versions[index].OldValue = &oldValue
		}
	}
}

func selectRollbackVersion(versions []model.ConfigVersion, versionID string) (model.ConfigVersion, bool) {
	if len(versions) == 0 {
		return model.ConfigVersion{}, false
	}
	if versionID == "" {
		return versions[0], true
	}
	for _, version := range versions {
		if version.ID == versionID {
			return version, true
		}
	}
	return model.ConfigVersion{}, false
}

func (p *Processor) ExtractConfigs(ctx appctx.RequestContext, projectID string, req model.ImportConfigRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	branch := util.NormalizeBranch(req.Branch)
	logger.Config.Info("extract configs requested")

	if environment == "" {
		logger.Config.Warn("extract configs invalid")
		return nil, model.InvalidInput("environment is required")
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, environment); err != nil {
		logger.Config.Warn("extract configs denied")
		return nil, err
	}
	if !util.IsSupportedConfigFormat(req.Format) {
		logger.Config.Warn("extract configs invalid")
		return nil, model.InvalidInput("format must be json, yaml, or properties")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("extract configs failed")
		return nil, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("extract configs failed")
		return nil, err
	}

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("extract configs parse failed")
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("extract configs invalid")
		return nil, model.InvalidInput("No config entries found in file content")
	}

	configID, fileErr := p.resolveConfigID(projectID, strings.TrimSpace(req.ConfigID), "")
	if fileErr != nil {
		return nil, fileErr
	}

	entries, created, updated, unchanged := p.previewParsedConfigs(projectID, environment, branch, configID, parsed)
	payload := map[string]any{
		"projectId":   projectID,
		"environment": environment,
		"branch":      branch,
		"format":      req.Format,
		"entries":     entries,
		"entryCount":  len(entries),
		"created":     created,
		"updated":     updated,
		"unchanged":   unchanged,
	}
	logger.Config.Info("configs extracted")
	return payload, nil
}

func (p *Processor) previewParsedConfigs(projectID, environment, branch, configID string, parsed []util.ParsedConfigEntry) ([]model.ConfigRevisionEntry, int, int, int) {
	entries := make([]model.ConfigRevisionEntry, 0, len(parsed))
	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		isSensitive := util.LooksSensitive(parsedEntry.Key)
		if existing, ok := p.store.FindConfigEntryByKey(projectID, environment, branch, parsedEntry.Key); ok {
			isSensitive = existing.IsSensitive || isSensitive
			if existing.Value == parsedEntry.Value && existing.ValueType == parsedEntry.ValueType {
				unchanged++
			} else {
				updated++
			}
		} else {
			created++
		}
		entries = append(entries, model.ConfigRevisionEntry{
			ConfigID:    configID,
			Key:         parsedEntry.Key,
			Value:       parsedEntry.Value,
			ValueType:   parsedEntry.ValueType,
			IsSensitive: isSensitive,
		})
	}
	return entries, created, updated, unchanged
}

func (p *Processor) ImportConfigs(ctx appctx.RequestContext, projectID string, req model.ImportConfigRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	branch := util.NormalizeBranch(req.Branch)
	logger.Config.Info("import configs requested")

	if environment == "" {
		logger.Config.Warn("import configs invalid")
		return nil, model.InvalidInput("environment is required")
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, environment); err != nil {
		logger.Config.Warn("import configs denied")
		return nil, err
	}
	if !util.IsSupportedConfigFormat(req.Format) {
		logger.Config.Warn("import configs invalid")
		return nil, model.InvalidInput("format must be json, yaml, or properties")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("import configs failed")
		return nil, err
	}
	if err := p.requireBranch(projectID, branch); err != nil {
		logger.Config.Warn("import configs failed")
		return nil, err
	}

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("import configs parse failed")
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("import configs invalid")
		return nil, model.InvalidInput("No config entries found in file content")
	}

	requestedConfigID, fileErr := p.resolveConfigID(projectID, strings.TrimSpace(req.ConfigID), "")
	if fileErr != nil {
		return nil, fileErr
	}
	entries := make([]model.ConfigEntry, 0, len(parsed))
	versions := make([]model.ConfigVersion, 0, len(parsed))
	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		existing, ok := p.store.FindConfigEntryByKey(projectID, environment, branch, parsedEntry.Key)
		configFileID := requestedConfigID
		if !ok {
			now := time.Now().UTC()
			entry := model.ConfigEntry{
				ID:          util.NewID("cfg"),
				ProjectID:   projectID,
				Environment: environment,
				ConfigID:    configFileID,
				Key:         parsedEntry.Key,
				Value:       parsedEntry.Value,
				ValueType:   parsedEntry.ValueType,
				IsSensitive: util.LooksSensitive(parsedEntry.Key),
				UpdatedBy:   ctx.ActorName(),
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			entries = append(entries, entry)
			versions = append(versions, newVersion(entry.ID, nil, entry.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "import config")))
			created++
			continue
		}

		if existing.Value == parsedEntry.Value && existing.ValueType == parsedEntry.ValueType {
			unchanged++
			continue
		}

		oldValue := existing.Value
		existing.ConfigID = configFileID
		existing.Value = parsedEntry.Value
		existing.ValueType = parsedEntry.ValueType
		existing.IsSensitive = existing.IsSensitive || util.LooksSensitive(parsedEntry.Key)
		existing.UpdatedBy = ctx.ActorName()
		existing.UpdatedAt = time.Now().UTC()
		entries = append(entries, existing)
		versions = append(versions, newVersion(existing.ID, &oldValue, existing.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "import config")))
		updated++
	}

	audit := newAudit(ctx.ActorName(), "config.import", "config", "", projectID, map[string]any{
		"environment": environment,
		"format":      req.Format,
		"configId":    requestedConfigID,
		"created":     created,
		"updated":     updated,
		"unchanged":   unchanged,
	})
	if err := p.store.SaveConfigEntries(entries, versions, audit); err != nil {
		logger.Config.Error("import configs persistence failed")
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}

	payload := map[string]any{
		"projectId":    projectID,
		"environment":  environment,
		"format":       req.Format,
		"imported":     len(parsed),
		"created":      created,
		"updated":      updated,
		"unchanged":    unchanged,
		"changeReason": util.Fallback(req.ChangeReason, "import config"),
	}
	logger.Config.Info("configs imported")
	return payload, nil
}
