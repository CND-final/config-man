package app

import (
	"config-man/backend/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getAuthRoutes() Routes {
	return Routes{
		{
			Name:    "Login",
			Method:  http.MethodPost,
			Pattern: "/auth/login",
			APIFunc: s.handleLogin,
		},
		{
			Name:    "Me",
			Method:  http.MethodGet,
			Pattern: "/auth/me",
			APIFunc: s.handleMe,
		},
	}
}

func (s *Server) handleLogin(c *gin.Context) {
	var req model.LoginRequest
	if err := decodeJSON(c, &req); err != nil {
		writeError(c, err)
		return
	}

	payload, err := s.processor.Login(req)
	if err != nil {
		writeError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, payload)
}

func (s *Server) handleMe(c *gin.Context) {
	user, err := s.processor.RequireUser(c.Request)
	if err != nil {
		writeError(c, err)
		return
	}
	writeJSON(c, http.StatusOK, user)
}
