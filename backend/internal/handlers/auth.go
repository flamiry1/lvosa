package handlers

import (
	"context"
	"fmt"
	"lvosa/backend/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

type authBody struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func NewAuthHandler(a Authenticator, ttl time.Duration, s bool) *AuthHandler {
	return &AuthHandler{auth: a, ttlCookie: ttl, secure: s}
}

func (a *AuthHandler) RegisterRouters(r *gin.Engine) {
	r.POST("/register", a.register)
	r.POST("/login", a.login)
}

func (a *AuthHandler) register(c *gin.Context) {
	var body authBody
	err := c.ShouldBindJSON(&body)
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	user, err := a.auth.Register(ctx, body.Email, body.Password)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (a *AuthHandler) login(c *gin.Context) {
	var body authBody
	err := c.ShouldBindJSON(&body)
	if err != nil {
		err = fmt.Errorf("%w: %w", models.ErrBadRequest, err)
		handleError(err, c)
		return
	}
	ctx := c.Request.Context()
	user, token, err := a.auth.Login(ctx, body.Email, body.Password)
	if err != nil {
		handleError(err, c)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session", token, int(a.ttlCookie/time.Second), "/", "", a.secure, true)
	c.JSON(http.StatusOK, user)
}
