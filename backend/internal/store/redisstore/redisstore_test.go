package redisstore

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

const testTTL time.Duration = 3 * time.Second

func initRedisRepo(t *testing.T) (*SessionRepo, *RateLimiterRepo) {
	t.Helper()
	testDbDsn := os.Getenv("TEST_REDIS_DSN_SESSIONS")
	if testDbDsn == "" {
		t.Skip("TEST_REDIS_DSN_SESSIONS not specified")
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
	return NewSessionRepo(client, time.Minute), NewRateLimiterRepo(client, testTTL)
}

func TestSessionRepo_CreateGet(t *testing.T) {
	rdb, _ := initRedisRepo(t)
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
	rdb, _ := initRedisRepo(t)
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
	rdb, _ := initRedisRepo(t)
	_, err := rdb.Get(context.Background(), "dima")
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("unexpected error: %v, want: %v", err, models.ErrNotFound)
	}
}

func TestSessionRepo_CreateDeleteGet(t *testing.T) {
	rdb, _ := initRedisRepo(t)
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
	rdb, _ := initRedisRepo(t)
	err := rdb.Delete(context.Background(), "hello, world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionRepo_CreateCheckTTL(t *testing.T) {
	rdb, _ := initRedisRepo(t)
	ctx := context.Background()
	token, err := rdb.Create(ctx, 100)
	if err != nil {
		t.Fatalf("redis create: %v", err)
	}
	ttl := rdb.client.PTTL(ctx, fmt.Sprintf("session:%s", token)).Val()
	if ttl > time.Minute {
		t.Fatalf("received ttl: %v, settled: %v", ttl, time.Minute)
	}
	if ttl < 0 {
		t.Fatalf("ttl less than 0: %v", ttl)
	}
}

func TestRateLimiterRepo_IncreaseRate(t *testing.T) {
	emails := []string{"dima@gmail.com", "Dima@gmail.com", "DIMA@GMAIL.COM"}
	key := createRateKey("dima@gmail.com")
	_, rdb := initRedisRepo(t)
	ctx := context.Background()
	var prevTTL time.Duration
	for i, email := range emails {
		want := i + 1
		count, err := rdb.IncreaseRate(ctx, email)
		if err != nil {
			t.Fatalf("increase rate %q: %v", email, err)
		}
		if count != want {
			t.Fatalf("email %q: expected count = %d, received: %d", email, want, count)
		}
		pttl, err := rdb.client.PTTL(ctx, key).Result()
		if err != nil {
			t.Fatalf("get pttl: %v", err)
		}
		if pttl <= 0 || pttl > testTTL {
			t.Fatalf("attempt %d: expected pttl in (0, %v], received: %v", want, testTTL, pttl)
		}
		// между попытками спим 50ms: без продления срок обязан уменьшиться хотя бы на 25ms
		if i > 0 && pttl > prevTTL-25*time.Millisecond {
			t.Fatalf("attempt %d: ttl was extended: before %v, now %v", want, prevTTL, pttl)
		}
		prevTTL = pttl
		time.Sleep(50 * time.Millisecond)
	}
}

func TestRateLimiterRepo_Expire(t *testing.T) {
	email := "dima@gmail.com"
	_, rdb := initRedisRepo(t)
	shortRdb := NewRateLimiterRepo(rdb.client, 50*time.Millisecond)
	ctx := context.Background()
	for range 2 {
		_, err := shortRdb.IncreaseRate(ctx, email)
		if err != nil {
			t.Fatalf("increase rate %q: %v", email, err)
		}
	}
	time.Sleep(60 * time.Millisecond)
	rate, err := shortRdb.IncreaseRate(ctx, email)
	if err != nil {
		t.Fatalf("increase rate %q: %v", email, err)
	}
	if rate != 1 {
		t.Fatalf("expected rate after expire: 1, received: %d", rate)
	}
}

func TestRateLimiterRepo_DropRate(t *testing.T) {
	email := "dima@gmail.com"
	_, rdb := initRedisRepo(t)
	ctx := context.Background()
	for range 3 {
		_, err := rdb.IncreaseRate(ctx, email)
		if err != nil {
			t.Fatalf("increase rate %q: %v", email, err)
		}
	}
	err := rdb.DropRate(ctx, email)
	if err != nil {
		t.Fatalf("drop rate %q: %v", email, err)
	}
	rate, err := rdb.IncreaseRate(ctx, email)
	if err != nil {
		t.Fatalf("increase rate %q: %v", email, err)
	}
	if rate != 1 {
		t.Fatalf("expected rate after drop: 1, received: %d", rate)
	}
}

func TestRateLimiterRepo_ConcurrentCheck(t *testing.T) {
	const n = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := make(map[int]int)
	email := "dima@gmail.com"
	_, rdb := initRedisRepo(t)
	ctx := context.Background()
	for range n {
		wg.Go(func() {
			rate, err := rdb.IncreaseRate(ctx, email)
			if err != nil {
				t.Errorf("increase rate %q: %v", email, err)
				return
			}
			mu.Lock()
			seen[rate]++
			mu.Unlock()
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	for i := 1; i <= n; i++ {
		if seen[i] != 1 {
			t.Fatalf("number %d received %d times, want 1; all: %v", i, seen[i], seen)
		}
	}
}
