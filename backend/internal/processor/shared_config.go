package processor

import (
	"fmt"
	"strings"
	"time"

	appctx "config-man/backend/internal/context"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) SharedConfigs(ctx appctx.RequestContext) []model.SharedConfig {
	items := p.withSharedConfigImpact(p.store.ListSharedConfigs())
	if ctx.Actor.Role == model.RoleSystemAdmin {
		return items
	}

	groups := p.store.ListGroups()
	visibleGroups := map[string]bool{}
	for _, group := range groups {
		if util.GroupHasMember(group, ctx.Actor.ID) {
			visibleGroups[group.ID] = true
		}
	}

	visible := make([]model.SharedConfig, 0, len(items))
	for _, item := range items {
		if item.Scope == model.ScopeGlobal || visibleGroups[item.ScopeID] {
			visible = append(visible, item)
		}
	}
	return visible
}

func (p *Processor) CreateGlobalSharedConfig(ctx appctx.RequestContext, req model.CreateSharedConfigRequest) (model.SharedConfig, *model.ErrorDetail) {
	if ctx.Actor.Role != model.RoleSystemAdmin {
		return model.SharedConfig{}, model.Forbidden("Only system_admin can create global shared config")
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return model.SharedConfig{}, model.InvalidInput("name is required")
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "yaml"
	}
	if !util.IsSupportedConfigFormat(format) {
		return model.SharedConfig{}, model.InvalidInput("format must be properties, json, or yaml")
	}
	entries, err := normalizeSharedConfigEntries(req.Entries)
	if err != nil {
		return model.SharedConfig{}, err
	}
	now := time.Now().UTC()
	item := model.SharedConfig{
		ID:          util.NewID("shc"),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Scope:       model.ScopeGlobal,
		ScopeName:   "Global",
		Format:      format,
		Entries:     entries,
		UpdatedBy:   ctx.ActorName(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := p.store.SaveSharedConfig(item, newAudit(ctx.ActorName(), "shared_config.create", "shared_config", item.ID, "", map[string]any{"scope": item.Scope, "entryCount": len(item.Entries)})); err != nil {
		return model.SharedConfig{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return item, nil
}

func (p *Processor) UpdateGlobalSharedConfig(ctx appctx.RequestContext, id string, req model.UpdateSharedConfigRequest) (model.SharedConfig, *model.ErrorDetail) {
	if ctx.Actor.Role != model.RoleSystemAdmin {
		return model.SharedConfig{}, model.Forbidden("Only system_admin can update global shared config")
	}
	current, ok := p.store.FindSharedConfig(strings.TrimSpace(id))
	if !ok {
		return model.SharedConfig{}, model.NotFound(fmt.Sprintf("Shared config %q not found", id))
	}
	if current.Scope != model.ScopeGlobal {
		return model.SharedConfig{}, model.Forbidden("Only global shared config update is supported here")
	}
	changeReason := strings.TrimSpace(req.ChangeReason)
	if changeReason == "" {
		return model.SharedConfig{}, model.InvalidInput("changeReason is required")
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		return model.SharedConfig{}, model.InvalidInput("name is required")
	}
	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = current.Format
	}
	if !util.IsSupportedConfigFormat(format) {
		return model.SharedConfig{}, model.InvalidInput("format must be properties, json, or yaml")
	}
	entries, err := normalizeSharedConfigEntries(req.Entries)
	if err != nil {
		return model.SharedConfig{}, err
	}
	updated := current
	updated.Name = name
	updated.Description = strings.TrimSpace(req.Description)
	updated.Format = format
	updated.Entries = entries
	updated.UpdatedBy = ctx.ActorName()
	updated.UpdatedAt = time.Now().UTC()
	updated = p.withSharedConfigImpact([]model.SharedConfig{updated})[0]

	audit := newAudit(ctx.ActorName(), "shared_config.update", "shared_config", updated.ID, "", map[string]any{
		"scope":                updated.Scope,
		"name":                 updated.Name,
		"entryCount":           len(updated.Entries),
		"changeReason":         changeReason,
		"affectedProjects":     updated.InheritedBy,
		"prodEnvironmentCount": updated.ProdEnvironmentCount,
	})
	if err := p.store.SaveSharedConfig(updated, audit); err != nil {
		return model.SharedConfig{}, model.InternalError("database persistence failed: " + err.Error())
	}
	_ = p.notifySharedConfigConsumers(updated, changeReason)
	return updated, nil
}

func (p *Processor) DeleteGlobalSharedConfig(ctx appctx.RequestContext, id string) *model.ErrorDetail {
	if ctx.Actor.Role != model.RoleSystemAdmin {
		return model.Forbidden("Only system_admin can delete global shared config")
	}
	item, ok := p.store.FindSharedConfig(strings.TrimSpace(id))
	if !ok {
		return model.NotFound(fmt.Sprintf("Shared config %q not found", id))
	}
	if item.Scope != model.ScopeGlobal {
		return model.Forbidden("Only global shared config deletion is supported here")
	}
	if err := p.store.DeleteSharedConfig(item.ID, newAudit(ctx.ActorName(), "shared_config.delete", "shared_config", item.ID, "", map[string]any{"scope": item.Scope, "name": item.Name})); err != nil {
		return model.InternalError("database persistence failed: " + err.Error())
	}
	return nil
}

func (p *Processor) SubmitSharedConfigUpdate(ctx appctx.RequestContext, id string, req model.SubmitSharedConfigUpdateRequest) (model.SharedConfigUpdateRequest, *model.ErrorDetail) {
	current, ok := p.store.FindSharedConfig(strings.TrimSpace(id))
	if !ok {
		return model.SharedConfigUpdateRequest{}, model.NotFound(fmt.Sprintf("Shared config %q not found", id))
	}
	if !p.canReadSharedConfig(ctx, current) {
		return model.SharedConfigUpdateRequest{}, model.Forbidden("You cannot view this shared config")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return model.SharedConfigUpdateRequest{}, model.InvalidInput("reason is required")
	}
	proposed := current
	if strings.TrimSpace(req.Name) != "" {
		proposed.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		proposed.Description = strings.TrimSpace(req.Description)
	}
	if strings.TrimSpace(req.Format) != "" {
		format := strings.ToLower(strings.TrimSpace(req.Format))
		if !util.IsSupportedConfigFormat(format) {
			return model.SharedConfigUpdateRequest{}, model.InvalidInput("format must be properties, json, or yaml")
		}
		proposed.Format = format
	}
	if len(req.Entries) > 0 {
		entries, err := normalizeSharedConfigEntries(req.Entries)
		if err != nil {
			return model.SharedConfigUpdateRequest{}, err
		}
		proposed.Entries = entries
	}
	proposed.UpdatedBy = ctx.ActorName()
	proposed.UpdatedAt = time.Now().UTC()

	now := time.Now().UTC()
	request := model.SharedConfigUpdateRequest{
		ID:               util.NewID("scr"),
		SharedConfigID:   current.ID,
		SharedConfigName: current.Name,
		Scope:            current.Scope,
		ScopeID:          current.ScopeID,
		Requester:        ctx.ActorName(),
		Status:           "pending",
		Reason:           reason,
		ProposedConfig:   proposed,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := p.store.SaveSharedConfigUpdateRequest(request, newAudit(ctx.ActorName(), "shared_config_update.submit", "shared_config", current.ID, "", map[string]any{"requestId": request.ID, "reason": request.Reason})); err != nil {
		return model.SharedConfigUpdateRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return request, nil
}

func (p *Processor) withSharedConfigImpact(items []model.SharedConfig) []model.SharedConfig {
	projects := p.store.ListProjects()
	projectByID := map[string]model.Project{}
	for _, project := range projects {
		projectByID[project.ID] = project
	}
	for index := range items {
		prodCount := 0
		for _, projectID := range items[index].AffectedProjects {
			project, ok := projectByID[projectID]
			if !ok {
				continue
			}
			for _, environment := range project.Environments {
				if strings.EqualFold(environment.Name, "prod") {
					prodCount++
					break
				}
			}
		}
		items[index].ProdEnvironmentCount = prodCount
		if items[index].InheritedBy == 0 {
			items[index].InheritedBy = len(items[index].AffectedProjects)
		}
	}
	return items
}

func (p *Processor) notifySharedConfigConsumers(item model.SharedConfig, reason string) error {
	projects := p.store.ListProjects()
	affected := map[string]bool{}
	for _, projectID := range item.AffectedProjects {
		affected[projectID] = true
	}
	seenUsers := map[string]bool{}
	notifications := make([]model.Notification, 0)
	now := time.Now().UTC()
	for _, project := range projects {
		if !affected[project.ID] {
			continue
		}
		for _, member := range project.Members {
			if member.ID == "" || seenUsers[member.ID] {
				continue
			}
			seenUsers[member.ID] = true
			notifications = append(notifications, model.Notification{
				ID:        util.NewID("not"),
				UserID:    member.ID,
				Title:     "Shared config updated",
				Message:   fmt.Sprintf("%s was updated. Reason: %s", item.Name, reason),
				CreatedAt: now,
			})
		}
	}
	if len(notifications) == 0 {
		return nil
	}
	return p.store.SaveNotifications(notifications)
}

func (p *Processor) canReadSharedConfig(ctx appctx.RequestContext, item model.SharedConfig) bool {
	if item.Scope == model.ScopeGlobal || ctx.Actor.Role == model.RoleSystemAdmin {
		return true
	}
	group, ok := p.store.FindGroup(item.ScopeID)
	return ok && util.GroupHasMember(group, ctx.Actor.ID)
}

func normalizeSharedConfigEntries(raw []model.SharedConfigEntry) ([]model.SharedConfigEntry, *model.ErrorDetail) {
	if len(raw) == 0 {
		return nil, model.InvalidInput("entries are required")
	}
	seen := map[string]bool{}
	entries := make([]model.SharedConfigEntry, 0, len(raw))
	for _, entry := range raw {
		key := strings.TrimSpace(entry.Key)
		environment := strings.TrimSpace(entry.Environment)
		if key == "" {
			return nil, model.InvalidInput("entry key is required")
		}
		valueType := util.NormalizeValueType(entry.ValueType)
		if valueType == "" {
			return nil, model.InvalidInput("entry valueType must be string, number, boolean, or json")
		}
		identity := key + "\x00" + environment
		if seen[identity] {
			continue
		}
		seen[identity] = true
		entries = append(entries, model.SharedConfigEntry{Key: key, Value: entry.Value, ValueType: valueType, Environment: environment, IsSensitive: entry.IsSensitive})
	}
	return entries, nil
}
