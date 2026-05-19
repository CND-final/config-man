package app

import (
	"config-man/backend/model"
	"net/http"

	"config-man/backend/internal/processor"

	"github.com/gin-gonic/gin"
)

func (s *Server) getProjectRoutes() Routes {
	return Routes{
		{
			Name:    "ListProjects",
			Method:  http.MethodGet,
			Pattern: "/projects",
			APIFunc: s.handleListProjects,
		},
		{
			Name:    "CreateProject",
			Method:  http.MethodPost,
			Pattern: "/projects",
			APIFunc: s.handleCreateProject,
		},
		{
			Name:    "GetProject",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId",
			APIFunc: s.handleGetProject,
		},
	}
}

func (s *Server) handleListProjects(c *gin.Context) {
	if _, err := s.processor.RequireUser(c.Request); err != nil {
		writeError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, s.processor.ListProjects())
}

func (s *Server) handleCreateProject(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}
	if user.Role != model.RoleSystemAdmin && user.Role != model.RoleProjectAdmin {
		writeError(c, &processor.AppError{Status: http.StatusForbidden, Message: "Only system_admin or project_admin can register projects"})
		return
	}

	var req model.CreateProjectRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	project, appErr := s.processor.CreateProject(req, user.Name)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusCreated, project)
}

func (s *Server) handleGetProject(c *gin.Context) {
	if _, err := s.processor.RequireUser(c.Request); err != nil {
		writeError(c, err)
		return
	}
	project, appErr := s.processor.GetProject(c.Param("projectId"))
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, project)
}
