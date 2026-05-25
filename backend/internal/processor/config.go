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

func (p *Processor) ListConfigFiles(ctx appctx.RequestContext, projectID string) (map[string]any, *model.ErrorDetail) {
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	files := p.store.ListConfigFiles(project.ID)
	return map[string]any{
		"projectId": project.ID,
		"files":     files,
	}, nil
}

func (p *Processor) CreateConfigFile(ctx appctx.RequestContext, projectID string, req model.CreateConfigFileRequest) (model.ConfigFile, *model.ErrorDetail) {
	project, err := p.requireReadableProject(ctx, projectID)
	if err != nil {
		return model.ConfigFile{}, err
	}
	if !util.CanManageProjectConfigFiles(ctx.Actor, project.Members) {
		return model.ConfigFile{}, model.Forbidden("Only project admins or developers can create config files")
	}
	name := model.NormalizeConfigFileName(req.Name)
	if len(name) < 2 {
		return model.ConfigFile{}, model.InvalidInput("name is required")
	}
	if p.store.ConfigFileNameExists(project.ID, name) {
		return model.ConfigFile{}, model.Conflict(fmt.Sprintf("Config file %q already exists", name))
	}
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = "blank"
	}
	sourceID := strings.TrimSpace(req.SourceID)
	if appErr := p.validateConfigFileSource(ctx, sourceType, sourceID); appErr != nil {
		return model.ConfigFile{}, appErr
	}
	now := time.Now().UTC()
	file := model.ConfigFile{
		ID:          model.ConfigFileID(project.ID, name),
		ProjectID:   project.ID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		SourceType:  sourceType,
		SourceID:    sourceID,
		Prefix:      model.ConfigFilePrefix(name),
		SortOrder:   100,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	audit := newAudit(ctx.ActorName(), "config_file.create", "config_file", file.ID, project.ID, map[string]any{
		"name":       file.Name,
		"sourceType": file.SourceType,
		"sourceId":   file.SourceID,
	})
	if err := p.store.SaveConfigFile(file, audit); err != nil {
		return model.ConfigFile{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return file, nil
}

func (p *Processor) validateConfigFileSource(ctx appctx.RequestContext, sourceType, sourceID string) *model.ErrorDetail {
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

func (p *Processor) ListConfigs(ctx appctx.RequestContext, projectID, environment string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
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

	p.store.EnsureProjectConfigFiles(projectID)
	entries := p.store.ListConfigEntries(projectID, environment)
	for index := range entries {
		if entries[index].IsSensitive && !revealSensitive {
			entries[index].Value = maskedValue
		}
	}

	payload := map[string]any{
		"projectId":    projectID,
		"environment":  environment,
		"entries":      entries,
		"files":        p.store.ListConfigFiles(projectID),
		"entryCount":   len(entries),
		"maskedValues": !revealSensitive,
	}
	logger.Config.Info("configs listed")
	return payload, nil
}

func (p *Processor) CreateConfig(ctx appctx.RequestContext, projectID string, req model.CreateConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
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
	if _, ok := p.store.FindConfigByKey(projectID, environment, key); ok {
		logger.Config.Warn("create config conflict")
		return model.ConfigEntry{}, model.Conflict(fmt.Sprintf("Config key %q already exists in %q", key, environment))
	}

	valueType := util.NormalizeValueType(req.ValueType)
	if valueType == "" {
		logger.Config.Warn("create config invalid")
		return model.ConfigEntry{}, model.InvalidInput("valueType must be string, number, boolean, json, or yaml")
	}
	configFileID, fileErr := p.resolveConfigFileID(projectID, strings.TrimSpace(req.ConfigFileID), key)
	if fileErr != nil {
		return model.ConfigEntry{}, fileErr
	}

	now := time.Now().UTC()
	entry := model.ConfigEntry{
		ID:           util.NewID("cfg"),
		ProjectID:    projectID,
		Environment:  environment,
		ConfigFileID: configFileID,
		Key:          key,
		Value:        req.Value,
		ValueType:    valueType,
		IsSensitive:  req.IsSensitive,
		UpdatedBy:    ctx.ActorName(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	version := newVersion(entry.ID, nil, entry.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "create config"))
	audit := newAudit(ctx.ActorName(), "config.create", "config_entry", entry.ID, projectID, map[string]any{
		"environment":  entry.Environment,
		"configFileId": entry.ConfigFileID,
		"key":          entry.Key,
		"isSensitive":  entry.IsSensitive,
	})
	if err := p.store.SaveConfig([]model.ConfigEntry{entry}, []model.ConfigVersion{version}, audit); err != nil {
		logger.Config.Error("create config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config created")
	return entry, nil
}

func (p *Processor) UpdateConfig(ctx appctx.RequestContext, projectID, configID string, req model.UpdateConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	logger.Config.Info("update config requested")

	if req.ConfigFileID == nil && req.Key == nil && req.Value == nil && req.ValueType == nil && req.IsSensitive == nil {
		logger.Config.Warn("update config invalid")
		return model.ConfigEntry{}, model.InvalidInput("No config fields provided for update")
	}

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("update config not found")
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, entry.Environment); err != nil {
		logger.Config.Warn("update config denied")
		return model.ConfigEntry{}, err
	}

	oldValue := entry.Value
	if req.ConfigFileID != nil {
		configFileID, fileErr := p.resolveConfigFileID(projectID, strings.TrimSpace(*req.ConfigFileID), entry.Key)
		if fileErr != nil {
			return model.ConfigEntry{}, fileErr
		}
		entry.ConfigFileID = configFileID
	}
	if req.Key != nil {
		nextKey := strings.TrimSpace(*req.Key)
		if nextKey == "" {
			logger.Config.Warn("update config invalid")
			return model.ConfigEntry{}, model.InvalidInput("key is required")
		}
		if nextKey != entry.Key {
			if existing, exists := p.store.FindConfigByKey(projectID, entry.Environment, nextKey); exists && existing.ID != entry.ID {
				logger.Config.Warn("update config conflict")
				return model.ConfigEntry{}, model.Conflict(fmt.Sprintf("Config key %q already exists in %q", nextKey, entry.Environment))
			}
			entry.Key = nextKey
			if req.ConfigFileID == nil {
				configFileID, fileErr := p.resolveConfigFileID(projectID, entry.ConfigFileID, entry.Key)
				if fileErr != nil {
					return model.ConfigEntry{}, fileErr
				}
				entry.ConfigFileID = configFileID
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
		"environment":  entry.Environment,
		"configFileId": entry.ConfigFileID,
		"key":          entry.Key,
		"valueType":    entry.ValueType,
		"isSensitive":  entry.IsSensitive,
	})
	if err := p.store.SaveConfig([]model.ConfigEntry{entry}, []model.ConfigVersion{version}, audit); err != nil {
		logger.Config.Error("update config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config updated")
	return entry, nil
}

func (p *Processor) resolveConfigFileID(projectID, requestedFileID, key string) (string, *model.ErrorDetail) {
	p.store.EnsureProjectConfigFiles(projectID)
	if requestedFileID != "" {
		if _, ok := p.store.FindConfigFile(projectID, requestedFileID); !ok {
			return "", model.NotFound(fmt.Sprintf("Config file %q not found", requestedFileID))
		}
		return requestedFileID, nil
	}
	return model.StandardConfigFileForKey(projectID, key, time.Now().UTC()).ID, nil
}

func (p *Processor) DeleteConfig(ctx appctx.RequestContext, projectID, configID string) (map[string]any, *model.ErrorDetail) {
	logger.Config.Info("delete config requested")

	entry, ok := p.store.FindConfig(projectID, configID)
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
	deleted, ok, err := p.store.DeleteConfig(projectID, configID, audit)
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

func (p *Processor) ListConfigVersions(ctx appctx.RequestContext, projectID, configID string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	logger.Config.Info("list config versions requested")

	entry, ok := p.store.FindConfig(projectID, configID)
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

func (p *Processor) RollbackConfig(ctx appctx.RequestContext, projectID, configID string, req model.RollbackConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	logger.Config.Info("rollback config requested")

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("rollback config not found")
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, entry.Environment); err != nil {
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
	if err := p.store.SaveConfig([]model.ConfigEntry{entry}, []model.ConfigVersion{nextVersion}, audit); err != nil {
		logger.Config.Error("rollback config persistence failed")
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config rolled back")
	return entry, nil
}

func (p *Processor) ListConfigHistory(ctx appctx.RequestContext, projectID, environment string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
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

	revisions := p.store.ListConfigRevisions(projectID, environment)
	if !revealSensitive {
		maskRevisionValues(revisions)
	}

	payload := map[string]any{
		"projectId":     projectID,
		"environment":   environment,
		"revisions":     revisions,
		"revisionCount": len(revisions),
		"maskedValues":  !revealSensitive,
	}
	logger.Config.Info("config history listed")
	return payload, nil
}

func (p *Processor) RollbackConfigRevision(ctx appctx.RequestContext, projectID string, req model.RollbackConfigRevisionRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	revisionID := strings.TrimSpace(req.RevisionID)
	logger.Config.Info("rollback config revision requested")

	if environment == "" || revisionID == "" {
		logger.Config.Warn("rollback config revision invalid")
		return nil, model.InvalidInput("environment and revisionId are required")
	}
	if _, err := p.requireWritableProjectEnvironment(ctx, projectID, environment); err != nil {
		logger.Config.Warn("rollback config revision denied")
		return nil, err
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("rollback config revision failed")
		return nil, err
	}

	revision, ok := selectRevision(p.store.ListConfigRevisions(projectID, environment), revisionID)
	if !ok {
		logger.Config.Warn("rollback config revision not found")
		return nil, model.NotFound("Config revision not found")
	}

	changeReason := util.Fallback(req.ChangeReason, "rollback config revision")
	audit := newAudit(ctx.ActorName(), "config.rollback_revision", "config_revision", revision.ID, projectID, map[string]any{
		"environment": environment,
		"revisionId":  revision.ID,
		"entryCount":  len(revision.Entries),
	})
	if err := p.store.RestoreConfigRevision(projectID, environment, revision, ctx.ActorName(), changeReason, audit); err != nil {
		logger.Config.Error("rollback config revision persistence failed")
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config revision rolled back")
	return map[string]any{
		"restored":    true,
		"projectId":   projectID,
		"environment": environment,
		"revisionId":  revision.ID,
		"entryCount":  len(revision.Entries),
	}, nil
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

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("extract configs parse failed")
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("extract configs invalid")
		return nil, model.InvalidInput("No config entries found in file content")
	}

	entries, created, updated, unchanged := p.previewParsedConfigs(projectID, environment, parsed)
	payload := map[string]any{
		"projectId":   projectID,
		"environment": environment,
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

func (p *Processor) previewParsedConfigs(projectID, environment string, parsed []util.ParsedConfigEntry) ([]model.ConfigRevisionEntry, int, int, int) {
	p.store.EnsureProjectConfigFiles(projectID)
	entries := make([]model.ConfigRevisionEntry, 0, len(parsed))
	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		isSensitive := util.LooksSensitive(parsedEntry.Key)
		if existing, ok := p.store.FindConfigByKey(projectID, environment, parsedEntry.Key); ok {
			isSensitive = existing.IsSensitive || isSensitive
			if existing.Value == parsedEntry.Value && existing.ValueType == parsedEntry.ValueType {
				unchanged++
			} else {
				updated++
			}
		} else {
			created++
		}
		configFileID := model.StandardConfigFileForKey(projectID, parsedEntry.Key, time.Now().UTC()).ID
		entries = append(entries, model.ConfigRevisionEntry{
			ConfigFileID: configFileID,
			Key:          parsedEntry.Key,
			Value:        parsedEntry.Value,
			ValueType:    parsedEntry.ValueType,
			IsSensitive:  isSensitive,
		})
	}
	return entries, created, updated, unchanged
}

func (p *Processor) ImportConfigs(ctx appctx.RequestContext, projectID string, req model.ImportConfigRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
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

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("import configs parse failed")
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("import configs invalid")
		return nil, model.InvalidInput("No config entries found in file content")
	}

	p.store.EnsureProjectConfigFiles(projectID)
	requestedConfigFileID := strings.TrimSpace(req.ConfigFileID)
	if requestedConfigFileID != "" {
		if _, ok := p.store.FindConfigFile(projectID, requestedConfigFileID); !ok {
			return nil, model.NotFound(fmt.Sprintf("Config file %q not found", requestedConfigFileID))
		}
	}
	entries := make([]model.ConfigEntry, 0, len(parsed))
	versions := make([]model.ConfigVersion, 0, len(parsed))
	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		existing, ok := p.store.FindConfigByKey(projectID, environment, parsedEntry.Key)
		configFileID := requestedConfigFileID
		if configFileID == "" {
			configFileID = model.StandardConfigFileForKey(projectID, parsedEntry.Key, time.Now().UTC()).ID
		}
		if !ok {
			now := time.Now().UTC()
			entry := model.ConfigEntry{
				ID:           util.NewID("cfg"),
				ProjectID:    projectID,
				Environment:  environment,
				ConfigFileID: configFileID,
				Key:          parsedEntry.Key,
				Value:        parsedEntry.Value,
				ValueType:    parsedEntry.ValueType,
				IsSensitive:  util.LooksSensitive(parsedEntry.Key),
				UpdatedBy:    ctx.ActorName(),
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			entries = append(entries, entry)
			versions = append(versions, newVersion(entry.ID, nil, entry.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "import config file")))
			created++
			continue
		}

		if existing.Value == parsedEntry.Value && existing.ValueType == parsedEntry.ValueType {
			unchanged++
			continue
		}

		oldValue := existing.Value
		existing.ConfigFileID = configFileID
		existing.Value = parsedEntry.Value
		existing.ValueType = parsedEntry.ValueType
		existing.IsSensitive = existing.IsSensitive || util.LooksSensitive(parsedEntry.Key)
		existing.UpdatedBy = ctx.ActorName()
		existing.UpdatedAt = time.Now().UTC()
		entries = append(entries, existing)
		versions = append(versions, newVersion(existing.ID, &oldValue, existing.Value, ctx.ActorName(), util.Fallback(req.ChangeReason, "import config file")))
		updated++
	}

	audit := newAudit(ctx.ActorName(), "config.import", "config_file", "", projectID, map[string]any{
		"environment":  environment,
		"format":       req.Format,
		"configFileId": requestedConfigFileID,
		"created":      created,
		"updated":      updated,
		"unchanged":    unchanged,
	})
	if err := p.store.SaveConfig(entries, versions, audit); err != nil {
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
		"changeReason": util.Fallback(req.ChangeReason, "import config file"),
	}
	logger.Config.Info("configs imported")
	return payload, nil
}
