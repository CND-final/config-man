package processor

import (
	appctx "config-man/backend/internal/context"
	"fmt"
	"strings"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ListReviewRequests(_ appctx.RequestContext, projectID string, filters model.ReviewFilters) ([]model.ReviewRequest, *model.ErrorDetail) {
	if projectID != "" {
		if _, err := p.requireProject(projectID); err != nil {
			return nil, err
		}
	}
	return p.store.ListReviewRequests(projectID, filters), nil
}

func (p *Processor) CreateReviewRequest(ctx appctx.RequestContext, req model.CreateReviewRequest) (model.ReviewRequest, *model.ErrorDetail) {
	if !util.CanCreateReview(ctx.Actor) {
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Environment) == "" || strings.TrimSpace(req.Reason) == "" {
		return model.ReviewRequest{}, model.InvalidInput("projectId, environment, and reason are required")
	}
	project, err := p.requireProject(req.ProjectID)
	if err != nil {
		return model.ReviewRequest{}, err
	}
	if err := p.requireEnvironment(req.ProjectID, req.Environment); err != nil {
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
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return request, nil
}

func (p *Processor) SetReviewStatus(ctx appctx.RequestContext, requestID, status, comment string) (model.ReviewRequest, *model.ErrorDetail) {
	if !util.CanReview(ctx.Actor) {
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	request, ok := p.store.FindReviewRequest(requestID)
	if !ok {
		return model.ReviewRequest{}, model.NotFound(fmt.Sprintf("Review request %q not found", requestID))
	}
	request.Status = status
	request.Reviewer = ctx.ActorName()
	request.Comment = strings.TrimSpace(comment)
	request.UpdatedAt = time.Now().UTC()

	audit := newAudit(ctx.ActorName(), "review_request."+status, "change_request", request.ID, request.ProjectID, map[string]any{
		"comment": request.Comment,
	})
	if err := p.store.SaveReviewRequest(request, audit); err != nil {
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}
	return request, nil
}
