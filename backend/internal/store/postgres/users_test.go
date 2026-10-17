package postgres

import (
	"context"
	"errors"
	"lvosa/backend/internal/models"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initUserRepo(t *testing.T) *UserRepo {
	t.Helper()
	testDbDsn := os.Getenv("TEST_DB_DSN")
	if testDbDsn == "" {
		t.Skip("TEST_DB_DSN not specified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, testDbDsn)
	if err != nil {
		t.Fatalf("DSN invalid: %v", err)
	}
	t.Cleanup(pool.Close)
	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	repo := NewUserRepo(pool)
	_, err = repo.pool.Exec(ctx, `TRUNCATE users RESTART IDENTITY`)
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}
	_, err = repo.pool.Exec(ctx, `INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3), ($4, $5, $6)`, "dima@gmail.com", "easypass", models.RoleStudent, "marjana@gmail.com", "hardpass", models.RoleAdmin)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return repo
}

func TestUserRepo_Create(t *testing.T) {
	tests := []struct {
		name     string
		user     models.User
		wantUser models.User
		wantErr  error
	}{
		{name: "valid creation with role", user: models.User{Email: "newuser@gmail.com", PasswordHash: "newpassword", Role: models.RoleAdmin}, wantUser: models.User{ID: 3, Email: "newuser@gmail.com", PasswordHash: "newpassword", Role: models.RoleAdmin}, wantErr: nil},
		{name: "exact double email creation", user: models.User{Email: "DIMA@gmail.com", PasswordHash: "newdimapass", Role: "admin"}, wantUser: models.User{}, wantErr: models.ErrConflict},
		{name: "conflict creation", user: models.User{Email: "dima@gmail.com", PasswordHash: "newdimapass", Role: "admin"}, wantUser: models.User{}, wantErr: models.ErrConflict},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := initUserRepo(t)
			user, err := repo.Create(context.Background(), test.user)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected: %v, received: %v", test.wantErr, err)
			}
			if user.ID != test.wantUser.ID {
				t.Fatalf("expected ID: %d, received: %d", test.wantUser.ID, user.ID)
			}
			if user.Email != test.wantUser.Email {
				t.Fatalf("expected email: %s, received: %s", test.wantUser.Email, user.Email)
			}
			if user.PasswordHash != test.wantUser.PasswordHash {
				t.Fatalf("expected password_hash: %s, received: %s", test.wantUser.PasswordHash, user.PasswordHash)
			}
			if user.Role != test.wantUser.Role {
				t.Fatalf("expected role: %s, received: %s", test.wantUser.Role, user.Role)
			}
			if test.wantErr == nil {
				if user.CreatedAt.IsZero() {
					t.Fatalf("created_at not set")
				}
			}
		})
	}
}

func TestUserRepo_CreateCheck(t *testing.T) {
	repo := initUserRepo(t)
	user := models.User{
		Email: "newuser@gmail.com", PasswordHash: "newpassword",
	}
	var pgErr *pgconn.PgError
	_, err := repo.Create(context.Background(), user)
	if err == nil {
		t.Fatalf("expected error, received: %v", err)
	}
	if errors.Is(err, models.ErrConflict) {
		t.Fatalf("check constraint mixed up with uniq: %v", err)
	}
	if !errors.As(err, &pgErr) {
		t.Fatalf("received not pgErr type: %v", err)
	}
	if errors.As(err, &pgErr) && pgErr.Code != checkViolation {
		t.Fatalf("expected code: 23514, received: %s", pgErr.Code)
	}
}

func TestUserRepo_FindByEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		wantUser models.User
		wantErr  error
	}{
		{name: "valid find", email: "dima@gmail.com", wantUser: models.User{ID: 1, Email: "dima@gmail.com", PasswordHash: "easypass", Role: models.RoleStudent}, wantErr: nil},
		{name: "invalid find (incorrect email)", email: "dima@gm.com", wantUser: models.User{}, wantErr: models.ErrNotFound},
		{name: "valid find (different register)", email: "DIMA@gmail.com", wantUser: models.User{ID: 1, Email: "dima@gmail.com", PasswordHash: "easypass", Role: models.RoleStudent}, wantErr: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := initUserRepo(t)
			user, err := repo.FindByEmail(context.Background(), test.email)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			if user.ID != test.wantUser.ID {
				t.Fatalf("expected ID: %d, received: %d", test.wantUser.ID, user.ID)
			}
			if user.Email != test.wantUser.Email {
				t.Fatalf("expected email: %s, received: %s", test.wantUser.Email, user.Email)
			}
			if user.PasswordHash != test.wantUser.PasswordHash {
				t.Fatalf("expected password_hash: %s, received: %s", test.wantUser.PasswordHash, user.PasswordHash)
			}
			if user.Role != test.wantUser.Role {
				t.Fatalf("expected role: %s, received: %s", test.wantUser.Role, user.Role)
			}
			if test.wantErr == nil {
				if user.CreatedAt.IsZero() {
					t.Fatalf("created_at not set")
				}
			}
		})
	}
}

func TestUserRepo_FindByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		wantUser models.User
		wantErr  error
	}{
		{name: "valid find byID", id: 1, wantUser: models.User{ID: 1, Email: "dima@gmail.com", PasswordHash: "easypass", Role: models.RoleStudent}, wantErr: nil},
		{name: "invalid find byID", id: 99, wantUser: models.User{}, wantErr: models.ErrNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := initUserRepo(t)
			user, err := repo.FindByID(context.Background(), test.id)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received: %v", test.wantErr, err)
			}
			if user.ID != test.wantUser.ID {
				t.Fatalf("expected ID: %d, received: %d", test.wantUser.ID, user.ID)
			}
			if user.Email != test.wantUser.Email {
				t.Fatalf("expected email: %s, received: %s", test.wantUser.Email, user.Email)
			}
			if user.PasswordHash != test.wantUser.PasswordHash {
				t.Fatalf("expected password_hash: %s, received: %s", test.wantUser.PasswordHash, user.PasswordHash)
			}
			if user.Role != test.wantUser.Role {
				t.Fatalf("expected role: %s, received: %s", test.wantUser.Role, user.Role)
			}
			if test.wantErr == nil {
				if user.CreatedAt.IsZero() {
					t.Fatalf("created_at not set")
				}
			}
		})
	}
}
