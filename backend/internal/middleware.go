package app

import (
	"net/http"
	"strings"

	appctx "config-man/backend/internal/context"

	"github.com/gin-gonic/gin"
)

const requestContextKey = "requestContext"

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx, err := s.processor.AuthenticateToken(extractToken(c.Request))
		if err != nil {
			writeError(c, err)
			c.Abort()
			return
		}
		c.Set(requestContextKey, reqCtx)
		c.Next()
	}
}

func extractToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-Actor"))
	if token != "" {
		return token
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		return strings.TrimSpace(authorization[7:])
	}
	return authorization
}

func requestContextFromGin(c *gin.Context) appctx.RequestContext {
	value, ok := c.Get(requestContextKey)
	if !ok {
		return appctx.RequestContext{}
	}
	reqCtx, ok := value.(appctx.RequestContext)
	if !ok {
		return appctx.RequestContext{}
	}
	return reqCtx
}
