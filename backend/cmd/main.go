package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	app "config-man/backend/internal"
	appctx "config-man/backend/internal/context"
	"config-man/backend/internal/logger"
	"config-man/backend/internal/processor"
	"config-man/backend/pkg/config"
)

func main() {
	cfg := config.NewConfig()
	log := logger.Main

	traceCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtime, err := appctx.NewConfigManContext(traceCtx, cfg)
	if err != nil {
		log.Error("initialize config-man context failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			log.Error("close config-man context failed", slog.Any("error", err))
		}
	}()

	proc, err := processor.NewProcessor(runtime.Store)
	if err != nil {
		log.Error("initialize config-man processor failed", slog.Any("error", err))
		os.Exit(1)
	}
	server, err := app.NewServer(proc, logger.API, cfg)
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
