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

	now := time.Now().UTC()
	request := model.ReviewRequest{
		ID:          util.NewID("rev"),
		ProjectID:   req.ProjectID,
		ProjectName: project.Name,
		Environment: req.Environment,
		ConfigKey:   strings.TrimSpace(req.ConfigKey),
		Requester:   ctx.ActorName(),
		Status:      "pending",
		Reason:      strings.TrimSpace(req.Reason),
		CreatedAt:   now,
		UpdatedAt:   now,
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
	if err := p.store.SaveReviewRequest(request, audit); err != nil {
		log.Error("review decision persistence failed")
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}

	log.Info("review request decided")
	return request, nil
}
