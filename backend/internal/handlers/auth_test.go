package handlers

import (
	"context"
	"fmt"
	"lvosa/backend/internal/models"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var _ Authenticator = (*fakeAuthenticator)(nil)

const cookieName string = "session"

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
}

func newAuthTestRouter(t *testing.T) (*gin.Engine, *fakeAuthenticator, *AuthHandler) {
	fakeAuth := &fakeAuthenticator{}
	t.Helper()
	gin.SetMode(gin.TestMode)
	srv := NewAuthHandler(fakeAuth, 5*time.Minute, false)
	r := gin.New()
	r.Use(gin.Recovery())
	srv.RegisterRouters(r)
	return r, fakeAuth, srv
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
	return nil
}

func (f *fakeAuthenticator) CurrentUser(ctx context.Context, token string) (models.User, error) {
	return models.User{}, nil
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
			rec := doRequest(t, router, http.MethodPost, "/register", test.body)
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
	serviceErr := fmt.Errorf("all is down")
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
		{name: "invalid log in: invalid creds", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceLoginCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusUnauthorized, serviceErr: models.ErrInvalidCredentials, wantBody: `{"status":"unauthorized"}`},
		{name: "invalid log in: service error", body: `{"email":"dima@mail.com", "password": "test-password"}`, wantServiceLoginCalls: 1, wantEmail: "dima@mail.com", wantPassword: "test-password", wantCode: http.StatusInternalServerError, serviceErr: serviceErr, wantBody: `{"status":"internal server error"}`},
		{name: "invalid log in: no password in body", body: `{"email":"dima@mail.com"}`, wantServiceLoginCalls: 0, wantEmail: "", wantPassword: "", wantCode: http.StatusBadRequest, serviceErr: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router, fakeAuth, srv := newAuthTestRouter(t)
			fakeAuth.loginErr = test.serviceErr
			fakeAuth.loginReturnToken = "test-secret-token"
			fakeAuth.returnUser = models.User{Email: "dima@mail.com", PasswordHash: "secret-hash-value"}
			rec := doRequest(t, router, http.MethodPost, "/login", test.body)
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
				if cookies.Name != cookieName {
					t.Fatalf("expected cookie name: %s, received: %s", cookieName, cookies.Name)
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
