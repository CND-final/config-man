package app

import (
	"config-man/backend/model"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) getConfigRoutes() Routes {
	return Routes{
		{
			Name:    "ListConfigs",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/configs",
			APIFunc: s.handleListConfigs,
		},
		{
			Name:    "CreateConfig",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs",
			APIFunc: s.handleCreateConfig,
		},
		{
			Name:    "ImportConfigs",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/import",
			APIFunc: s.handleImportConfigs,
		},
		{
			Name:    "UpdateConfig",
			Method:  http.MethodPut,
			Pattern: "/projects/:projectId/configs/:configId",
			APIFunc: s.handleUpdateConfig,
		},
		{
			Name:    "DeleteConfig",
			Method:  http.MethodDelete,
			Pattern: "/projects/:projectId/configs/:configId",
			APIFunc: s.handleDeleteConfig,
		},
	}
}

func (s *Server) handleListConfigs(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigs(user, c.Param("projectId"), c.Query("env"), reveal)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, payload)
}

func (s *Server) handleCreateConfig(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}

	var req model.CreateConfigRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	entry, appErr := s.processor.CreateConfig(user, c.Param("projectId"), req)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusCreated, entry)
}

func (s *Server) handleImportConfigs(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}

	var req model.ImportConfigRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	payload, appErr := s.processor.ImportConfigs(user, c.Param("projectId"), req)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusCreated, payload)
}

func (s *Server) handleUpdateConfig(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}

	var req model.UpdateConfigRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	entry, appErr := s.processor.UpdateConfig(user, c.Param("projectId"), c.Param("configId"), req)
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, entry)
}

func (s *Server) handleDeleteConfig(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}

	payload, appErr := s.processor.DeleteConfig(user, c.Param("projectId"), c.Param("configId"))
	if appErr != nil {
		writeError(c, appErr)
		return
	}
	writeJSON(c, http.StatusOK, payload)
}
