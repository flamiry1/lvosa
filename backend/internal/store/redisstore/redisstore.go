package redisstore

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSessionRepo(client *redis.Client, t time.Duration) *SessionRepo {
	return &SessionRepo{client: client, ttl: t}
}

func createKey(token string) string {
	newKey := fmt.Sprintf("session:%s", token)
	return newKey
}

func (r *SessionRepo) Create(ctx context.Context, userID int) (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	key := createKey(token)
	strUseID := strconv.Itoa(userID)
	status := r.client.Set(ctx, key, strUseID, r.ttl)
	if status.Err() != nil {
		return "", fmt.Errorf("redis-set session: %w", status.Err())
	}
	return token, nil
}

func (r *SessionRepo) Get(ctx context.Context, token string) (int, error) {
	key := createKey(token)
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
	key := createKey(token)
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
