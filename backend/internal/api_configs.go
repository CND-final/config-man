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
			Name:    "ListConfigHistory",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/config-history",
			APIFunc: s.handleListConfigHistory,
		},
		{
			Name:    "RollbackConfigHistory",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/config-history/rollback",
			APIFunc: s.handleRollbackConfigHistory,
		},
		{
			Name:    "CreateConfig",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs",
			APIFunc: s.handleCreateConfig,
		},
		{
			Name:    "ExtractConfigs",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/extract",
			APIFunc: s.handleExtractConfigs,
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
			Name:    "ListConfigVersions",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/configs/:configId/versions",
			APIFunc: s.handleListConfigVersions,
		},
		{
			Name:    "RollbackConfig",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/:configId/rollback",
			APIFunc: s.handleRollbackConfig,
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

func (s *Server) handleListConfigHistory(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigHistory(reqCtx, c.Param("projectId"), c.Query("env"), reveal)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleRollbackConfigHistory(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.RollbackConfigRevisionRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	payload, appErr := s.processor.RollbackConfigRevision(reqCtx, c.Param("projectId"), req)
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

func (s *Server) handleExtractConfigs(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.ImportConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	payload, appErr := s.processor.ExtractConfigs(reqCtx, c.Param("projectId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
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

func (s *Server) handleListConfigVersions(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigVersions(reqCtx, c.Param("projectId"), c.Param("configId"), reveal)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleRollbackConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.RollbackConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.RollbackConfig(reqCtx, c.Param("projectId"), c.Param("configId"), req)
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
