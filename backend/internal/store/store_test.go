package store

import (
	"context"
	"errors"
	"lvosa/backend/internal/models"
	"slices"
	"sync"
	"testing"
)

func createStore(t *testing.T) *ItemMemory {
	t.Helper()
	store := New()
	_, err := store.Create(context.Background(), models.Item{Name: "prvi"})
	if err != nil {
		t.Fatalf("seed %q: %v", err, "prvi")
	}
	_, err = store.Create(context.Background(), models.Item{Name: "drugi"})
	if err != nil {
		t.Fatalf("seed %q: %v", err, "drugi")
	}
	return store
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name        string
		newName     string
		numberItems int
		wantErr     error
		wantName    string
		wantID      int
	}{
		{name: "valid name", newName: "novo", numberItems: 3, wantErr: nil, wantName: "novo", wantID: 3},
		{name: "exact double", newName: "drugi", numberItems: 2, wantErr: models.ErrConflict, wantName: "", wantID: 0},
		{name: "exact double diff register", newName: "DRUGI", numberItems: 2, wantErr: models.ErrConflict, wantName: "", wantID: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := createStore(t)
			item, err := store.Create(context.Background(), models.Item{Name: test.newName})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected: %v, received: %v", test.wantErr, err)
			}
			allItems, err := store.All(context.Background())
			if err != nil {
				t.Fatalf("unexpected error returned %v", err)
			}
			if len(allItems) != test.numberItems {
				t.Fatalf("expected number of ids: %d, received: %d", test.numberItems, len(allItems))
			}
			if item.Name != test.wantName {
				t.Fatalf("expected name of new item: %q, received: %q", test.wantName, item.Name)
			}
			if item.ID != test.wantID {
				t.Fatalf("expected ID: %d, received ID: %d", test.wantID, item.ID)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		newName   string
		wantErr   error
		wantName1 string
		wantName2 string
	}{
		{name: "valid rename", id: 1, newName: "novo", wantErr: nil, wantName1: "novo", wantName2: "drugi"},
		{name: "same name other case", id: 1, newName: "Prvi", wantErr: nil, wantName1: "Prvi", wantName2: "drugi"},
		{name: "conflict with item after", id: 1, newName: "drugi", wantErr: models.ErrConflict, wantName1: "prvi", wantName2: "drugi"},
		{name: "conflict with item before", id: 2, newName: "prvi", wantErr: models.ErrConflict, wantName1: "prvi", wantName2: "drugi"},
		{name: "not found", id: 99, newName: "novo", wantErr: models.ErrNotFound, wantName1: "prvi", wantName2: "drugi"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := createStore(t)
			err := store.Update(context.Background(), test.id, models.Item{Name: test.newName})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			item1, err := store.Find(context.Background(), 1)
			if err != nil {
				t.Fatalf("find id 1, unexpected error: %v", err)
			}
			if item1.Name != test.wantName1 {
				t.Errorf("wanted: %q, received: %q", test.wantName1, item1.Name)
			}
			item2, err := store.Find(context.Background(), 2)
			if err != nil {
				t.Fatalf("find id 2, unexpected error: %v", err)
			}
			if item2.Name != test.wantName2 {
				t.Errorf("wanted: %q, received: %q", test.wantName2, item2.Name)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name              string
		id                int
		wantErr           error
		wantNumberOfItems int
		wantRemaining     int
	}{
		{name: "valid id(1)", id: 1, wantErr: nil, wantNumberOfItems: 1, wantRemaining: 2},
		{name: "valid id(2)", id: 2, wantErr: nil, wantNumberOfItems: 1, wantRemaining: 1},
		{name: "wrong id", id: 99, wantErr: models.ErrNotFound, wantNumberOfItems: 2, wantRemaining: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := createStore(t)
			err := store.Delete(context.Background(), test.id)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			items, err := store.All(context.Background())
			if err != nil {
				t.Fatalf("unexpected error returned %v", err)
			}
			if len(items) != test.wantNumberOfItems {
				t.Fatalf("expected number of items: %d, received: %d", test.wantNumberOfItems, len(items))
			}
			_, err = store.Find(context.Background(), test.wantRemaining)
			if err != nil {
				t.Fatalf("expecting id %d not found: %v", test.wantRemaining, err)
			}
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name      string
		wantItems []models.Item
		wantErr   error
	}{
		{name: "valid receiving items", wantItems: []models.Item{{Name: "prvi", ID: 1}, {Name: "drugi", ID: 2}}, wantErr: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := createStore(t)
			items, err := store.All(context.Background())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("unexpected error %v, want %v", err, test.wantErr)
			}
			if !slices.Equal(items, test.wantItems) {
				t.Fatalf("received items %v not equal to downloaded %v", items, test.wantItems)
			}
		})
	}
}

func TestCreateConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	testItem := models.Item{Name: "dimas"}
	store := createStore(t)
	for range 50 {
		wg.Go(func() {
			_, err := store.Create(context.Background(), testItem)
			if !errors.Is(err, models.ErrConflict) && err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}
		})

	}
	wg.Wait()
	items, err := store.All(context.Background())
	if err != nil {
		t.Fatalf("unexpected error returned %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3, received %d", len(items))
	}
}
