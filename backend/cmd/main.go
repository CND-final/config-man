package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	app "config-man/backend/internal"
	"config-man/backend/internal/logger"
	"config-man/backend/internal/processor"
	"config-man/backend/pkg/config"
)

func main() {
	cfg := config.NewConfig()
	log := logger.MainLog

	traceCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proc, err := newProcessor(traceCtx, cfg)
	if err != nil {
		log.Error("initialize processor failed", slog.Any("error", err))
		os.Exit(1)
	}

	server, err := app.NewServer(proc, logger.APILog, cfg)
	if err != nil {
		log.Error("initialize config-man server failed", slog.Any("error", err))
		os.Exit(1)
	}

	var wg sync.WaitGroup
	if err := server.Run(traceCtx, &wg); err != nil {
		log.Error("run config-man server failed", slog.Any("error", err))
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	cancel()
	server.Stop()
	wg.Wait()
}

func newProcessor(ctx context.Context, cfg config.Config) (*processor.Processor, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required for backend startup")
	}

	logger.DBLog.Info("DATABASE_URL detected; using PostgreSQL store")
	return processor.NewWithDatabase(ctx, logger.ProcessorLog, cfg.DatabaseURL)
}
