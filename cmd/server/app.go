package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"goflow/internal/config"
	"goflow/internal/mq"
	"goflow/internal/pkg/logger"

	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg        *config.Config
	httpServer *http.Server
	mqRouter   *mq.Router
	redis      *redis.Client
}

func NewApp(cfg *config.Config, handler http.Handler, mqRouter *mq.Router, redisClient *redis.Client) *App {
	return &App{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
			Handler:      handler,
			ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
			WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		mqRouter: mqRouter,
		redis:    redisClient,
	}
}

func (a *App) Run(ctx context.Context) error {
	trimCtx, trimCancel := context.WithCancel(ctx)
	mq.StartTrimmer(trimCtx, &a.cfg.MQ, a.redis)

	errCh := make(chan error, 2)

	go func() {
		logger.Log.Infof("server starting on port %d", a.cfg.Server.Port)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	if a.mqRouter != nil {
		go func() {
			if err := a.mqRouter.Run(ctx); err != nil && ctx.Err() == nil {
				errCh <- fmt.Errorf("mq router failed: %w", err)
			}
		}()
	}

	var runErr error
	select {
	case <-ctx.Done():
	case err := <-errCh:
		runErr = err
	}

	trimCancel()

	shutdownTimeout := 10 * time.Second
	if a.cfg.Server.ShutdownTimeout > 0 {
		shutdownTimeout = time.Duration(a.cfg.Server.ShutdownTimeout) * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if a.mqRouter != nil {
		if err := a.mqRouter.Close(); err != nil && runErr == nil {
			runErr = fmt.Errorf("close mq router failed: %w", err)
		}
	}

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil && runErr == nil {
		runErr = fmt.Errorf("shutdown http server failed: %w", err)
	}

	return runErr
}
