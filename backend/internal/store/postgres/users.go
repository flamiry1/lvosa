package postgres

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"lvosa/backend/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ service.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
	pool *pgxpool.Pool
}

func userScan(row pgx.Row) (models.User, error) {
	var user models.User
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return user, err
}

func NewUserRepo(p *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: p}
}

func (r *UserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	err := r.pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id, created_at`, user.Email, user.PasswordHash, user.Role).Scan(&user.ID, &user.CreatedAt)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return models.User{}, fmt.Errorf("create user: %w", models.ErrConflict)
		}
		return models.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, email, password_hash, role, created_at FROM users WHERE lower(email) = lower($1)`, email)
	user, err := userScan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("find user by email: %w", models.ErrNotFound)
		}
		return models.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int) (models.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, email, password_hash, role, created_at FROM users WHERE id = $1`, id)
	user, err := userScan(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("find user by ID: %w", models.ErrNotFound)
		}
		return models.User{}, fmt.Errorf("find user by ID: %w", err)
	}
	return user, nil
}
