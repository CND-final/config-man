package app

import (
	"config-man/backend/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getTemplateRoutes() Routes {
	return Routes{
		{
			Name:    "BaseTemplate",
			Method:  http.MethodGet,
			Pattern: "/templates/base",
			APIFunc: s.handleBaseTemplate,
		},
	}
}

func (s *Server) handleBaseTemplate(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.BaseTemplate(reqCtx))
}
