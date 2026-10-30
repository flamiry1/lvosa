package handlers

import (
	"context"
	"errors"
	"lvosa/backend/internal/models"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var _ Authenticator = (*fakeAuthenticator)(nil)

var serviceErr error = errors.New("all is down")

type fakeAuthenticator struct {
	returnUser               models.User
	registerCalls            int
	registerErr              error
	registerReceivedEmail    string
	registerReceivedPassword string
	loginCalls               int
	loginErr                 error
	loginReceivedEmail       string
	loginReceivedPassword    string
	loginReturnToken         string
	logoutCalls              int
	logoutErr                error
	logoutReceivedToken      string
	currentUserCalls         int
	currentUserErr           error
	currentUserReceivedToken string
}

func newAuthTestRouter(t *testing.T) (*gin.Engine, *fakeAuthenticator, *AuthHandler) {
	fakeAuth := &fakeAuthenticator{}
	t.Helper()
	gin.SetMode(gin.TestMode)
	authHandler := NewAuthHandler(fakeAuth, 5*time.Minute, false)
	r := gin.New()
	r.Use(gin.Recovery())
	authHandler.RegisterRouters(r)
	return r, fakeAuth, authHandler
}

func setCookie(t *testing.T, token string, req *http.Request) {
	c1 := &http.Cookie{Name: "lang", Value: "ru"}
	c2 := &http.Cookie{Name: sessionKey, Value: token}
	t.Helper()
	req.AddCookie(c1)
	req.AddCookie(c2)
}

func (f *fakeAuthenticator) Register(ctx context.Context, email, password string) (models.User, error) {
	f.registerCalls++
	f.registerReceivedEmail = email
	f.registerReceivedPassword = password
	if f.registerErr != nil {
		return models.User{}, f.registerErr
	}
	return f.returnUser, nil
}

func (f *fakeAuthenticator) Login(ctx context.Context, email, password string) (models.User, string, error) {
	f.loginCalls++
	f.loginReceivedEmail = email
	f.loginReceivedPassword = password
	if f.loginErr != nil {
		return models.User{}, "", f.loginErr
	}
	return f.returnUser, f.loginReturnToken, nil
}

func (f *fakeAuthenticator) Logout(ctx context.Context, token string) error {
	f.logoutCalls++
	f.logoutReceivedToken = token
	if f.logoutErr != nil {
		return f.logoutErr
	}
	return nil
}

func (f *fakeAuthenticator) CurrentUser(ctx context.Context, token string) (models.User, error) {
	f.currentUserCalls++
	f.currentUserReceivedToken = token
	if f.currentUserErr != nil {
		return models.User{}, f.currentUserErr
	}
	return f.returnUser, nil
}

func TestAuthRegister(t *testing.T) {
	tests := []struct {
		name                     string
		body                     string
		wantServiceRegisterCalls int
		wantCode                 int
		wantEmail                string
		wantPassword             string
		serviceErr               error
	}{
		{name: "valid", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceRegisterCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusCreated, serviceErr: nil},
		{name: "bad request (no body)", body: ``, wantServiceRegisterCalls: 0, wantEmail: "", wantPassword: "", wantCode: http.StatusBadRequest, serviceErr: nil},
		{name: "bad request (no password)", body: `{"email":"dima@mail.com"}`, wantServiceRegisterCalls: 0, wantEmail: "", wantPassword: "", wantCode: http.StatusBadRequest, serviceErr: nil},
		{name: "conflict", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceRegisterCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusConflict, serviceErr: models.ErrConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, fakeAuth, _ := newAuthTestRouter(t)
			fakeAuth.registerErr = test.serviceErr
			fakeAuth.returnUser = models.User{Email: "dima@mail.com", PasswordHash: "secret-hash-value"}
			rec := doRequest(t, router, http.MethodPost, "/register", test.body, "")
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if rec.Code == http.StatusCreated {
				if !strings.Contains(rec.Body.String(), test.wantEmail) {
					t.Fatalf("body doesn't contain email: %s", test.wantEmail)
				}
				if strings.Contains(rec.Body.String(), "secret-hash-value") {
					t.Fatalf("unexpectedly return hash")
				}
			}
			if fakeAuth.registerCalls != test.wantServiceRegisterCalls {
				t.Fatalf("expected registerCalls: %d, received: %d", test.wantServiceRegisterCalls, fakeAuth.registerCalls)
			}
			if fakeAuth.registerReceivedEmail != test.wantEmail {
				t.Fatalf("expected email: %s, received: %s", test.wantEmail, fakeAuth.registerReceivedEmail)
			}
			if fakeAuth.registerReceivedPassword != test.wantPassword {
				t.Fatalf("expected password: %s, received: %s", test.wantPassword, fakeAuth.registerReceivedPassword)
			}
		})
	}
}

