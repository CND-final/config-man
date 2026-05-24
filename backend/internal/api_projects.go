package app

import (
	"config-man/backend/internal/response"
	"config-man/backend/model"
	"net/http"

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
		{
			Name:    "ListProjectMembers",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/members",
			APIFunc: s.handleListProjectMembers,
		},
		{
			Name:    "UpdateProjectMembers",
			Method:  http.MethodPut,
			Pattern: "/projects/:projectId/members",
			APIFunc: s.handleUpdateProjectMembers,
		},
	}
}

func (s *Server) handleListProjects(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.ListProjects(reqCtx))
}

func (s *Server) handleCreateProject(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateProjectRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	project, appErr := s.processor.CreateProject(reqCtx, req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, project)
}

func (s *Server) handleGetProject(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	project, appErr := s.processor.GetProject(reqCtx, c.Param("projectId"))
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, project)
}

func (s *Server) handleListProjectMembers(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	members, appErr := s.processor.ListProjectMembers(reqCtx, c.Param("projectId"))
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"members": members})
}

func (s *Server) handleUpdateProjectMembers(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.UpdateProjectMembersRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	members, appErr := s.processor.UpdateProjectMembers(reqCtx, c.Param("projectId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"members": members})
}
