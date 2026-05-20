package app

import (
	"config-man/backend/internal/response"
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
	reqCtx := requestContextFromGin(c)

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigs(reqCtx, c.Param("projectId"), c.Query("env"), reveal)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleCreateConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.CreateConfig(reqCtx, c.Param("projectId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, entry)
}

func (s *Server) handleImportConfigs(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.ImportConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	payload, appErr := s.processor.ImportConfigs(reqCtx, c.Param("projectId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, payload)
}

func (s *Server) handleUpdateConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.UpdateConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.UpdateConfig(reqCtx, c.Param("projectId"), c.Param("configId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, entry)
}

func (s *Server) handleDeleteConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	payload, appErr := s.processor.DeleteConfig(reqCtx, c.Param("projectId"), c.Param("configId"))
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}
