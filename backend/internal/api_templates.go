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
