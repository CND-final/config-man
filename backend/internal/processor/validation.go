package processor

import (
	appctx "config-man/backend/internal/context"
	"sort"
	"strings"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ValidateProject(_ appctx.RequestContext, projectID string, req model.ValidateProjectRequest) (model.ValidationResult, *model.ErrorDetail) {
	project, err := p.requireProject(projectID)
	if err != nil {
		return model.ValidationResult{}, err
	}

	targetEnvs := make([]string, 0)
	if strings.TrimSpace(req.Environment) != "" {
		if err := p.requireEnvironment(projectID, req.Environment); err != nil {
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

	return util.ValidateEntries(projectID, targetEnvs, entries), nil
}
