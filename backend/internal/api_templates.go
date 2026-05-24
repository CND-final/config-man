package app

import (
	"config-man/backend/internal/response"
	"config-man/backend/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getTemplateRoutes() Routes {
	return Routes{
		{
			Name:    "ListTemplates",
			Method:  http.MethodGet,
			Pattern: "/templates",
			APIFunc: s.handleListTemplates,
		},
		{
			Name:    "BaseTemplate",
			Method:  http.MethodGet,
			Pattern: "/templates/base",
			APIFunc: s.handleBaseTemplate,
		},
		{
			Name:    "CreateTemplate",
			Method:  http.MethodPost,
			Pattern: "/templates",
			APIFunc: s.handleCreateTemplate,
		},
		{
			Name:    "ListSharedConfigs",
			Method:  http.MethodGet,
			Pattern: "/shared-configs",
			APIFunc: s.handleListSharedConfigs,
		},
		{
			Name:    "CreateSharedConfig",
			Method:  http.MethodPost,
			Pattern: "/shared-configs",
			APIFunc: s.handleCreateSharedConfig,
		},
		{
			Name:    "UpdateSharedConfig",
			Method:  http.MethodPut,
			Pattern: "/shared-configs/:sharedConfigId",
			APIFunc: s.handleUpdateSharedConfig,
		},
		{
			Name:    "DeleteSharedConfig",
			Method:  http.MethodDelete,
			Pattern: "/shared-configs/:sharedConfigId",
			APIFunc: s.handleDeleteSharedConfig,
		},
		{
			Name:    "SubmitSharedConfigUpdate",
			Method:  http.MethodPost,
			Pattern: "/shared-configs/:sharedConfigId/submit-update",
			APIFunc: s.handleSubmitSharedConfigUpdate,
		},
	}
}

func (s *Server) handleListTemplates(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.Templates(reqCtx))
}

func (s *Server) handleBaseTemplate(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.BaseTemplate(reqCtx))
}

func (s *Server) handleCreateTemplate(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateTemplateRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	template, appErr := s.processor.CreateTemplate(reqCtx, req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, template)
}

func (s *Server) handleListSharedConfigs(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.SharedConfigs(reqCtx))
}

func (s *Server) handleCreateSharedConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.CreateSharedConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	item, appErr := s.processor.CreateGlobalSharedConfig(reqCtx, req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, item)
}

func (s *Server) handleUpdateSharedConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.UpdateSharedConfigRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	item, appErr := s.processor.UpdateGlobalSharedConfig(reqCtx, c.Param("sharedConfigId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusOK, item)
}

func (s *Server) handleDeleteSharedConfig(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	if appErr := s.processor.DeleteGlobalSharedConfig(reqCtx, c.Param("sharedConfigId")); appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleSubmitSharedConfigUpdate(c *gin.Context) {
	reqCtx := requestContextFromGin(c)

	var req model.SubmitSharedConfigUpdateRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	request, appErr := s.processor.SubmitSharedConfigUpdate(reqCtx, c.Param("sharedConfigId"), req)
	if appErr != nil {
		response.WriteError(c, appErr)
		return
	}
	response.WriteJSON(c, http.StatusCreated, request)
}