func TestAuthLogin(t *testing.T) {
	tests := []struct {
		name                  string
		body                  string
		wantBody              string
		wantServiceLoginCalls int
		wantEmail             string
		wantPassword          string
		wantCode              int
		serviceErr            error
	}{
		{name: "valid log in", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceLoginCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusOK, serviceErr: nil},
		{name: "invalid log in: invalid creds", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceLoginCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusUnauthorized, serviceErr: models.ErrInvalidCredentials, wantBody: `{"status":"wrong email or password"}`},
		{name: "invalid log in: service error", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceLoginCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusInternalServerError, serviceErr: serviceErr, wantBody: `{"status":"internal server error"}`},
		{name: "invalid log in: no password in body", body: `{"email":"dima@mail.com"}`, wantServiceLoginCalls: 0, wantEmail: "", wantPassword: "", wantCode: http.StatusBadRequest, serviceErr: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, fakeAuth, srv := newAuthTestRouter(t)
			fakeAuth.loginErr = test.serviceErr
			fakeAuth.loginReturnToken = "test-secret-token"
			fakeAuth.returnUser = models.User{Email: "dima@mail.com", PasswordHash: "secret-hash-value"}
			rec := doRequest(t, router, http.MethodPost, "/login", test.body, "")
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if test.serviceErr != nil {
				if rec.Body.String() != test.wantBody {
					t.Fatalf("expected body: %v, received: %v", test.wantBody, rec.Body.String())
				}
			}
			if rec.Code == http.StatusOK {
				if len(rec.Result().Cookies()) == 0 {
					t.Fatalf("expected cookies filled, received: %d", len(rec.Result().Cookies()))
				}
				if strings.Contains(rec.Body.String(), "secret-hash-value") {
					t.Fatalf("unexpectedly return hash")
				}
				if !strings.Contains(rec.Body.String(), test.wantEmail) {
					t.Fatalf("body doesn't contain email: %s", test.wantEmail)
				}
				if strings.Contains(rec.Body.String(), "test-secret-token") {
					t.Fatalf("unexpectedly return token")
				}
				cookies := rec.Result().Cookies()[0]
				if cookies.Secure {
					t.Fatalf("expected HttpOnly: %v, received: %v", srv.secure, cookies.Secure)
				}
				if !cookies.HttpOnly {
					t.Fatalf("expected HttpOnly: true, received: %v", cookies.HttpOnly)
				}
				if cookies.MaxAge != 300 {
					t.Fatalf("expected MaxAge: %d, received: %d", int(srv.ttlCookie/time.Second), cookies.MaxAge)
				}
				if cookies.SameSite != http.SameSiteLaxMode {
					t.Fatalf("expected SameSite: 2, received: %d", cookies.SameSite)
				}
				if cookies.Value != fakeAuth.loginReturnToken {
					t.Fatalf("expected session token: %s, received: %s", fakeAuth.loginReturnToken, cookies.Value)
				}
				if cookies.Name != sessionKey {
					t.Fatalf("expected cookie name: %s, received: %s", sessionKey, cookies.Name)
				}
			} else {
				if len(rec.Result().Cookies()) > 0 {
					t.Fatalf("expected cookies empty, received: %d", len(rec.Result().Cookies()))
				}
			}
			if fakeAuth.loginCalls != test.wantServiceLoginCalls {
				t.Fatalf("expected registerCalls: %d, received: %d", test.wantServiceLoginCalls, fakeAuth.loginCalls)
			}
			if fakeAuth.loginReceivedEmail != test.wantEmail {
				t.Fatalf("expected email: %s, received: %s", test.wantEmail, fakeAuth.loginReceivedPassword)
			}
			if fakeAuth.loginReceivedPassword != test.wantPassword {
				t.Fatalf("expected password: %s, received: %s", test.wantPassword, fakeAuth.loginReceivedPassword)
			}
		})
	}
}

