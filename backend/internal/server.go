package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"config-man/backend/internal/processor"
	"config-man/backend/pkg/config"

	"github.com/gin-gonic/gin"
)

const defaultShutdownTimeout = 2 * time.Second

type Server struct {
	processor  *processor.Processor
	log        *slog.Logger
	cfg        config.Config
	httpServer *http.Server
	router     *gin.Engine
}

func NewServer(proc *processor.Processor, log *slog.Logger, cfg config.Config) (*Server, error) {
	s := &Server{
		processor: proc,
		log:       log,
		cfg:       cfg,
	}

	s.router = newRouter(s)
	s.httpServer = &http.Server{
		Addr:              cfg.Addr(),
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return s, nil
}

func newRouter(s *Server) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery(), corsMiddleware())
	s.router = router

	apiGroup := router.Group("/api/v1")
	applyRoutes(apiGroup, s.getHealthRoutes())
	applyRoutes(apiGroup, s.getPublicAuthRoutes())

	protectedGroup := apiGroup.Group("")
	protectedGroup.Use(s.authMiddleware())
	applyRoutes(protectedGroup, s.getProtectedAuthRoutes())
	applyRoutes(protectedGroup, s.getTemplateRoutes())
	applyRoutes(protectedGroup, s.getProjectRoutes())
	applyRoutes(protectedGroup, s.getGroupsRoutes())
	applyRoutes(protectedGroup, s.getConfigRoutes())
	applyRoutes(protectedGroup, s.getReviewRoutes())

	return router
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) Run(traceCtx context.Context, wg *sync.WaitGroup) error {
	if s.httpServer == nil {
		return fmt.Errorf("HTTP server is not initialized")
	}

	wg.Add(1)
	go s.startServer(wg)
	return nil
}

func (s *Server) Stop() {
	if s.httpServer == nil {
		return
	}

	s.log.Info("stop config-man server", slog.String("addr", s.httpServer.Addr))
	ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Error("could not close config-man server", slog.Any("error", err))
	}
}

func (s *Server) startServer(wg *sync.WaitGroup) {
	defer func() {
		if p := recover(); p != nil {
			s.log.Error("panic in config-man server", slog.Any("panic", p), slog.String("stack", string(debug.Stack())))
		}
		wg.Done()
	}()

	s.log.Info("start config-man server", slog.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Error("config-man server error", slog.Any("error", err))
	}
	s.log.Info("config-man server stopped", slog.String("addr", s.httpServer.Addr))
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Actor")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
