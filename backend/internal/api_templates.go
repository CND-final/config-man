package app

import (
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
	if _, err := s.processor.RequireUser(c.Request); err != nil {
		writeError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, s.processor.BaseTemplate())
}
