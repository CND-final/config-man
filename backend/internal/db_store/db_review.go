package dbstore

import (
	"context"
	"database/sql"
	"encoding/json"

	"config-man/backend/model"
)

func (s *Store) SaveReviewRequest(ctx context.Context, request model.ReviewRequest, audit model.AuditLog) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		if err := upsertReviewRequestTx(ctx, tx, request); err != nil {
			return err
		}
		return insertAuditTx(ctx, tx, audit)
	})
}

func upsertReviewRequestTx(ctx context.Context, tx *sql.Tx, request model.ReviewRequest) error {
	proposed, err := json.Marshal(request.ProposedChanges)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO review_requests (id, project_id, project_name, environment, config_key, requester, reviewer, status, reason, comment, proposed_changes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET project_id = EXCLUDED.project_id, project_name = EXCLUDED.project_name, environment = EXCLUDED.environment, config_key = EXCLUDED.config_key, requester = EXCLUDED.requester, reviewer = EXCLUDED.reviewer, status = EXCLUDED.status, reason = EXCLUDED.reason, comment = EXCLUDED.comment, proposed_changes = EXCLUDED.proposed_changes, updated_at = EXCLUDED.updated_at`, request.ID, request.ProjectID, request.ProjectName, request.Environment, request.ConfigKey, request.Requester, request.Reviewer, request.Status, request.Reason, request.Comment, proposed, request.CreatedAt, request.UpdatedAt)
	return err
}

func (s *Store) ListReviewRequests(ctx context.Context, projectID string, filters model.ReviewFilters) ([]model.ReviewRequest, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, project_id, project_name, environment, config_key, requester, reviewer, status, reason, comment, proposed_changes, created_at, updated_at FROM review_requests
		WHERE ($1 = '' OR project_id = $1)
		AND ($2 = '' OR environment = $2)
		AND ($3 = '' OR config_key = $3)
		AND ($4 = '' OR status = $4)
		ORDER BY created_at DESC`, projectID, filters.Environment, filters.ConfigKey, filters.Status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]model.ReviewRequest, 0)
	for rows.Next() {
		request := model.ReviewRequest{}
		var proposed []byte
		if err := rows.Scan(&request.ID, &request.ProjectID, &request.ProjectName, &request.Environment, &request.Branch, &request.ConfigKey, &request.Requester, &request.Reviewer, &request.Status, &request.Reason, &request.Comment, &proposed, &request.CreatedAt, &request.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(proposed, &request.ProposedChanges)
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (s *Store) FindReviewRequest(ctx context.Context, requestID string) (model.ReviewRequest, bool, error) {
	request := model.ReviewRequest{}
	var proposed []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, project_id, project_name, environment, branch, config_key, requester, reviewer, status, reason, comment, proposed_changes, created_at, updated_at FROM review_requests WHERE id = $1`, requestID).Scan(&request.ID, &request.ProjectID, &request.ProjectName, &request.Environment, &request.Branch, &request.ConfigKey, &request.Requester, &request.Reviewer, &request.Status, &request.Reason, &request.Comment, &proposed, &request.CreatedAt, &request.UpdatedAt)
	if err == sql.ErrNoRows {
		return model.ReviewRequest{}, false, nil
	}
	if err != nil {
		return model.ReviewRequest{}, false, err
	}
	_ = json.Unmarshal(proposed, &request.ProposedChanges)
	return request, true, nil
}
