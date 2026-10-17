package postgres

import (
	"context"
	"errors"
	"lvosa/backend/internal/models"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func initRepo(t *testing.T) *ItemRepo {
	t.Helper()
	testDbDsn := os.Getenv("TEST_DB_DSN")
	if testDbDsn == "" {
		t.Skip("TEST_DB_DSN not specified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDbDsn)
	if err != nil {
		t.Fatalf("DSN is invalid: %v", err)
	}
	t.Cleanup(pool.Close)
	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	repo := New(pool)
	_, err = repo.pool.Exec(ctx, `TRUNCATE items RESTART IDENTITY`)
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}
	_, err = repo.pool.Exec(ctx, `INSERT INTO items (name) VALUES ($1), ($2)`, "prvi", "drugi")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return repo
}

func TestItemRepo_Find(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		wantName string
		wantErr  error
	}{
		{name: "valid find", id: 1, wantName: "prvi", wantErr: nil},
		{name: "invalid find", id: 999, wantName: "", wantErr: models.ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := initRepo(t)
			item, err := r.Find(context.Background(), test.id)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("unexpected error: %v, want %v", err, test.wantErr)
			}
			if item.Name != test.wantName {
				t.Fatalf("unexpected name: %v,  want %v", item.Name, test.wantName)
			}
		})
	}
}

func TestItemRepo_Create(t *testing.T) {
	tests := []struct {
		name         string
		newName      string
		numerOfItems int
		wantName     string
		wantID       int
		wantErr      error
	}{
		{name: "valid creation", newName: "tretji", numerOfItems: 3, wantName: "tretji", wantID: 3, wantErr: nil},
		{name: "exact double diff register", newName: "PRVI", numerOfItems: 2, wantName: "", wantID: 0, wantErr: models.ErrConflict},
		{name: "conflict creation", newName: "prvi", numerOfItems: 2, wantName: "", wantID: 0, wantErr: models.ErrConflict},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := initRepo(t)
			item, err := r.Create(context.Background(), models.Item{Name: test.newName})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected: %v, received: %v", test.wantErr, err)
			}
			allItems, err := r.All(context.Background())
			if err != nil {
				t.Fatalf("unexpected error returned %v", err)
			}
			if len(allItems) != test.numerOfItems {
				t.Fatalf("expected number of ids: %d, received: %d", test.numerOfItems, len(allItems))
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

func TestItemRepo_Update(t *testing.T) {
	tests := []struct {
		name      string
		newName   string
		ID        int
		wantName1 string
		wantName2 string
		wantErr   error
	}{
		{name: "valid update", newName: "tretji", ID: 1, wantName1: "tretji", wantName2: "drugi", wantErr: nil},
		{name: "exact double diff register", newName: "Drugi", ID: 1, wantName1: "prvi", wantName2: "drugi", wantErr: models.ErrConflict},
		{name: "exact double ID diff register", newName: "Prvi", ID: 1, wantName1: "Prvi", wantName2: "drugi", wantErr: nil},
		{name: "conflict update with item after", newName: "drugi", ID: 1, wantName1: "prvi", wantName2: "drugi", wantErr: models.ErrConflict},
		{name: "conflict update with item before", newName: "prvi", ID: 2, wantName1: "prvi", wantName2: "drugi", wantErr: models.ErrConflict},
		{name: "not found", newName: "drugi", ID: 99, wantName1: "prvi", wantName2: "drugi", wantErr: models.ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := initRepo(t)
			err := r.Update(context.Background(), test.ID, models.Item{Name: test.newName})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			item1, err := r.Find(context.Background(), 1)
			if err != nil {
				t.Fatalf("find id 1, unexpected error: %v", err)
			}
			if item1.Name != test.wantName1 {
				t.Errorf("wanted: %q, received: %q", test.wantName1, item1.Name)
			}
			item2, err := r.Find(context.Background(), 2)
			if err != nil {
				t.Fatalf("find id 2, unexpected error: %v", err)
			}
			if item2.Name != test.wantName2 {
				t.Fatalf("wanted: %q, received: %q", test.wantName2, item2.Name)
			}
		})
	}
}

func TestItemRepo_All(t *testing.T) {
	tests := []struct {
		name      string
		wantItems []models.Item
		wantErr   error
	}{
		{name: "valid", wantItems: []models.Item{{Name: "prvi", ID: 1}, {Name: "drugi", ID: 2}}, wantErr: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := initRepo(t)
			items, err := r.All(context.Background())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("unexpected error %v, want %v", err, test.wantErr)
			}
			if !slices.Equal(items, test.wantItems) {
				t.Fatalf("received items %v not equal to downloaded %v", items, test.wantItems)
			}
		})
	}
}

func TestItemRepo_Delete(t *testing.T) {
	tests := []struct {
		name              string
		ID                int
		wantErr           error
		wantNumberOfItems int
		wantRemaining     int
	}{
		{name: "valid deletion ID 1", ID: 1, wantErr: nil, wantNumberOfItems: 1},
		{name: "valid deletion ID 2", ID: 2, wantErr: nil, wantNumberOfItems: 1},
		{name: "inexistence ID", ID: 99, wantErr: models.ErrNotFound, wantNumberOfItems: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := initRepo(t)
			err := r.Delete(context.Background(), test.ID)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			items, err := r.All(context.Background())
			if err != nil {
				t.Fatalf("unexpected error returned %v", err)
			}
			if len(items) != test.wantNumberOfItems {
				t.Fatalf("expected number of items: %d, received: %d", test.wantNumberOfItems, len(items))
			}
			_, err = r.Find(context.Background(), test.ID)
			if !errors.Is(err, models.ErrNotFound) {
				t.Fatalf("unexcepted error: %v", err)
			}
		})
	}
}

func TestItemRepo_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	testItem := models.Item{Name: "testNAME"}
	r := initRepo(t)
	for range 50 {
		wg.Go(func() {
			_, err := r.Create(context.Background(), testItem)
			if !errors.Is(err, models.ErrConflict) && err != nil {
				t.Errorf("unexpected error %v", err)
				return
			}
		})
	}
	wg.Wait()
	items, err := r.All(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3, received %d", len(items))
	}
}

func TestItemRepo_AllEmptyItems(t *testing.T) {
	r := initRepo(t)
	_, err := r.pool.Exec(context.Background(), `TRUNCATE items RESTART IDENTITY`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	items, err := r.All(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items == nil {
		t.Fatalf("got nil slice, want empty")
	}
	if len(items) != 0 {
		t.Fatalf("got %d instead of 0", len(items))
	}
}
