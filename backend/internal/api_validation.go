package app

import (
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
	if _, err := s.processor.RequireUser(c.Request); err != nil {
		writeError(c, err)
		return
	}

	var req model.ValidateProjectRequest
	if err := decodeOptionalJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	result, appErr := s.processor.ValidateProject(c.Param("projectId"), req)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, result)
}
