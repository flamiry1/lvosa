package service

import (
	"context"
	"fmt"
	"lvosa/backend/internal/models"
	"strings"
	"unicode/utf8"
)

type ItemService struct {
	repo ItemRepository
}

type ItemRepository interface {
	All(ctx context.Context) []models.Item
	Find(ctx context.Context, id int) (models.Item, error)
	Create(ctx context.Context, item models.Item) (models.Item, error)
	Update(ctx context.Context, id int, item models.Item) error
	Delete(ctx context.Context, id int) error
}

func New(s ItemRepository) *ItemService {
	return &ItemService{repo: s}
}

func checkNameLength(name string) (string, bool) {
	name = strings.TrimSpace(name)
	lenRunes := utf8.RuneCountInString(name)
	if 1 <= lenRunes && lenRunes <= 100 {
		return name, true
	}
	return "", false
}

func (s *ItemService) List(ctx context.Context) []models.Item {
	return s.repo.All(ctx)
}

func (s *ItemService) Get(ctx context.Context, id int) (models.Item, error) {
	return s.repo.Find(ctx, id)
}

func (s *ItemService) Create(ctx context.Context, item models.Item) (models.Item, error) {
	name, isValidName := checkNameLength(item.Name)
	if !isValidName {
		return models.Item{}, fmt.Errorf("%w: length should be in 1 - 100", models.ErrBadRequest)
	}
	item.Name = name
	item, err := s.repo.Create(ctx, item)
	if err != nil {
		return models.Item{}, err
	}
	return item, nil
}

func (s *ItemService) Update(ctx context.Context, item models.Item, id int) error {
	name, isValidName := checkNameLength(item.Name)
	if !isValidName {
		return fmt.Errorf("%w: length should be in 1 - 100", models.ErrBadRequest)
	}
	item.Name = name
	return s.repo.Update(ctx, id, item)
}

func (s *ItemService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
