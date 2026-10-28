package main

import (
	"context"
	"fmt"
	"lvosa/backend/internal/config"
	"lvosa/backend/internal/handlers"
	"lvosa/backend/internal/service"
	"lvosa/backend/internal/store/postgres"
	"lvosa/backend/internal/store/redisstore"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func postgresConnect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func redisConnect(ctx context.Context, dsn string) (*redis.Client, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	status := client.Ping(ctx)
	if status.Err() != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", status.Err())
	}
	return client, nil
}

func newRouter(p *pgxpool.Pool, rdb *redis.Client, c *config.Config) (*gin.Engine, error) {
	itemRepo := postgres.NewItemRepo(p)
	itemService := service.NewItemService(itemRepo)
	sessionRepo := redisstore.NewSessionRepo(rdb, c.SessionTTL)
	usersRepo := postgres.NewUserRepo(p)
	authService, err := service.NewAuthService(usersRepo, sessionRepo)
	if err != nil {
		return nil, err
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	authHandler := handlers.NewAuthHandler(authService, c.SessionTTL, c.Secure)
	authHandler.RegisterRouters(r)
	itemsHandler := handlers.NewItemHandler(itemService)
	itemsHandler.RegisterRouters(r, authHandler.AuthMiddleware())
	return r, nil
}
