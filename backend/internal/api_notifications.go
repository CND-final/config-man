package app

import (
	"net/http"

	"config-man/backend/internal/response"

	"github.com/gin-gonic/gin"
)

func (s *Server) getNotificationRoutes() Routes {
	return Routes{
		{
			Name:    "ListNotifications",
			Method:  http.MethodGet,
			Pattern: "/notifications",
			APIFunc: s.handleListNotifications,
		},
	}
}

func (s *Server) handleListNotifications(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, s.processor.Notifications(reqCtx))
}