func TestAuthLogout(t *testing.T) {
	tests := []struct {
		name            string
		wantLogoutCalls int
		wantCode        int
		wantToken       string
		wantBody        string
		cookieToken     string
		serviceErr      error
	}{
		{name: "valid log out: token", wantLogoutCalls: 1, wantCode: http.StatusNoContent, wantToken: "test-secret-token", cookieToken: "test-secret-token", serviceErr: nil, wantBody: ""},
		{name: "valid log out: no token", wantLogoutCalls: 0, wantCode: http.StatusNoContent, wantToken: "", cookieToken: "", serviceErr: nil, wantBody: ""},
		{name: "invalid log out: service error", wantLogoutCalls: 1, wantCode: http.StatusInternalServerError, wantToken: "test-secret-token", cookieToken: "test-secret-token", serviceErr: serviceErr, wantBody: `{"status":"internal server error"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, fakeAuth, _ := newAuthTestRouter(t)
			fakeAuth.logoutErr = test.serviceErr
			rec := doRequest(t, router, http.MethodPost, "/logout", "", test.cookieToken)
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if rec.Code == http.StatusNoContent {
				if len(rec.Result().Cookies()) == 0 {
					t.Fatalf("logout did not clear the session cookie")
				}
				cookies := rec.Result().Cookies()[0]
				if cookies.Name != sessionKey {
					t.Fatalf("expected name: %s, received: %s", sessionKey, cookies.Name)
				}
				if cookies.Value != "" {
					t.Fatalf("expected empty token, received: %s", cookies.Value)
				}
				if cookies.MaxAge != -1 {
					t.Fatalf("expected MaxAge -1, received: %d", cookies.MaxAge)
				}
			}
			if test.serviceErr != nil {
				if rec.Body.String() != test.wantBody {
					t.Fatalf("expected body: %v, received: %v", test.wantBody, rec.Body.String())
				}
			}
			if fakeAuth.logoutCalls != test.wantLogoutCalls {
				t.Fatalf("expected logoutCalls: %d, received: %d", test.wantLogoutCalls, fakeAuth.logoutCalls)
			}
			if fakeAuth.logoutReceivedToken != test.wantToken {
				t.Fatalf("expected token: %s, received: %s", test.wantToken, fakeAuth.logoutReceivedToken)
			}
		})
	}
}

func TestAuthCurrentUser(t *testing.T) {
	tests := []struct {
		name                 string
		wantCurrentUserCalls int
		wantCode             int
		wantToken            string
		wantBody             string
		cookieToken          string
		serviceErr           error
	}{
		{name: "valid current user", wantCurrentUserCalls: 1, wantCode: http.StatusOK, wantToken: "test-secret-token", cookieToken: "test-secret-token", serviceErr: nil, wantBody: ""},
		{name: "invalid current user: no token in cookies", wantCurrentUserCalls: 0, wantCode: http.StatusUnauthorized, wantToken: "", cookieToken: "", serviceErr: nil, wantBody: ""},
		{name: "invalid current user: unauthorized", wantCurrentUserCalls: 1, wantCode: http.StatusUnauthorized, wantToken: "test-secret-token", cookieToken: "test-secret-token", serviceErr: models.ErrUnauthorized, wantBody: `{"status":"unauthorized"}`},
		{name: "invalid current user: service error", wantCurrentUserCalls: 1, wantCode: http.StatusInternalServerError, wantToken: "test-secret-token", cookieToken: "test-secret-token", serviceErr: serviceErr, wantBody: `{"status":"internal server error"}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, fakeAuth, _ := newAuthTestRouter(t)
			fakeAuth.currentUserErr = test.serviceErr
			fakeAuth.returnUser = models.User{Email: "dima@mail.com", PasswordHash: "secret-hash-value"}
			rec := doRequest(t, router, http.MethodGet, "/me", "", test.cookieToken)
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if test.serviceErr != nil {
				if rec.Body.String() != test.wantBody {
					t.Fatalf("expected body: %v, received: %v", test.wantBody, rec.Body.String())
				}
			}
			if rec.Code == http.StatusOK {
				if strings.Contains(rec.Body.String(), "secret-hash-value") {
					t.Fatalf("unexpectedly return hash")
				}
				if strings.Contains(rec.Body.String(), "test-secret-token") {
					t.Fatalf("unexpectedly return token")
				}
				if !strings.Contains(rec.Body.String(), "dima@mail.com") {
					t.Fatalf("body doesn't contain email: %s", "dima@mail.com")
				}
			}
			if fakeAuth.currentUserCalls != test.wantCurrentUserCalls {
				t.Fatalf("expected currentUserCalls: %d, received: %d", test.wantCurrentUserCalls, fakeAuth.currentUserCalls)
			}
			if fakeAuth.currentUserReceivedToken != test.wantToken {
				t.Fatalf("expected token: %s, received: %s", test.wantToken, fakeAuth.currentUserReceivedToken)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name                 string
		email                string
		wantEmail            string
		wantCode             int
		wantHandleCalled     bool
		wantCurrentUserCalls int
		wantBody             string
		serviceErr           error
		cookieToken          string
		wantToken            string
	}{
		{name: "valid entering", email: "dima@gmail.com", wantCode: http.StatusOK, wantHandleCalled: true, wantCurrentUserCalls: 1, wantBody: "", cookieToken: "test-secret-token", wantToken: "test-secret-token", serviceErr: nil},
		{name: "invalid entering: no cookies", email: "dima@gmail.com", wantCode: http.StatusUnauthorized, wantHandleCalled: false, wantCurrentUserCalls: 0, wantBody: "", cookieToken: "", wantToken: "", serviceErr: nil},
		{name: "invalid entering: service return unauthorized", email: "dima@gmail.com", wantCode: http.StatusUnauthorized, wantHandleCalled: false, wantCurrentUserCalls: 1, wantBody: `{"status":"unauthorized"}`, cookieToken: "test-secret-token", wantToken: "test-secret-token", serviceErr: models.ErrUnauthorized},
		{name: "invalid entering: internal server error", email: "dima@gmail.com", wantCode: http.StatusInternalServerError, wantHandleCalled: false, wantCurrentUserCalls: 1, wantBody: `{"status":"internal server error"}`, cookieToken: "test-secret-token", wantToken: "test-secret-token", serviceErr: serviceErr},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handleCalled := false
			gin.SetMode(gin.TestMode)
			router := gin.New()
			fakeAuth := &fakeAuthenticator{returnUser: models.User{Email: "dima@gmail.com", PasswordHash: "secret-hash-value"}}
			fakeAuth.currentUserErr = test.serviceErr
			authHandler := NewAuthHandler(fakeAuth, 5*time.Minute, false)
			router.Use(gin.Recovery())
			router.GET("/protected", authHandler.AuthMiddleware(), func(c *gin.Context) {
				handleCalled = true
				user, err := getUserFromContext(c)
				if err != nil {
					return
				}
				c.JSON(http.StatusOK, user)
			})
			rec := doRequest(t, router, http.MethodGet, "/protected", "", test.cookieToken)
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if rec.Code == http.StatusOK {
				if strings.Contains(rec.Body.String(), "secret-hash-value") {
					t.Fatalf("unexpectedly return hash")
				}
				if strings.Contains(rec.Body.String(), "test-secret-token") {
					t.Fatalf("unexpectedly return token")
				}
				if !strings.Contains(rec.Body.String(), "dima@gmail.com") {
					t.Fatalf("body doesn't contain email: %s", "dima@mail.com")
				}
			}
			if test.serviceErr != nil {
				if rec.Body.String() != test.wantBody {
					t.Fatalf("expected body: %v, received: %v", test.wantBody, rec.Body.String())
				}
			}
			if handleCalled != test.wantHandleCalled {
				t.Fatalf("expected handle be called, received: %v", handleCalled)
			}
			if fakeAuth.currentUserCalls != test.wantCurrentUserCalls {
				t.Fatalf("expected currentUserCalls: %d, received: %d", test.wantCurrentUserCalls, fakeAuth.currentUserCalls)
			}
			if fakeAuth.currentUserReceivedToken != test.wantToken {
				t.Fatalf("expected token: %s, received: %s", test.wantToken, fakeAuth.currentUserReceivedToken)
			}
		})
	}
}

