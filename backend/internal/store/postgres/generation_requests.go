package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/flamiry1/lvosa/backend/internal/models"
	"github.com/flamiry1/lvosa/backend/internal/service"
	"github.com/flamiry1/lvosa/backend/internal/store/postgres/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ service.GenerationRequestRepository = (*GenerationRequestRepo)(nil)

type GenerationRequestRepo struct {
	pool *pgxpool.Pool
	db   *db.Queries
}

func NewGenerationRequestRepo(p *pgxpool.Pool) *GenerationRequestRepo {
	return &GenerationRequestRepo{pool: p, db: db.New(p)}
}

func textPtr(i pgtype.Text) *string {
	if !i.Valid {
		return nil
	}
	return &i.String
}

func timeStampTZPtr(i pgtype.Timestamptz) *time.Time {
	if !i.Valid {
		return nil
	}
	return &i.Time
}

func (g *GenerationRequestRepo) Create(ctx context.Context, req models.GenerationRequest) (int, error) {
	id, err := g.db.AddRequest(ctx, db.AddRequestParams{
		UserID:     int64(req.UserID),
		Level:      string(req.Level),
		Topic:      req.Topic,
		Difficulty: string(req.Difficulty),
		Focus:      string(req.Focus),
		Size:       string(req.Size),
	})
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return 0, fmt.Errorf("create generation request: %w", models.ErrConflict)
		}
		return 0, fmt.Errorf("create generation request: %w", err)
	}
	return int(id), nil
}

func (g *GenerationRequestRepo) Get(ctx context.Context, id, userID int) (models.GenerationRequest, error) {
	req, err := g.db.GetRequest(ctx, db.GetRequestParams{
		ID:     int64(id),
		UserID: int64(userID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GenerationRequest{}, models.ErrNotFound
		}
		return models.GenerationRequest{}, fmt.Errorf("get generation request: %w", err)
	}
	modelsGenerationReuest := models.GenerationRequest{
		ID:         int(req.ID),
		UserID:     int(req.UserID),
		LessonID:   int8Ptr(req.LessonID),
		Level:      models.Level(req.Level),
		Topic:      req.Topic,
		Difficulty: models.Difficulty(req.Difficulty),
		Focus:      models.Focus(req.Focus),
		Size:       models.Size(req.Size),
		Status:     models.GenerationStatus(req.Status),
		Error:      textPtr(req.Error),
		Attempts:   int(req.Attempts),
		StartedAt:  timeStampTZPtr(req.StartedAt),
		FinishedAt: timeStampTZPtr(req.FinishedAt),
		CreatedAt:  req.CreatedAt.Time,
	}
	return modelsGenerationReuest, nil
}
