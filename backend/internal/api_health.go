package app

import (
	"config-man/backend/internal/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getHealthRoutes() Routes {
	return Routes{
		{
			Name:    "Health",
			Method:  http.MethodGet,
			Pattern: "/health",
			APIFunc: s.handleHealth,
		},
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	response.WriteJSON(c, http.StatusOK, gin.H{"status": "ok", "service": "config-man-go"})
}
