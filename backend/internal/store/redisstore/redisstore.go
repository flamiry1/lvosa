package redisstore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flamiry1/lvosa/backend/internal/models"
	"github.com/flamiry1/lvosa/backend/internal/service"

	"github.com/redis/go-redis/v9"
)

var _ service.SessionRepository = (*SessionRepo)(nil)

type SessionRepo struct {
	client *redis.Client
	ttl    time.Duration
}

type RateLimiterRepo struct {
	client *redis.Client
	ttl    time.Duration
}

var upsertScript = redis.NewScript(`
	local current = redis.call('INCR', KEYS[1])
	if current == 1 then
		redis.call('PEXPIRE', KEYS[1], ARGV[1])
	end
	return current
	`)

func NewSessionRepo(client *redis.Client, ttl time.Duration) *SessionRepo {
	return &SessionRepo{client: client, ttl: ttl}
}

func NewRateLimiterRepo(client *redis.Client, ttl time.Duration) *RateLimiterRepo {
	return &RateLimiterRepo{client: client, ttl: ttl}
}

func createSessionKey(token string) string {
	newKey := fmt.Sprintf("session:%s", token)
	return newKey
}

func createRateKey(email string) string {
	newKey := fmt.Sprintf("login_attempts:%s", strings.ToLower(email))
	return newKey
}

func (r *SessionRepo) Create(ctx context.Context, userID int) (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	key := createSessionKey(token)
	strUseID := strconv.Itoa(userID)
	err = r.client.Set(ctx, key, strUseID, r.ttl).Err()
	if err != nil {
		return "", fmt.Errorf("redis-set session: %w", err)
	}
	return token, nil
}

func (r *SessionRepo) Get(ctx context.Context, token string) (int, error) {
	key := createSessionKey(token)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("redis-get session: %w", models.ErrNotFound)
		}
		return 0, fmt.Errorf("redis-get session: %w", err)
	}
	strUserID, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("convert ID to int: %w", err)
	}
	return strUserID, nil
}

func (r *SessionRepo) Delete(ctx context.Context, token string) error {
	key := createSessionKey(token)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *RateLimiterRepo) IncreaseRate(ctx context.Context, email string) (int, error) {
	key := createRateKey(email)
	rate, err := upsertScript.Run(ctx, r.client, []string{key}, r.ttl.Milliseconds()).Int()
	if err != nil {
		return 0, fmt.Errorf("redis increase rate: %w", err)
	}
	return rate, nil
}

func (r *RateLimiterRepo) DropRate(ctx context.Context, email string) error {
	key := createRateKey(email)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("drop rate limiter: %w", err)
	}
	return nil
}
