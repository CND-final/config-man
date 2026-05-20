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
	log := logger.Review.With(
		"operation", "review_request.list",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
	).With(
		logger.FieldProjectID, projectID,
		logger.FieldEnvironment, filters.Environment,
		logger.FieldConfigKey, filters.ConfigKey,
		"status", filters.Status,
	)
	if projectID != "" {
		if _, err := p.requireProject(projectID); err != nil {
			log.Warn("list review requests failed", "error_kind", err.Kind, "error", err.Detail)
			return nil, err
		}
	}

	requests := p.store.ListReviewRequests(projectID, filters)
	log.Info("review requests listed", "request_count", len(requests))
	return requests, nil
}

func (p *Processor) CreateReviewRequest(ctx appctx.RequestContext, req model.CreateReviewRequest) (model.ReviewRequest, *model.ErrorDetail) {
	log := logger.Review.With(
		"operation", "review_request.create",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
	).With(
		logger.FieldProjectID, strings.TrimSpace(req.ProjectID),
		logger.FieldEnvironment, strings.TrimSpace(req.Environment),
		logger.FieldConfigKey, strings.TrimSpace(req.ConfigKey),
	)
	log.Info("create review request requested")

	if !util.CanCreateReview(ctx.Actor) {
		log.Warn("create review request denied", "reason", "role_not_allowed")
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	if strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Environment) == "" || strings.TrimSpace(req.Reason) == "" {
		log.Warn("create review request invalid", "reason", "missing_required_fields")
		return model.ReviewRequest{}, model.InvalidInput("projectId, environment, and reason are required")
	}
	project, err := p.requireProject(req.ProjectID)
	if err != nil {
		log.Warn("create review request failed", "error_kind", err.Kind, "error", err.Detail)
		return model.ReviewRequest{}, err
	}
	if err := p.requireEnvironment(req.ProjectID, req.Environment); err != nil {
		log.Warn("create review request failed", "error_kind", err.Kind, "error", err.Detail)
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
		log.Error("create review request persistence failed", "error", err)
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}

	log.Info("review request created", logger.FieldReviewRequestID, request.ID, "status", request.Status)
	return request, nil
}

func (p *Processor) SetReviewStatus(ctx appctx.RequestContext, requestID, status, comment string) (model.ReviewRequest, *model.ErrorDetail) {
	log := logger.Review.With(
		"operation", "review_request.decide",
		logger.FieldUserID, ctx.Actor.ID,
		logger.FieldActor, ctx.ActorName(),
		logger.FieldRole, string(ctx.Actor.Role),
		logger.FieldReviewRequestID, requestID,
		"status", status,
	)
	log.Info("review decision requested")

	if !util.CanReview(ctx.Actor) {
		log.Warn("review decision denied", "reason", "role_not_allowed")
		return model.ReviewRequest{}, model.Forbidden(fmt.Sprintf("Role %q is not allowed", ctx.Actor.Role))
	}
	request, ok := p.store.FindReviewRequest(requestID)
	if !ok {
		log.Warn("review request not found")
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
		log.Error("review decision persistence failed", "error", err)
		return model.ReviewRequest{}, model.InternalError("database persistence failed: " + err.Error())
	}

	log.Info("review request decided", logger.FieldProjectID, request.ProjectID, logger.FieldEnvironment, request.Environment)
	return request, nil
}
