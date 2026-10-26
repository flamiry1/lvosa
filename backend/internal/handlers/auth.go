package handlers

import (
	"context"
	"fmt"
	"lvosa/backend/internal/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const sessionKey string = "session"

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
	r.POST("/logout", a.logout)
	r.GET("/me", a.currentUser)
}

func (a *AuthHandler) setCookieSession(c *gin.Context, value string, ttl int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionKey, value, ttl, "/", "", a.secure, true)
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
	a.setCookieSession(c, token, int(a.ttlCookie/time.Second))
	c.JSON(http.StatusOK, user)
}

func (a *AuthHandler) logout(c *gin.Context) {
	session, err := c.Cookie(sessionKey)
	if err != nil {
		a.setCookieSession(c, "", -1)
		c.Status(http.StatusNoContent)
		return
	}
	ctx := c.Request.Context()
	err = a.auth.Logout(ctx, session)
	if err != nil {
		handleError(err, c)
		return
	}
	a.setCookieSession(c, "", -1)
	c.Status(http.StatusNoContent)
}

func (a *AuthHandler) currentUser(c *gin.Context) {
	session, err := c.Cookie(sessionKey)
	if err != nil {
		handleError(models.ErrUnauthorized, c)
		return
	}
	ctx := c.Request.Context()
	user, err := a.auth.CurrentUser(ctx, session)
	if err != nil {
		handleError(err, c)
		return
	}
	c.JSON(http.StatusOK, user)
}
