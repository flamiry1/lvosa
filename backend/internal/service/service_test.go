package service

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"strings"
	"testing"
)

type fakeRepo struct {
	lastItem models.Item
	calls    int
	err      error
	lastId   int
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		item      models.Item
		wantErr   error
		wantCalls int
		wantName  string
		repoErr   error
	}{
		{name: "conflict from repo", item: models.Item{Name: "dimondr"}, wantErr: models.ErrConflict, wantCalls: 1, wantName: "dimondr", repoErr: fmt.Errorf("%w: name already registered", models.ErrConflict)},
		{name: "empty name", item: models.Item{Name: "  "}, wantErr: models.ErrBadRequest, wantCalls: 0, wantName: ""},
		{name: "1 symbol name", item: models.Item{Name: "a"}, wantErr: nil, wantCalls: 1, wantName: "a"},
		{name: "name with spaces", item: models.Item{Name: "   dimas  "}, wantErr: nil, wantCalls: 1, wantName: "dimas"},
		{name: "valid name", item: models.Item{Name: "dimas"}, wantErr: nil, wantCalls: 1, wantName: "dimas"},
		{name: "100 symbols of Latin name", item: models.Item{Name: strings.Repeat("a", 100)}, wantErr: nil, wantCalls: 1, wantName: strings.Repeat("a", 100)},
		{name: "101 symbols of Latin name", item: models.Item{Name: strings.Repeat("a", 101)}, wantErr: models.ErrBadRequest, wantCalls: 0, wantName: ""},
		{name: "100 symbols of Slovenian name", item: models.Item{Name: strings.Repeat("č", 100)}, wantErr: nil, wantCalls: 1, wantName: strings.Repeat("č", 100)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRepo{err: test.repoErr}
			svc := New(fake)
			_, err := svc.Create(context.Background(), test.item)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if fake.calls != test.wantCalls {
				t.Fatalf("repo calls = %d, should be %d", fake.calls, test.wantCalls)
			}
			if fake.lastItem.Name != test.wantName {
				t.Fatalf("unexpected name %q, should be %q", fake.lastItem.Name, test.wantName)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name       string
		item       models.Item
		id         int
		wantErr    error
		wantName   string
		wantCalls  int
		wantLastId int
		repoErr    error
	}{
		{name: "conflict from repo", item: models.Item{Name: "BOSS"}, id: 10, wantErr: models.ErrConflict, wantCalls: 1, wantName: "BOSS", repoErr: fmt.Errorf("%w: name already registered", models.ErrConflict), wantLastId: 10},
		{name: "not found from repo", item: models.Item{Name: "BOSS"}, id: 11, wantErr: models.ErrNotFound, wantCalls: 1, wantName: "BOSS", repoErr: fmt.Errorf("%w: name not found", models.ErrNotFound), wantLastId: 11},
		{name: "empty name", item: models.Item{Name: "  "}, id: 1, wantErr: models.ErrBadRequest, wantName: "", wantCalls: 0, wantLastId: 0},
		{name: "valid name", item: models.Item{Name: "dimas"}, id: 2, wantErr: nil, wantName: "dimas", wantCalls: 1, wantLastId: 2},
		{name: "name with spaces", item: models.Item{Name: "  dimas  "}, id: 3, wantErr: nil, wantName: "dimas", wantCalls: 1, wantLastId: 3},
		{name: "1 symbol name", item: models.Item{Name: "a"}, id: 4, wantErr: nil, wantName: "a", wantCalls: 1, wantLastId: 4},
		{name: "100 symbols of Latin name", item: models.Item{Name: strings.Repeat("a", 100)}, id: 5, wantErr: nil, wantName: strings.Repeat("a", 100), wantCalls: 1, wantLastId: 5},
		{name: "101 symbols of Latin name", item: models.Item{Name: strings.Repeat("a", 101)}, id: 6, wantErr: models.ErrBadRequest, wantName: "", wantCalls: 0, wantLastId: 0},
		{name: "100 symbols of Slovenian name", item: models.Item{Name: strings.Repeat("č", 100)}, id: 7, wantErr: nil, wantName: strings.Repeat("č", 100), wantCalls: 1, wantLastId: 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRepo{err: test.repoErr}
			svc := New(fake)
			err := svc.Update(context.Background(), test.item, test.id)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if fake.calls != test.wantCalls {
				t.Fatalf("repo calls = %d, should be %d", fake.calls, test.wantCalls)
			}
			if fake.lastId != test.wantLastId {
				t.Fatalf("expected id: %d, received: %d", test.wantLastId, fake.lastId)
			}
			if fake.lastItem.Name != test.wantName {
				t.Fatalf("unexpected name %q, should be %q", fake.lastItem.Name, test.wantName)
			}
		})

	}
}

func TestServiceList(t *testing.T) {
	errDB := errors.New("db down")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "repo ok", wantErr: nil, repoErr: nil},
		{name: "repo failed", wantErr: errDB, repoErr: errDB},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRepo{err: test.repoErr}
			s := New(fake)
			_, err := s.List(context.Background())
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("unexpected error %v, want %v", err, test.wantErr)
			}
		})
	}
}

func (f *fakeRepo) Create(ctx context.Context, n models.Item) (models.Item, error) {
	f.lastItem = n
	f.calls++
	if f.err != nil {
		return models.Item{}, f.err
	}
	return n, nil

}
func (f *fakeRepo) Update(ctx context.Context, id int, n models.Item) error {
	f.lastItem = n
	f.lastId = id
	f.calls++
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeRepo) Find(ctx context.Context, id int) (models.Item, error) {
	return models.Item{}, models.ErrNotFound
}

func (f *fakeRepo) Delete(ctx context.Context, id int) error {
	return models.ErrNotFound
}

func (f *fakeRepo) All(ctx context.Context) ([]models.Item, error) {
	return []models.Item{}, f.err
}
