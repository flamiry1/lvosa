package redisstore

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func initRedisRepo(t *testing.T) *SessionRepo {
	t.Helper()
	testDbDsn := os.Getenv("TEST_REDIS_DSN")
	if testDbDsn == "" {
		t.Skip("TEST_REDIS_DSN not specified")
	}
	opts, err := redis.ParseURL(testDbDsn)
	if err != nil {
		t.Fatalf("parse redis url: %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() {
		client.Close()
	})
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	if err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	return NewSessionRepo(client, time.Minute)
}

func TestSessionRepo_CreateGet(t *testing.T) {
	rdb := initRedisRepo(t)
	ctx := context.Background()
	token, err := rdb.Create(ctx, 43)
	if err != nil {
		t.Fatalf("redis create: %v", err)
	}
	userID, err := rdb.Get(ctx, token)
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	if userID != 43 {
		t.Fatalf("expected userID: 43, received: %d", userID)
	}
}

func TestSessionRepo_CreateTwoTokens(t *testing.T) {
	rdb := initRedisRepo(t)
	ctx := context.Background()
	token1, err := rdb.Create(ctx, 32)
	if err != nil {
		t.Fatalf("redis token1 create: %v", err)
	}
	token2, err := rdb.Create(ctx, 34)
	if err != nil {
		t.Fatalf("redis token2 create: %v", err)
	}
	if token1 == token2 {
		t.Fatalf("redis create made two identical tokens")
	}
}

func TestSessionRepo_GetInvalid(t *testing.T) {
	rdb := initRedisRepo(t)
	_, err := rdb.Get(context.Background(), "dima")
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("unexpected error: %v, want: %v", err, models.ErrNotFound)
	}
}

func TestSessionRepo_CreateDeleteGet(t *testing.T) {
	rdb := initRedisRepo(t)
	ctx := context.Background()
	token, err := rdb.Create(ctx, 43)
	if err != nil {
		t.Fatalf("redis create: %v", err)
	}
	err = rdb.Delete(ctx, token)
	if err != nil {
		t.Fatalf("redis delete: %v", err)
	}
	userID, err := rdb.Get(ctx, token)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("expected error: %v, received: %v", models.ErrNotFound, err)
	}
	if userID != 0 {
		t.Fatalf("expected userID: 0, received: %d", userID)
	}
}

func TestSessionRepo_DeleteInvalid(t *testing.T) {
	rdb := initRedisRepo(t)
	err := rdb.Delete(context.Background(), "hello, world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionRepo_CreateCheckTTL(t *testing.T) {
	rdb := initRedisRepo(t)
	ctx := context.Background()
	token, err := rdb.Create(ctx, 100)
	if err != nil {
		t.Fatalf("redis create: %v", err)
	}
	ttl := rdb.client.TTL(ctx, fmt.Sprintf("session:%s", token)).Val()
	if ttl > time.Minute {
		t.Fatalf("received ttl: %v, settled: %v", ttl, time.Minute)
	}
	if ttl < 0 {
		t.Fatalf("ttl less than 0: %v", ttl)
	}
}
