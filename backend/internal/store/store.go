package store

import (
	"context"
	"fmt"
	"lvosa/backend/internal/models"
	"slices"
	"strings"
	"sync"
)

type ItemMemory struct {
	items  []models.Item
	mu     sync.RWMutex
	nextID int
}

func New() *ItemMemory {
	return &ItemMemory{nextID: 1}
}

func (i *ItemMemory) Create(ctx context.Context, n models.Item) (models.Item, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, item := range i.items {
		if strings.EqualFold(item.Name, n.Name) {
			return models.Item{}, fmt.Errorf("%w: name already registered", models.ErrConflict)
		}
	}
	n.ID = i.nextID
	i.nextID++
	i.items = append(i.items, n)
	return n, nil

}

func (i *ItemMemory) Find(ctx context.Context, id int) (models.Item, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	for _, item := range i.items {
		if item.ID == id {
			return item, nil
		}
	}
	return models.Item{}, models.ErrNotFound
}

func (i *ItemMemory) Update(ctx context.Context, id int, n models.Item) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	var found, alreadyExisted bool
	var foundIdx int
	for idx, item := range i.items {
		if strings.EqualFold(item.Name, n.Name) && item.ID != id {
			alreadyExisted = true
		}
		if id == item.ID {
			foundIdx = idx
			found = true
		}
	}
	if !found {
		return models.ErrNotFound
	}
	if alreadyExisted {
		return fmt.Errorf("%w: name already registered", models.ErrConflict)
	}
	i.items[foundIdx].Name = n.Name
	return nil
}

func (i *ItemMemory) Delete(ctx context.Context, id int) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	for index, item := range i.items {
		if item.ID == id {
			i.items = slices.Delete(i.items, index, index+1)
			return nil
		}
	}
	return models.ErrNotFound
}

func (i *ItemMemory) All(ctx context.Context) []models.Item {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return slices.Clone(i.items)
}
