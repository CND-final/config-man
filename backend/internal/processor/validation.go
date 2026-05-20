package processor

import (
	appctx "config-man/backend/internal/context"
	"sort"
	"strings"

	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ValidateProject(ctx appctx.RequestContext, projectID string, req model.ValidateProjectRequest) (model.ValidationResult, *model.ErrorDetail) {
	log := logger.Validation.With(
		"operation", "validation.project",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, strings.TrimSpace(req.Environment),
	)
	log.Info("project validation requested", "draft_entry_count", len(req.DraftEntries))

	project, err := p.requireProject(projectID)
	if err != nil {
		log.Warn("project validation failed", "error_kind", err.Kind, "error", err.Detail)
		return model.ValidationResult{}, err
	}

	targetEnvs := make([]string, 0)
	if strings.TrimSpace(req.Environment) != "" {
		if err := p.requireEnvironment(projectID, req.Environment); err != nil {
			log.Warn("project validation failed", "error_kind", err.Kind, "error", err.Detail)
			return model.ValidationResult{}, err
		}
		targetEnvs = append(targetEnvs, req.Environment)
	} else {
		envs := append([]model.ProjectEnvironment(nil), project.Environments...)
		sort.Slice(envs, func(i, j int) bool { return envs[i].SortOrder < envs[j].SortOrder })
		for _, env := range envs {
			targetEnvs = append(targetEnvs, env.Name)
		}
	}

	entries := p.store.ValidationEntries(projectID, targetEnvs)
	for _, draft := range req.DraftEntries {
		if util.Contains(targetEnvs, draft.Environment) {
			entries = append(entries, model.ValidationEntry{
				Environment: draft.Environment,
				Key:         draft.Key,
				Value:       draft.Value,
				ValueType:   util.Fallback(draft.ValueType, "string"),
				IsSensitive: draft.IsSensitive,
			})
		}
	}

	result := util.ValidateEntries(projectID, targetEnvs, entries)
	log.Info("project validation completed", "environment_count", len(targetEnvs), "entry_count", len(entries), "error_count", len(result.Errors), "warning_count", len(result.Warnings), "is_valid", result.Valid)
	return result, nil
}
