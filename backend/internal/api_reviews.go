package app

import (
	"config-man/backend/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getReviewRoutes() Routes {
	return Routes{
		{
			Name:    "ListReviewRequests",
			Method:  http.MethodGet,
			Pattern: "/review-requests",
			APIFunc: s.handleListReviewRequests,
		},
		{
			Name:    "CreateReviewRequest",
			Method:  http.MethodPost,
			Pattern: "/review-requests",
			APIFunc: s.handleCreateReviewRequest,
		},
		{
			Name:    "ListProjectReviewRequests",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/review-requests",
			APIFunc: s.handleListProjectReviewRequests,
		},
		{
			Name:    "ApproveReviewRequest",
			Method:  http.MethodPut,
			Pattern: "/review-requests/:requestId/approve",
			APIFunc: s.handleApproveReviewRequest,
		},
		{
			Name:    "RejectReviewRequest",
			Method:  http.MethodPut,
			Pattern: "/review-requests/:requestId/reject",
			APIFunc: s.handleRejectReviewRequest,
		},
	}
}

func (s *Server) handleListReviewRequests(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	requests, appErr := s.processor.ListReviewRequests(reqCtx, "", reviewFiltersFromContext(c))
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, requests)
}

func (s *Server) handleListProjectReviewRequests(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	requests, appErr := s.processor.ListReviewRequests(reqCtx, c.Param("projectId"), reviewFiltersFromContext(c))
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, requests)
}

func (s *Server) handleCreateReviewRequest(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateReviewRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	request, appErr := s.processor.CreateReviewRequest(reqCtx, req)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusCreated, request)
}

func (s *Server) handleApproveReviewRequest(c *gin.Context) {
	s.handleReviewDecision(c, "approved")
}

func (s *Server) handleRejectReviewRequest(c *gin.Context) {
	s.handleReviewDecision(c, "rejected")
}

func (s *Server) handleReviewDecision(c *gin.Context, status string) {
	reqCtx := requestContextFromGin(c)

	var req model.ReviewDecisionRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	request, appErr := s.processor.SetReviewStatus(reqCtx, c.Param("requestId"), status, req.Comment)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, request)
}

func reviewFiltersFromContext(c *gin.Context) model.ReviewFilters {
	configKey := c.Query("key")
	if configKey == "" {
		configKey = c.Query("configKey")
	}
	environment := c.Query("env")
	if environment == "" {
		environment = c.Query("environment")
	}
	return model.ReviewFilters{
		Environment: environment,
		ConfigKey:   configKey,
		Status:      c.Query("status"),
	}
}
