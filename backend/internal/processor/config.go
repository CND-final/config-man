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

func (p *Processor) ListConfigs(ctx appctx.RequestContext, projectID, environment string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
	fields := []any{
		"operation", "config.list",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		"reveal_sensitive", revealSensitive,
	}
	logger.Config.Info("list configs requested", fields...)

	if environment == "" {
		logger.Config.Warn("list configs invalid", append(fields, "reason", "missing_environment")...)
		return nil, model.InvalidInput(`Query parameter "env" is required`)
	}
	if revealSensitive && !util.CanRevealSensitive(ctx.Actor) {
		logger.Config.Warn("list configs denied", append(fields, "reason", "reveal_sensitive_not_allowed")...)
		return nil, model.Forbidden("Role cannot reveal sensitive values")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("list configs failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return nil, err
	}

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
		"entryCount":   len(entries),
		"maskedValues": !revealSensitive,
	}
	logger.Config.Info("configs listed", append(fields, "entry_count", len(entries), "masked_values", !revealSensitive)...)
	return payload, nil
}

func (p *Processor) CreateConfig(ctx appctx.RequestContext, projectID string, req model.CreateConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	key := strings.TrimSpace(req.Key)
	fields := []any{
		"operation", "config.create",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		logger.FieldConfigKey, key,
		"is_sensitive", req.IsSensitive,
	}
	logger.Config.Info("create config requested", fields...)

	if environment == "" || key == "" {
		logger.Config.Warn("create config invalid", append(fields, "reason", "missing_environment_or_key")...)
		return model.ConfigEntry{}, model.InvalidInput("environment and key are required")
	}
	if !util.CanWriteEnvironment(ctx.Actor, environment) {
		logger.Config.Warn("create config denied", append(fields, "reason", "environment_write_not_allowed")...)
		return model.ConfigEntry{}, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, environment))
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("create config failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return model.ConfigEntry{}, err
	}
	if _, ok := p.store.FindConfigByKey(projectID, environment, key); ok {
		logger.Config.Warn("create config conflict", fields...)
		return model.ConfigEntry{}, model.Conflict(fmt.Sprintf("Config key %q already exists in %q", key, environment))
	}

	valueType := util.NormalizeValueType(req.ValueType)
	if valueType == "" {
		logger.Config.Warn("create config invalid", append(fields, "reason", "unsupported_value_type", "value_type", req.ValueType)...)
		return model.ConfigEntry{}, model.InvalidInput("valueType must be string, number, boolean, or json")
	}

	now := time.Now().UTC()
	entry := model.ConfigEntry{
		ID:          util.NewID("cfg"),
		ProjectID:   projectID,
		Environment: environment,
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
		"key":         entry.Key,
		"isSensitive": entry.IsSensitive,
	})
	if err := p.store.SaveConfig(entry, version, audit); err != nil {
		logger.Config.Error("create config persistence failed", append(fields, "error", err)...)
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config created", append(fields, logger.FieldConfigID, entry.ID, "value_type", entry.ValueType)...)
	return entry, nil
}

func (p *Processor) UpdateConfig(ctx appctx.RequestContext, projectID, configID string, req model.UpdateConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	fields := []any{
		"operation", "config.update",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldConfigID, configID,
	}
	logger.Config.Info("update config requested", fields...)

	if req.Value == nil && req.ValueType == nil && req.IsSensitive == nil {
		logger.Config.Warn("update config invalid", append(fields, "reason", "empty_update")...)
		return model.ConfigEntry{}, model.InvalidInput("No config fields provided for update")
	}

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("update config not found", fields...)
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	fields = append(fields, logger.FieldEnvironment, entry.Environment, logger.FieldConfigKey, entry.Key)
	if !util.CanWriteEnvironment(ctx.Actor, entry.Environment) {
		logger.Config.Warn("update config denied", append(fields, "reason", "environment_write_not_allowed")...)
		return model.ConfigEntry{}, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, entry.Environment))
	}

	oldValue := entry.Value
	if req.Value != nil {
		entry.Value = *req.Value
	}
	if req.ValueType != nil {
		valueType := util.NormalizeValueType(*req.ValueType)
		if valueType == "" {
			logger.Config.Warn("update config invalid", append(fields, "reason", "unsupported_value_type", "value_type", *req.ValueType)...)
			return model.ConfigEntry{}, model.InvalidInput("valueType must be string, number, boolean, or json")
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
		"key":         entry.Key,
		"valueType":   entry.ValueType,
		"isSensitive": entry.IsSensitive,
	})
	if err := p.store.SaveConfig(entry, version, audit); err != nil {
		logger.Config.Error("update config persistence failed", append(fields, "error", err)...)
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config updated", append(fields,
		"value_updated", req.Value != nil,
		"value_type_updated", req.ValueType != nil,
		"sensitive_updated", req.IsSensitive != nil,
		"is_sensitive", entry.IsSensitive,
	)...)
	return entry, nil
}

