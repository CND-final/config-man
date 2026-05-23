package app

import (
	"config-man/backend/internal/response"
	"config-man/backend/model"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getPublicAuthRoutes() Routes {
	return Routes{
		{
			Name:    "Login",
			Method:  http.MethodPost,
			Pattern: "/auth/login",
			APIFunc: s.handleLogin,
		},
	}
}

func (s *Server) getProtectedAuthRoutes() Routes {
	return Routes{
		{
			Name:    "Me",
			Method:  http.MethodGet,
			Pattern: "/auth/me",
			APIFunc: s.handleMe,
		},
		{
			Name:    "ListUsers",
			Method:  http.MethodGet,
			Pattern: "/users",
			APIFunc: s.handleListUsers,
		},
	}
}

func (s *Server) handleLogin(c *gin.Context) {
	var req model.LoginRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}

	payload, err := s.processor.Login(req)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, payload)
}

func (s *Server) handleMe(c *gin.Context) {
	reqCtx := requestContextFromGin(c)
	response.WriteJSON(c, http.StatusOK, reqCtx.Actor)
}

func (s *Server) handleListUsers(c *gin.Context) {
	users, err := s.processor.ListUsers(requestContextFromGin(c))
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"users": users})
}
