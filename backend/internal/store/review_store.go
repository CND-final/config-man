package store

import (
	"context"

	"config-man/backend/model"
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
