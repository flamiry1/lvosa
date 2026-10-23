package handlers

import (
	"context"
	"lvosa/backend/internal/models"
	"time"
)

type Authenticator interface {
	Register(ctx context.Context, email, password string) (models.User, error)
	Login(ctx context.Context, email, password string) (models.User, string, error)
	Logout(ctx context.Context, token string) error
	CurrentUser(ctx context.Context, token string) (models.User, error)
}

type AuthHandler struct {
	auth      Authenticator
	ttlCookie time.Duration
	secure    bool
}