func TestCheckRole(t *testing.T) {
	tests := []struct {
		name             string
		role             models.Role
		userCtx          models.User
		isUser           bool
		wantCode         int
		wantBody         string
		wantHandleCalled bool
	}{
		{name: "valid: role admin", wantCode: http.StatusOK, role: models.RoleAdmin, userCtx: models.User{Email: "dima@gmail.com", PasswordHash: "secret-hash-value", Role: models.RoleAdmin}, isUser: true, wantBody: `{"result":"ok"}`, wantHandleCalled: true},
		{name: "valid: role student for role student", wantCode: http.StatusOK, role: models.RoleStudent, userCtx: models.User{Email: "dima@gmail.com", PasswordHash: "secret-hash-value", Role: models.RoleStudent}, isUser: true, wantBody: `{"result":"ok"}`, wantHandleCalled: true},
		{name: "invalid: role student for role admin", wantCode: http.StatusForbidden, role: models.RoleAdmin, userCtx: models.User{Email: "dima@gmail.com", PasswordHash: "secret-hash-value", Role: models.RoleStudent}, isUser: true, wantBody: `{"status":"forbidden"}`, wantHandleCalled: false},
		{name: "invalid: no user set in context", wantCode: http.StatusInternalServerError, role: models.RoleAdmin, userCtx: models.User{Email: "dima@gmail.com", PasswordHash: "secret-hash-value", Role: models.RoleAdmin}, isUser: false, wantBody: `{"status":"internal server error"}`, wantHandleCalled: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handleCalled := false
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(gin.Recovery())
			router.GET("/admin-only", func(c *gin.Context) {
				if test.isUser {
					c.Set(userCtxKey, test.userCtx)
				}
				c.Next()
			}, AuthCheckRole(test.role), func(c *gin.Context) {
				handleCalled = true
				c.JSON(http.StatusOK, gin.H{"result": "ok"})
			})
			rec := doRequest(t, router, http.MethodGet, "/admin-only", "", "")
			if rec.Code != test.wantCode {
				t.Fatalf("expected status %d, received: %d", test.wantCode, rec.Code)
			}
			if handleCalled != test.wantHandleCalled {
				t.Fatalf("expected handle be called, received: %v", handleCalled)
			}
			if rec.Body.String() != test.wantBody {
				t.Fatalf("expected body: %v, received: %v", test.wantBody, rec.Body.String())
			}
		})
	}
}