func (p *Processor) DeleteConfig(ctx appctx.RequestContext, projectID, configID string) (map[string]any, *model.ErrorDetail) {
	fields := []any{
		"operation", "config.delete",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldConfigID, configID,
	}
	logger.Config.Info("delete config requested", fields...)

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("delete config not found", fields...)
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	fields = append(fields, logger.FieldEnvironment, entry.Environment, logger.FieldConfigKey, entry.Key)
	if !util.CanWriteEnvironment(ctx.Actor, entry.Environment) {
		logger.Config.Warn("delete config denied", append(fields, "reason", "environment_write_not_allowed")...)
		return nil, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, entry.Environment))
	}

	audit := newAudit(ctx.ActorName(), "config.delete", "config_entry", configID, projectID, map[string]any{
		"environment": entry.Environment,
		"key":         entry.Key,
	})
	deleted, ok, err := p.store.DeleteConfig(projectID, configID, audit)
	if err != nil {
		logger.Config.Error("delete config persistence failed", append(fields, "error", err)...)
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}
	if !ok {
		logger.Config.Warn("delete config disappeared before delete", fields...)
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}

	logger.Config.Info("config deleted", fields...)
	return map[string]any{"deleted": true, "config": deleted}, nil
}

func (p *Processor) ListConfigVersions(ctx appctx.RequestContext, projectID, configID string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	fields := []any{
		"operation", "config.version.list",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldConfigID, configID,
		"reveal_sensitive", revealSensitive,
	}
	logger.Config.Info("list config versions requested", fields...)

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("list config versions not found", fields...)
		return nil, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	fields = append(fields, logger.FieldEnvironment, entry.Environment, logger.FieldConfigKey, entry.Key)
	if revealSensitive && !util.CanRevealSensitive(ctx.Actor) {
		logger.Config.Warn("list config versions denied", append(fields, "reason", "reveal_sensitive_not_allowed")...)
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
	logger.Config.Info("config versions listed", append(fields, "version_count", len(versions), "masked_values", entry.IsSensitive && !revealSensitive)...)
	return payload, nil
}

func (p *Processor) RollbackConfig(ctx appctx.RequestContext, projectID, configID string, req model.RollbackConfigRequest) (model.ConfigEntry, *model.ErrorDetail) {
	fields := []any{
		"operation", "config.rollback",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldConfigID, configID,
	}
	logger.Config.Info("rollback config requested", fields...)

	entry, ok := p.store.FindConfig(projectID, configID)
	if !ok {
		logger.Config.Warn("rollback config not found", fields...)
		return model.ConfigEntry{}, model.NotFound(fmt.Sprintf("Config %q not found for project %q", configID, projectID))
	}
	fields = append(fields, logger.FieldEnvironment, entry.Environment, logger.FieldConfigKey, entry.Key)
	if !util.CanWriteEnvironment(ctx.Actor, entry.Environment) {
		logger.Config.Warn("rollback config denied", append(fields, "reason", "environment_write_not_allowed")...)
		return model.ConfigEntry{}, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, entry.Environment))
	}

	versions := p.store.ListConfigVersions(configID)
	version, ok := selectRollbackVersion(versions, strings.TrimSpace(req.VersionID))
	if !ok {
		logger.Config.Warn("rollback config version not found", append(fields, "version_id", req.VersionID)...)
		return model.ConfigEntry{}, model.NotFound("Rollback version not found")
	}
	if version.OldValue == nil {
		logger.Config.Warn("rollback config unavailable", append(fields, "version_id", version.ID, "reason", "no_previous_value")...)
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
	if err := p.store.SaveConfig(entry, nextVersion, audit); err != nil {
		logger.Config.Error("rollback config persistence failed", append(fields, "error", err)...)
		return model.ConfigEntry{}, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config rolled back", append(fields, "rollback_version_id", version.ID)...)
	return entry, nil
}

func (p *Processor) ListConfigHistory(ctx appctx.RequestContext, projectID, environment string, revealSensitive bool) (map[string]any, *model.ErrorDetail) {
	environment = strings.TrimSpace(environment)
	fields := []any{
		"operation", "config.history.list",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		"reveal_sensitive", revealSensitive,
	}
	logger.Config.Info("list config history requested", fields...)

	if environment == "" {
		logger.Config.Warn("list config history invalid", append(fields, "reason", "missing_environment")...)
		return nil, model.InvalidInput(`Query parameter "env" is required`)
	}
	if revealSensitive && !util.CanRevealSensitive(ctx.Actor) {
		logger.Config.Warn("list config history denied", append(fields, "reason", "reveal_sensitive_not_allowed")...)
		return nil, model.Forbidden("Role cannot reveal sensitive values")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("list config history failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return nil, err
	}

	snapshots := p.store.ListConfigSnapshots(projectID, environment)
	if !revealSensitive {
		maskSnapshotValues(snapshots)
	}

	payload := map[string]any{
		"projectId":     projectID,
		"environment":   environment,
		"snapshots":     snapshots,
		"snapshotCount": len(snapshots),
		"maskedValues":  !revealSensitive,
	}
	logger.Config.Info("config history listed", append(fields, "snapshot_count", len(snapshots), "masked_values", !revealSensitive)...)
	return payload, nil
}

func (p *Processor) RollbackConfigSnapshot(ctx appctx.RequestContext, projectID string, req model.RollbackConfigSnapshotRequest) (map[string]any, *model.ErrorDetail) {
	environment := strings.TrimSpace(req.Environment)
	snapshotID := strings.TrimSpace(req.SnapshotID)
	fields := []any{
		"operation", "config.history.rollback",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		"snapshot_id", snapshotID,
	}
	logger.Config.Info("rollback config snapshot requested", fields...)

	if environment == "" || snapshotID == "" {
		logger.Config.Warn("rollback config snapshot invalid", append(fields, "reason", "missing_environment_or_snapshot")...)
		return nil, model.InvalidInput("environment and snapshotId are required")
	}
	if !util.CanWriteEnvironment(ctx.Actor, environment) {
		logger.Config.Warn("rollback config snapshot denied", append(fields, "reason", "environment_write_not_allowed")...)
		return nil, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, environment))
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("rollback config snapshot failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return nil, err
	}

	snapshot, ok := selectSnapshot(p.store.ListConfigSnapshots(projectID, environment), snapshotID)
	if !ok {
		logger.Config.Warn("rollback config snapshot not found", fields...)
		return nil, model.NotFound("Config snapshot not found")
	}

	changeReason := util.Fallback(req.ChangeReason, "rollback config snapshot")
	audit := newAudit(ctx.ActorName(), "config.rollback_snapshot", "config_snapshot", snapshot.ID, projectID, map[string]any{
		"environment": environment,
		"snapshotId":  snapshot.ID,
		"entryCount":  len(snapshot.Entries),
	})
	if err := p.store.RestoreConfigSnapshot(projectID, environment, snapshot, ctx.ActorName(), changeReason, audit); err != nil {
		logger.Config.Error("rollback config snapshot persistence failed", append(fields, "error", err)...)
		return nil, model.InternalError("database persistence failed: " + err.Error())
	}

	logger.Config.Info("config snapshot rolled back", append(fields, "entry_count", len(snapshot.Entries))...)
	return map[string]any{
		"restored":    true,
		"projectId":   projectID,
		"environment": environment,
		"snapshotId":  snapshot.ID,
		"entryCount":  len(snapshot.Entries),
	}, nil
}

func maskSnapshotValues(snapshots []model.ConfigSnapshot) {
	for snapshotIndex := range snapshots {
		for entryIndex := range snapshots[snapshotIndex].Entries {
			if snapshots[snapshotIndex].Entries[entryIndex].IsSensitive {
				snapshots[snapshotIndex].Entries[entryIndex].Value = maskedValue
			}
		}
	}
}

func selectSnapshot(snapshots []model.ConfigSnapshot, snapshotID string) (model.ConfigSnapshot, bool) {
	for _, snapshot := range snapshots {
		if snapshot.ID == snapshotID {
			return snapshot, true
		}
	}
	return model.ConfigSnapshot{}, false
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
	fields := []any{
		"operation", "config.extract",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		"format", req.Format,
	}
	logger.Config.Info("extract configs requested", fields...)

	if environment == "" {
		logger.Config.Warn("extract configs invalid", append(fields, "reason", "missing_environment")...)
		return nil, model.InvalidInput("environment is required")
	}
	if !util.CanWriteEnvironment(ctx.Actor, environment) {
		logger.Config.Warn("extract configs denied", append(fields, "reason", "environment_write_not_allowed")...)
		return nil, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, environment))
	}
	if !util.IsSupportedConfigFormat(req.Format) {
		logger.Config.Warn("extract configs invalid", append(fields, "reason", "unsupported_format")...)
		return nil, model.InvalidInput("format must be json, yaml, or properties")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("extract configs failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return nil, err
	}

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("extract configs parse failed", append(fields, "error", parseErr)...)
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("extract configs invalid", append(fields, "reason", "empty_config_file")...)
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
	logger.Config.Info("configs extracted", append(fields, "entry_count", len(entries), "created", created, "updated", updated, "unchanged", unchanged)...)
	return payload, nil
}

