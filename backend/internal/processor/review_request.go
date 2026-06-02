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

func (p *Processor) ListReviewRequests(ctx appctx.RequestContext, projectID string, filters model.ReviewFilters) ([]model.ReviewRequest, *model.ErrorDetail) {
	log := logger.Review
	if projectID != "" {
		project, err := p.requireReadableProject(ctx, projectID)
		if err != nil {
			log.Warn("list review requests failed")
			return nil, err
		}
		if !util.CanCreateProjectReview(ctx.Actor, project.Members) {
			log.Info("review requests hidden for read-only member")
			return []model.ReviewRequest{}, nil
		}
	}

	requests := p.store.ListReviewRequests(projectID, filters)
	if projectID == "" && ctx.Actor.Role != model.RoleSystemAdmin {
		visibleProjectIDs := map[string]bool{}
		for _, project := range p.ListProjects(ctx) {
			if util.CanCreateProjectReview(ctx.Actor, project.Members) {
				visibleProjectIDs[project.ID] = true
			}
		}
		visible := make([]model.ReviewRequest, 0, len(requests))
		for _, request := range requests {
			if visibleProjectIDs[request.ProjectID] {
				visible = append(visible, request)
			}
		}
		requests = visible
	}
	log.Info("review requests listed")
	return requests, nil
}

