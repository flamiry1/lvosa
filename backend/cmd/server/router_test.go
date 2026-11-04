package main

import (
	"context"
	"errors"
	"lvosa/backend/internal/config"
	"lvosa/backend/internal/models"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type client struct {
	cookie *http.Cookie
	r      *gin.Engine
}

type redisClients struct {
	sessionDB *redis.Client
	ratesDB   *redis.Client
}

const attempts int = 5

var userBody string = `{"email": "dima@gmail.com", "password": "secret-password"}`
var itemBody string = `{"name": "test-name"}`

func (c *client) doRequest(t *testing.T, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	c.r.ServeHTTP(rec, req)
	recCookie := rec.Result().Cookies()
	for _, cookie := range recCookie {
		if cookie.Name == "session" {
			c.cookie = cookie
		}
	}
	return rec
}

func truncatePostgres(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `TRUNCATE items RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate postgre items table: %v", err)
	}
	_, err = pool.Exec(ctx, `TRUNCATE users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate postgre users table: %v", err)
	}
}

func flushRedis(t *testing.T, ctx context.Context, clientSession, clientRates *redis.Client) {
	t.Helper()
	if err := clientSession.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush sessions redis: %v", err)
	}
	if err := clientRates.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush rates redis: %v", err)
	}

}

func testRouter(t *testing.T) (*client, *pgxpool.Pool, *redisClients) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dbDSN := os.Getenv("TEST_DB_DSN")
	if dbDSN == "" {
		t.Skip("TEST_DB_DSN not specified")
	}
	ratesDSN := os.Getenv("TEST_REDIS_DSN_RATES")
	if ratesDSN == "" {
		t.Skip("TEST_REDIS_DSN_RATES not specified")
	}
	sessionDSN := os.Getenv("TEST_REDIS_DSN_SESSIONS")
	if sessionDSN == "" {
		t.Skip("TEST_REDIS_DSN_SESSIONS not specified")
	}
	pool, err := postgresConnect(ctx, dbDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	truncatePostgres(t, ctx, pool)
	sessionRDB, ratesRDB, err := getRedisClients(ctx, sessionDSN, ratesDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sessionRDB.Close() })
	t.Cleanup(func() { ratesRDB.Close() })
	flushRedis(t, ctx, sessionRDB, ratesRDB)
	cfg := &config.Config{
		SessionTTL:    168 * time.Hour,
		RateTTL:       15 * time.Minute,
		Secure:        false,
		LoginAttempts: attempts,
	}
	r, err := newRouter(pool, sessionRDB, ratesRDB, cfg)
	if err != nil {
		t.Fatal(err)
	}
	rDB := &redisClients{
		sessionDB: sessionRDB,
		ratesDB:   ratesRDB,
	}
	return &client{r: r}, pool, rDB
}

func TestFullUserRoute(t *testing.T) {
	client, _, _ := testRouter(t)
	rec := client.doRequest(t, http.MethodPost, "/register", userBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodPost, "/login", userBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodGet, "/me", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, received: %d", rec.Code)
	}
	oldCookieSaved := client.cookie
	rec = client.doRequest(t, http.MethodPost, "/logout", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, received: %d", rec.Code)
	}
	client.cookie = oldCookieSaved
	rec = client.doRequest(t, http.MethodGet, "/me", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, received: %d", rec.Code)
	}
}

func TestAdminRoleGrantsItemCreate(t *testing.T) {
	ctx := context.Background()
	client, p, _ := testRouter(t)
	rec := client.doRequest(t, http.MethodPost, "/register", userBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodPost, "/login", userBody)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodPost, "/item", itemBody)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, received: %d", rec.Code)
	}
	tag, err := p.Exec(ctx, `UPDATE users SET role = $1 WHERE email = $2`, models.RoleAdmin, "dima@gmail.com")
	if err != nil {
		t.Fatalf("error trying to set admin role: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("UPDATE affected not 1 row, affected rows: %d", tag.RowsAffected())
	}
	rec = client.doRequest(t, http.MethodPost, "/item", itemBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
}

func TestLoginTooManyRequests(t *testing.T) {
	ctx := context.Background()
	client, _, clients := testRouter(t)
	rec := client.doRequest(t, http.MethodPost, "/register", userBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
	for range attempts {
		rec = client.doRequest(t, http.MethodPost, "/login", `{"email": "dima@gmail.com", "password": "wrong-secret-password"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, received: %d", rec.Code)
		}
	}
	rec = client.doRequest(t, http.MethodPost, "/login", userBody)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, received: %d", rec.Code)
	}
	rate, err := clients.ratesDB.Get(ctx, "login_attempts:dima@gmail.com").Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			t.Fatalf("redis didn't find rate")
		}
		t.Fatalf("redis get rate: %v", err)
	}
	if rate != 6 {
		t.Fatalf("expected rate 6, received: %d", rate)
	}
	_, err = clients.sessionDB.Get(ctx, "login_attempts:dima@gmail.com").Result()
	if err == nil {
		t.Fatalf("rate key found in session db")
	}
	if !errors.Is(err, redis.Nil) {
		t.Fatalf("redis look for rate in session db: %v", err)
	}
}

func TestUnauthorized(t *testing.T) {
	client, _, _ := testRouter(t)
	rec := client.doRequest(t, http.MethodPost, "/register", userBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodGet, "/me", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodPost, "/item", itemBody)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, received: %d", rec.Code)
	}
}

func TestEmailWithSpacesDifferentRegister(t *testing.T) {
	client, _, _ := testRouter(t)
	rec := client.doRequest(t, http.MethodPost, "/register", userBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, received: %d", rec.Code)
	}
	rec = client.doRequest(t, http.MethodPost, "/login", `{"email": "   DIMA@gmail.com   ", "password": "secret-password"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, received: %d", rec.Code)
	}
}