func (p *Processor) previewParsedConfigs(projectID, environment string, parsed []util.ParsedConfigEntry) ([]model.ConfigSnapshotEntry, int, int, int) {
	entries := make([]model.ConfigSnapshotEntry, 0, len(parsed))
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
		entries = append(entries, model.ConfigSnapshotEntry{
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
	fields := []any{
		"operation", "config.import",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, environment,
		"format", req.Format,
	}
	logger.Config.Info("import configs requested", fields...)

	if environment == "" {
		logger.Config.Warn("import configs invalid", append(fields, "reason", "missing_environment")...)
		return nil, model.InvalidInput("environment is required")
	}
	if !util.CanWriteEnvironment(ctx.Actor, environment) {
		logger.Config.Warn("import configs denied", append(fields, "reason", "environment_write_not_allowed")...)
		return nil, model.Forbidden(fmt.Sprintf("Role %q cannot modify %q config", ctx.Actor.Role, environment))
	}
	if !util.IsSupportedConfigFormat(req.Format) {
		logger.Config.Warn("import configs invalid", append(fields, "reason", "unsupported_format")...)
		return nil, model.InvalidInput("format must be json, yaml, or properties")
	}
	if err := p.requireEnvironment(projectID, environment); err != nil {
		logger.Config.Warn("import configs failed", append(fields, "error_kind", err.Kind, "error", err.Detail)...)
		return nil, err
	}

	parsed, parseErr := util.ParseConfigFile(req.Format, req.Content)
	if parseErr != nil {
		logger.Config.Warn("import configs parse failed", append(fields, "error", parseErr)...)
		return nil, model.InvalidInput(parseErr.Error())
	}
	if len(parsed) == 0 {
		logger.Config.Warn("import configs invalid", append(fields, "reason", "empty_config_file")...)
		return nil, model.InvalidInput("No config entries found in file content")
	}

	entries := make([]model.ConfigEntry, 0, len(parsed))
	versions := make([]model.ConfigVersion, 0, len(parsed))
	created, updated, unchanged := 0, 0, 0
	for _, parsedEntry := range parsed {
		existing, ok := p.store.FindConfigByKey(projectID, environment, parsedEntry.Key)
		if !ok {
			now := time.Now().UTC()
			entry := model.ConfigEntry{
				ID:          util.NewID("cfg"),
				ProjectID:   projectID,
				Environment: environment,
				Key:         parsedEntry.Key,
				Value:       parsedEntry.Value,
				ValueType:   parsedEntry.ValueType,
				IsSensitive: util.LooksSensitive(parsedEntry.Key),
				UpdatedBy:   ctx.ActorName(),
				CreatedAt:   now,
				UpdatedAt:   now,
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
		"environment": environment,
		"format":      req.Format,
		"created":     created,
		"updated":     updated,
		"unchanged":   unchanged,
	})
	if err := p.store.SaveConfigBatch(entries, versions, audit); err != nil {
		logger.Config.Error("import configs persistence failed", append(fields, "error", err)...)
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
	logger.Config.Info("configs imported", append(fields, "imported", len(parsed), "created", created, "updated", updated, "unchanged", unchanged)...)
	return payload, nil
}