func (p *Processor) CreateReviewRequest(ctx appctx.RequestContext, req model.CreateReviewRequest) (model.ReviewRequest, *model.ErrorDetail) {
	log := logger.Review
	log.Info("create review request requested")

	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Environment) == "" || strings.TrimSpace(req.Reason) == "" {
		log.Warn("create review request invalid")
		return model.ReviewRequest{}, model.InvalidInput("projectId, environment, and reason are required")
	}
	project, err := p.requireReadableProject(ctx, req.ProjectID)
	if err != nil {
		log.Warn("create review request failed")
		return model.ReviewRequest{}, err
	}
	if !util.CanCreateProjectReview(ctx.Actor, project.Members) {
		log.Warn("create review request denied")
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	if err := p.requireEnvironment(req.ProjectID, req.Environment); err != nil {
		log.Warn("create review request failed")
		return model.ReviewRequest{}, err
	}
	changes, changeErr := p.normalizeReviewChanges(project, req)
	if changeErr != nil {
		log.Warn("create review request invalid")
		return model.ReviewRequest{}, changeErr
	}

	now := time.Now().UTC()
	request := model.ReviewRequest{
		ID:              util.NewID("rev"),
		ProjectID:       req.ProjectID,
		ProjectName:     project.Name,
		Environment:     req.Environment,
		Branch:          util.NormalizeBranch(req.Branch),
		ConfigKey:       strings.TrimSpace(req.ConfigKey),
		Requester:       ctx.ActorName(),
		Status:          "pending",
		Reason:          strings.TrimSpace(req.Reason),
		ProposedChanges: changes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	audit := newAudit(ctx.ActorName(), "review_request.create", "change_request", request.ID, request.ProjectID, map[string]any{
		"environment": request.Environment,
		"configKey":   request.ConfigKey,
		"reason":      request.Reason,
	})
	if err := p.store.SaveReviewRequest(request, audit); err != nil {
		log.Error("create review request persistence failed")
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}
	_ = p.notifyProjectReviewers(project, request)

	log.Info("review request created")
	return request, nil
}

func (p *Processor) SetReviewStatus(ctx appctx.RequestContext, requestID, status, comment string) (model.ReviewRequest, *model.ErrorDetail) {
	log := logger.Review
	log.Info("review decision requested")

	request, ok := p.store.FindReviewRequest(requestID)
	if !ok {
		log.Warn("review request not found")
		return model.ReviewRequest{}, model.NotFound(fmt.Sprintf("Review request %q not found", requestID))
	}
	project, err := p.requireReadableProject(ctx, request.ProjectID)
	if err != nil {
		log.Warn("review decision denied")
		return model.ReviewRequest{}, err
	}
	if !util.CanReviewProject(ctx.Actor, project.Members) {
		log.Warn("review decision denied")
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	request.Status = status
	request.Reviewer = ctx.ActorName()
	request.Comment = strings.TrimSpace(comment)
	request.UpdatedAt = time.Now().UTC()

	audit := newAudit(ctx.ActorName(), "review_request."+status, "change_request", request.ID, request.ProjectID, map[string]any{
		"comment": request.Comment,
	})
	if status == "approved" {
		if appErr := p.applyApprovedReviewChanges(ctx, project, request); appErr != nil {
			return model.ReviewRequest{}, appErr
		}
	}
	if err := p.store.SaveReviewRequest(request, audit); err != nil {
		log.Error("review decision persistence failed")
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}
	if status == "approved" {
		_ = p.notifyProjectMembersAfterApproval(ctx, project, request)
	}

	log.Info("review request decided")
	return request, nil
}

func (p *Processor) normalizeReviewChanges(project model.Project, req model.CreateReviewRequest) ([]model.ReviewConfigChange, *model.ErrorDetail) {
	if len(req.ProposedChanges) == 0 {
		return nil, nil
	}
	changes := make([]model.ReviewConfigChange, 0, len(req.ProposedChanges))
	for _, change := range req.ProposedChanges {
		environment := strings.TrimSpace(change.Environment)
		if environment == "" {
			environment = strings.TrimSpace(req.Environment)
		}
		if environment != req.Environment {
			return nil, model.InvalidInput("proposedChanges environment must match review environment")
		}
		key := strings.TrimSpace(change.Key)
		if key == "" {
			return nil, model.InvalidInput("proposedChanges key is required")
		}
		valueType := util.NormalizeValueType(change.ValueType)
		if valueType == "" {
			return nil, model.InvalidInput("proposedChanges valueType must be string, number, boolean, json, or yaml")
		}
		branch := util.NormalizeBranch(change.Branch)
		if strings.TrimSpace(change.Branch) == "" {
			branch = util.NormalizeBranch(req.Branch)
		}
		configID := strings.TrimSpace(change.ConfigID)
		if configID == "" {
			return nil, model.InvalidInput("proposedChanges configId is required")
		}
		if _, ok := p.store.FindConfig(project.ID, configID); !ok {
			return nil, model.NotFound(fmt.Sprintf("Config %q not found", configID))
		}
		changes = append(changes, model.ReviewConfigChange{
			ConfigEntryID: strings.TrimSpace(change.ConfigEntryID),
			ConfigID:      configID,
			Key:           key,
			OldValue:      change.OldValue,
			Value:         change.Value,
			ValueType:     valueType,
			IsSensitive:   change.IsSensitive,
			Environment:   environment,
			Branch:        branch,
		})
	}
	return changes, nil
}

func (p *Processor) applyApprovedReviewChanges(ctx appctx.RequestContext, project model.Project, request model.ReviewRequest) *model.ErrorDetail {
	if len(request.ProposedChanges) == 0 {
		return nil
	}
	now := time.Now().UTC()
	entries := make([]model.ConfigEntry, 0, len(request.ProposedChanges))
	versions := make([]model.ConfigVersion, 0, len(request.ProposedChanges))
	for _, change := range request.ProposedChanges {
		entry, ok := model.ConfigEntry{}, false
		if change.ConfigEntryID != "" && !strings.HasPrefix(change.ConfigEntryID, "draft-") {
			entry, ok = p.store.FindConfigEntry(project.ID, change.ConfigEntryID)
		}
		if !ok {
			entry, ok = p.store.FindConfigEntryByKey(project.ID, change.Environment, util.NormalizeBranch(change.Branch), change.Key)
		}
		oldValue := ""
		if ok {
			oldValue = entry.Value
		} else {
			entry = model.ConfigEntry{
				ID:          util.NewID("cfg"),
				ProjectID:   project.ID,
				Environment: change.Environment,
				CreatedAt:   now,
			}
		}
		entry.ConfigID = change.ConfigID
		entry.Key = change.Key
		entry.Value = change.Value
		entry.ValueType = change.ValueType
		entry.IsSensitive = change.IsSensitive
		entry.UpdatedBy = ctx.ActorName()
		entry.UpdatedAt = now
		entries = append(entries, entry)
		var oldValuePtr *string
		if ok {
			oldValuePtr = &oldValue
		}
		versions = append(versions, newVersion(entry.ID, oldValuePtr, entry.Value, ctx.ActorName(), request.Reason))
	}
	audit := newAudit(ctx.ActorName(), "review_request.apply", "change_request", request.ID, project.ID, map[string]any{"changeCount": len(entries)})
	if err := p.store.SaveConfigEntries(entries, versions, audit); err != nil {
		return model.InternalError("database persistence failed: " + err.Error())
	}
	return nil
}

func (p *Processor) notifyProjectReviewers(project model.Project, request model.ReviewRequest) error {
	now := time.Now().UTC()
	notifications := make([]model.Notification, 0)
	for _, member := range project.Members {
		if member.ProjectRole != model.RoleProjectMemberAdmin && member.ProjectRole != model.RoleProjectReviewer {
			continue
		}
		if member.Name == request.Requester || member.ID == "" {
			continue
		}
		notifications = append(notifications, model.Notification{
			ID:        util.NewID("not"),
			UserID:    member.ID,
			Title:     "Review requested",
			Message:   fmt.Sprintf("%s requested a prod config review for %s", request.Requester, project.Name),
			CreatedAt: now,
		})
	}
	if len(notifications) == 0 {
		return nil
	}
	return p.store.SaveNotifications(notifications)
}

func (p *Processor) notifyProjectMembersAfterApproval(ctx appctx.RequestContext, project model.Project, request model.ReviewRequest) error {
	now := time.Now().UTC()
	notifications := make([]model.Notification, 0, len(project.Members))
	for _, member := range project.Members {
		if member.ID == "" || member.ID == ctx.Actor.ID {
			continue
		}
		notifications = append(notifications, model.Notification{
			ID:        util.NewID("not"),
			UserID:    member.ID,
			Title:     "Prod config updated",
			Message:   fmt.Sprintf("%s prod config was updated after review approval", project.Name),
			CreatedAt: now,
		})
	}
	if len(notifications) == 0 {
		return nil
	}
	return p.store.SaveNotifications(notifications)
}
