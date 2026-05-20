package app

import (
	"config-man/backend/internal/response"
	"config-man/backend/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getValidationRoutes() Routes {
	return Routes{
		{
			Name:    "ValidateProject",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/validate",
			APIFunc: s.handleValidateProject,
		},
	}
}

func (s *Server) handleValidateProject(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.ValidateProjectRequest
	if err := response.DecodeOptionalJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	result, appErr := s.processor.ValidateProject(reqCtx, c.Param("projectId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, result)
}
