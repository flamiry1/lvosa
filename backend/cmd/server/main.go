package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"lvosa/backend/internal/service"
	"lvosa/backend/internal/store"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lvosa/backend/internal/config"
	"lvosa/backend/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setLogger(c config.Config) *slog.Logger {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: c.Slog})
	logger := slog.New(h)
	return logger
}
func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error found in configuration:\n%v", err)
	}
	slog.SetDefault(setLogger(cfg))
	gin.SetMode(cfg.GinMode)
	s := store.New()
	sr := service.New(s)
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
		slog.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forcefully closed by context", "err", err)
		return
	}
	slog.Info("Server successfully closed")
}
