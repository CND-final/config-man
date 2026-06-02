package store

import (
	"context"
	"time"

	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (s *Store) ListReviewRequests(projectID string, filters model.ReviewFilters) []model.ReviewRequest {
	requests, err := s.db.ListReviewRequests(context.Background(), projectID, filters)
	if err != nil {
		return nil
	}
	return requests
}

func (s *Store) FindReviewRequest(requestID string) (model.ReviewRequest, bool) {
	request, ok, err := s.db.FindReviewRequest(context.Background(), requestID)
	if err != nil {
		return model.ReviewRequest{}, false
	}
	return request, ok
}

func (s *Store) SaveReviewRequest(request model.ReviewRequest, audit model.AuditLog) error {
	return s.db.SaveReviewRequest(context.Background(), request, audit)
}

func (s *Store) ensureDefaultReviewRequests() {
	if _, ok := s.FindReviewRequest("seed-pending-shared-runtime-review"); ok {
		return
	}
	project, ok := s.FindProject("customer-portal")
	if !ok {
		return
	}
	runtimeConfig, ok := s.FindConfig(project.ID, model.ConfigID(project.ID, "runtime-defaults.yaml"))
	if !ok {
		return
	}
	now := time.Now().UTC()
	previousLoggingLevel := "WARN"
	request := model.ReviewRequest{
		ID:          "seed-pending-shared-runtime-review",
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Environment: "prod",
		Branch:      "default",
		ConfigKey:   "logging.level.root",
		Requester:   "Nora Chen",
		Status:      "pending",
		Reason:      "Tune local prod logging override for linked shared runtime config",
		ProposedChanges: []model.ReviewConfigChange{
			{
				ConfigID:    runtimeConfig.ID,
				Key:         "logging.level.root",
				OldValue:    &previousLoggingLevel,
				Value:       "ERROR",
				ValueType:   "string",
				IsSensitive: false,
				Environment: "prod",
				Branch:      "default",
			},
		},
		CreatedAt: now.Add(-12 * time.Minute),
		UpdatedAt: now.Add(-12 * time.Minute),
	}
	_ = s.SaveReviewRequest(request, model.AuditLog{
		ID:           util.NewID("aud"),
		Actor:        "system",
		Action:       "review_request.seed",
		ResourceType: "change_request",
		ResourceID:   request.ID,
		ProjectID:    request.ProjectID,
		Metadata:     map[string]any{"configKey": request.ConfigKey, "environment": request.Environment},
		CreatedAt:    now,
	})
}
