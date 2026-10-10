package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string
	Slog            slog.Level
	GinMode         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func getDuration(n string) (time.Duration, error) {
	t := os.Getenv(n)
	if t == "" {
		return 10 * time.Second, nil
	}
	v, err := time.ParseDuration(t)
	if err != nil {
		return 0, fmt.Errorf("wrong format for %s: %w", n, err)
	}

	return v, nil
}

func Load() (Config, error) {
	var cfg Config
	var err error
	if p := os.Getenv("PORT"); p == "" {
		cfg.Port = ":8080"
	} else if convertedPort, err := strconv.Atoi(p); err == nil && (1 <= convertedPort && convertedPort <= 65535) {
		cfg.Port = fmt.Sprintf(":%d", convertedPort)
	} else {
		return Config{}, fmt.Errorf("wrong port %v", p)
	}
	if l := strings.ToLower(os.Getenv("LOG_LEVEL")); l == "" || l == "debug" {
		cfg.Slog = slog.LevelDebug
	} else {
		if l == "info" {
			cfg.Slog = slog.LevelInfo
		} else if l == "warn" {
			cfg.Slog = slog.LevelWarn
		} else if l == "error" {
			cfg.Slog = slog.LevelError
		} else {
			return Config{}, fmt.Errorf("wrong level: %v", l)
		}
	}
	if appEnv := os.Getenv("APP_ENV"); appEnv == "" {
		cfg.GinMode = "debug"
	} else if appEnv == "prod" {
		cfg.GinMode = "release"
	} else if appEnv == "dev" {
		cfg.GinMode = "debug"
	} else {
		return Config{}, fmt.Errorf("wrong APP_ENV: %v", appEnv)
	}
	cfg.WriteTimeout, err = getDuration("WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.ReadTimeout, err = getDuration("READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.IdleTimeout, err = getDuration("IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout, err = getDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
