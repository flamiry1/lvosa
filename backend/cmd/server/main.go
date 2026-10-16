package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"lvosa/backend/internal/service"
	"lvosa/backend/internal/store/postgres"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lvosa/backend/internal/config"
	"lvosa/backend/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func run() error {
	redis.SetLogger(slogAdapter{})
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	level.Set(cfg.Slog)
	gin.SetMode(cfg.GinMode)
	startCtx, startCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer startCancel()
	pool, err := pgxpool.New(startCtx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("parse postgres dsn: %w", err)
	}
	defer pool.Close()
	err = pool.Ping(startCtx)
	if err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	slog.Info("postgres connected")

	opts, err := redis.ParseURL(cfg.RedisDSN)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	defer client.Close()
	status := client.Ping(startCtx)
	if status.Err() != nil {
		return fmt.Errorf("ping redis: %w", status.Err())
	}
	slog.Info("redis connected")
	repo := postgres.New(pool)
	sr := service.New(repo)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	h := handlers.New(sr)
	h.RegisterRouters(r)
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	errCh := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("shutdown", "signal", sig.String())
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forcefully closed by context: %w", err)
	}
	slog.Info("Server successfully closed")
	return nil
}

var level slog.LevelVar

func main() {
	_ = godotenv.Load()
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: &level})
	logger := slog.New(h)
	slog.SetDefault(logger)
	err := run()
	if err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}
