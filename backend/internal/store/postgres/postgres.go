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

var _ service.ItemRepository = (*ItemRepo)(nil)

type ItemRepo struct {
	pool *pgxpool.Pool
}

const (
	checkViolation  string = "23514"
	uniqueViolation string = "23505"
)

func New(p *pgxpool.Pool) *ItemRepo {
	return &ItemRepo{pool: p}
}

func (db *ItemRepo) Create(ctx context.Context, item models.Item) (models.Item, error) {
	err := db.pool.QueryRow(ctx, `INSERT INTO items (name) VALUES ($1) RETURNING id`, item.Name).Scan(&item.ID)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && (pgErr.Code == uniqueViolation) {
			return models.Item{}, fmt.Errorf("create item: %w", models.ErrConflict)
		}
		return models.Item{}, fmt.Errorf("create item: %w", err)
	}
	return item, nil
}

func (db *ItemRepo) Find(ctx context.Context, id int) (models.Item, error) {
	var item models.Item
	err := db.pool.QueryRow(ctx, `SELECT id, name FROM items WHERE id = $1`, id).Scan(&item.ID, &item.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Item{}, fmt.Errorf("find item: %w", models.ErrNotFound)
		}
		return models.Item{}, fmt.Errorf("find item: %w", err)
	}
	return item, nil
}

func (db *ItemRepo) All(ctx context.Context) ([]models.Item, error) {
	rows, err := db.pool.Query(ctx, `SELECT id, name FROM items ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()
	items := []models.Item{}
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	return items, nil
}

func (db *ItemRepo) Update(ctx context.Context, id int, item models.Item) error {
	tag, err := db.pool.Exec(ctx, `UPDATE items SET name = $1 WHERE id = $2`, item.Name, id)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return fmt.Errorf("update item: %w", models.ErrConflict)
		}
		return fmt.Errorf("update item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update item: %w", models.ErrNotFound)
	}
	return nil
}

func (db *ItemRepo) Delete(ctx context.Context, id int) error {
	tag, err := db.pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete item: %w", models.ErrNotFound)
	}
	return nil
}
