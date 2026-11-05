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
	PostgresDSN     string
	RedisDSNUsers   string
	RedisDSNRates   string
	SessionTTL      time.Duration
	RateTTL         time.Duration
	Secure          bool
	LoginAttempts   int
}

// func env(k, def string) string {
// 	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
// 		return v
// 	}
// 	return def
// }

func getDuration(n string, def time.Duration) (time.Duration, error) {
	t := os.Getenv(n)
	if t == "" {
		return def, nil
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
	cfg.PostgresDSN = os.Getenv("DB_DSN")
	if cfg.PostgresDSN == "" {
		return Config{}, fmt.Errorf("DB_DSN: empty db path")
	}
	cfg.RedisDSNUsers = os.Getenv("REDIS_DSN_SESSIONS")
	if cfg.RedisDSNUsers == "" {
		return Config{}, fmt.Errorf("REDIS_DSN_SESSIONS: empty db path")
	}
	cfg.RedisDSNRates = os.Getenv("REDIS_DSN_RATES")
	if cfg.RedisDSNRates == "" {
		return Config{}, fmt.Errorf("REDIS_DSN_RATES: empty db path")
	}
	if p := os.Getenv("PORT"); p == "" {
		cfg.Port = ":8080"
	} else if convertedPort, err := strconv.Atoi(p); err == nil && (1 <= convertedPort && convertedPort <= 65535) {
		cfg.Port = fmt.Sprintf(":%d", convertedPort)
	} else {
		return Config{}, fmt.Errorf("wrong port %v", p)
	}
	if level := strings.ToLower(os.Getenv("LOG_LEVEL")); level == "" || level == "debug" {
		cfg.Slog = slog.LevelDebug
	} else {
		switch level {
		case "info":
			cfg.Slog = slog.LevelInfo
		case "warn":
			cfg.Slog = slog.LevelWarn
		case "error":
			cfg.Slog = slog.LevelError
		default:
			return Config{}, fmt.Errorf("wrong level: %v", level)
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
	cfg.WriteTimeout, err = getDuration("WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.ReadTimeout, err = getDuration("READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.IdleTimeout, err = getDuration("IDLE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.ShutdownTimeout, err = getDuration("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.SessionTTL, err = getDuration("SESSION_TTL", 168*time.Hour)
	if err != nil {
		return Config{}, err
	}
	cfg.RateTTL, err = getDuration("LIMIT_RATE_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	secure := os.Getenv("COOKIE_SECURE")
	if secure == "" {
		cfg.Secure = false
	} else {
		parsedSecure, err := strconv.ParseBool(secure)
		if err != nil {
			return Config{}, fmt.Errorf("wrong format for COOKIE_SECURE: %v", err)
		}
		cfg.Secure = parsedSecure
	}
	loginAttempts := os.Getenv("LOGIN_ATTEMPTS")
	if loginAttempts == "" {
		cfg.LoginAttempts = 5
	} else {
		attempts, err := strconv.Atoi(loginAttempts)
		if err != nil {
			return Config{}, fmt.Errorf("wrong format for LOGIN_ATTEMPTS: %v", err)
		}
		if attempts <= 0 {
			return Config{}, fmt.Errorf("LOGIN_ATTEMPTS cannot be 0 or less than 0")
		}
		cfg.LoginAttempts = attempts
	}
	return cfg, nil
}
