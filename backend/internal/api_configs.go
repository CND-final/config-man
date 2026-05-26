package app

import (
	"config-man/backend/internal/response"
	"config-man/backend/model"
	"encoding/json"
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
			Name:    "ListConfigEntries",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/config-entries",
			APIFunc: s.handleListConfigEntries,
		},
		{
			Name:    "CreateConfigEntry",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/config-entries",
			APIFunc: s.handleCreateConfigEntry,
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
			Name:    "ExtractConfigEntries",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/config-entries/extract",
			APIFunc: s.handleExtractConfigs,
		},
		{
			Name:    "ImportConfigEntries",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/config-entries/import",
			APIFunc: s.handleImportConfigs,
		},
		{
			Name:    "UpdateConfigEntry",
			Method:  http.MethodPut,
			Pattern: "/projects/:projectId/config-entries/:configId",
			APIFunc: s.handleUpdateConfigEntry,
		},
		{
			Name:    "ListConfigEntryVersions",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/config-entries/:configId/versions",
			APIFunc: s.handleListConfigEntryVersions,
		},
		{
			Name:    "RollbackConfigEntry",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/config-entries/:configId/rollback",
			APIFunc: s.handleRollbackConfigEntry,
		},
		{
			Name:    "DeleteConfigEntry",
			Method:  http.MethodDelete,
			Pattern: "/projects/:projectId/config-entries/:configId",
			APIFunc: s.handleDeleteConfigEntry,
		},
		{
			Name:    "LegacyExtractConfigs",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/extract",
			APIFunc: s.handleExtractConfigs,
		},
		{
			Name:    "LegacyImportConfigs",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/import",
			APIFunc: s.handleImportConfigs,
		},
		{
			Name:    "LegacyUpdateConfigEntry",
			Method:  http.MethodPut,
			Pattern: "/projects/:projectId/configs/:configId",
			APIFunc: s.handleUpdateConfigEntry,
		},
		{
			Name:    "LegacyListConfigEntryVersions",
			Method:  http.MethodGet,
			Pattern: "/projects/:projectId/configs/:configId/versions",
			APIFunc: s.handleListConfigEntryVersions,
		},
		{
			Name:    "LegacyRollbackConfigEntry",
			Method:  http.MethodPost,
			Pattern: "/projects/:projectId/configs/:configId/rollback",
			APIFunc: s.handleRollbackConfigEntry,
		},
		{
			Name:    "LegacyDeleteConfigEntry",
			Method:  http.MethodDelete,
			Pattern: "/projects/:projectId/configs/:configId",
			APIFunc: s.handleDeleteConfigEntry,
		},
	}
}

func (s *Server) handleListConfigs(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	if strings.TrimSpace(c.Query("env")) != "" {
		reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
		payload, appErr := s.processor.ListConfigEntries(reqCtx, c.Param("projectId"), c.Query("env"), reveal)
		if appErr != nil {
			response.WriteError(c, appErr)
			return
		}
		response.WriteJSON(c, http.StatusOK, payload)
		return
	}

	payload, appErr := s.processor.ListConfigs(reqCtx, c.Param("projectId"))
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleCreateConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	raw, readErr := c.GetRawData()
	if readErr != nil {
		response.WriteError(c, model.InvalidInput("Invalid JSON body: "+readErr.Error()))
		return
	}
	var configReq model.CreateConfigRequest
	if err := json.Unmarshal(raw, &configReq); err != nil {
		response.WriteError(c, model.InvalidInput("Invalid JSON body: "+err.Error()))
		return
	}
	if strings.TrimSpace(configReq.Name) == "" {
		var entryReq model.CreateConfigEntryRequest
		if err := json.Unmarshal(raw, &entryReq); err != nil {
			response.WriteError(c, model.InvalidInput("Invalid JSON body: "+err.Error()))
			return
		}
		entry, appErr := s.processor.CreateConfigEntry(reqCtx, c.Param("projectId"), entryReq)
		if appErr != nil {
			response.WriteError(c, appErr)
			return
		}
		response.WriteJSON(c, http.StatusCreated, entry)
		return
	}

	config, appErr := s.processor.CreateConfig(reqCtx, c.Param("projectId"), configReq)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, config)
}

func (s *Server) handleListConfigEntries(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigEntries(reqCtx, c.Param("projectId"), c.Query("env"), reveal)
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

func (s *Server) handleCreateConfigEntry(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateConfigEntryRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.CreateConfigEntry(reqCtx, c.Param("projectId"), req)
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

func (s *Server) handleUpdateConfigEntry(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.UpdateConfigEntryRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.UpdateConfigEntry(reqCtx, c.Param("projectId"), c.Param("configId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, entry)
}

func (s *Server) handleListConfigEntryVersions(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	reveal := strings.EqualFold(c.Query("revealSensitive"), "true")
	payload, appErr := s.processor.ListConfigEntryVersions(reqCtx, c.Param("projectId"), c.Param("configId"), reveal)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleRollbackConfigEntry(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.RollbackConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	entry, appErr := s.processor.RollbackConfigEntry(reqCtx, c.Param("projectId"), c.Param("configId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, entry)
}

func (s *Server) handleDeleteConfigEntry(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	payload, appErr := s.processor.DeleteConfigEntry(reqCtx, c.Param("projectId"), c.Param("configId"))
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}
