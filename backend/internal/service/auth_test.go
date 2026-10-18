package service

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

var _ UserRepository = (*fakeUserRepo)(nil)

type fakeUserRepo struct {
	saveUser   models.User
	returnUser models.User
	calls      int
	errLogin   error
	errCreate  error
}

func (r *fakeUserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	r.calls++
	if r.errCreate != nil {
		return models.User{}, r.errCreate
	}
	r.saveUser = user
	return r.saveUser, nil
}

func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	r.calls++
	if r.errLogin != nil {
		return models.User{}, r.errLogin
	}
	return r.returnUser, nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id int) (models.User, error) {
	return models.User{}, nil
}

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		password  string
		wantEmail string
		wantErr   error
		wantCalls int
		repoErr   error
	}{
		{name: "valid registration", email: "marjana@gmail.com", password: "password", wantEmail: "marjana@gmail.com", wantErr: nil, wantCalls: 1, repoErr: nil},
		{name: "valid registration: email with spaces", email: "  dima@gmail.com  ", password: "password", wantEmail: "dima@gmail.com", wantErr: nil, wantCalls: 1, repoErr: nil},
		{name: "invalid registration: email with name", email: "Dima <dima@gmail.com>", password: "password", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: wrong email format", email: "Dima", password: "password", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: short password", email: "marjana@gmail.com", password: "word", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: long password", email: "marjana@gmail.com", password: strings.Repeat("a", 73), wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: conflict from repo", email: "marjana@gmail.com", password: "password", wantEmail: "", wantErr: models.ErrConflict, wantCalls: 1, repoErr: fmt.Errorf("register user: %w", models.ErrConflict)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeUserRepo{errCreate: test.repoErr}
			srv, err := NewAuthService(fake)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			user, err := srv.Register(context.Background(), test.email, test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if user.Email != test.wantEmail {
				t.Fatalf("unexpected email %q, should be %q", user.Email, test.wantEmail)
			}
			if test.password != "" && test.wantErr == nil {
				err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(test.password))
				if err != nil {
					t.Fatalf("different hashed password and password: %v", err)
				}
				if user.Role != models.RoleStudent {
					t.Fatalf("expected role: %q, received: %q", models.RoleStudent, user.Role)
				}
			}
			if fake.calls != test.wantCalls {
				t.Fatalf("repo calls = %d, should be %d", fake.calls, test.wantCalls)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	errDB := errors.New("db down")
	tests := []struct {
		name      string
		email     string
		password  string
		wantEmail string
		wantErr   error
		wantCalls int
		repoErr   error
	}{
		{name: "valid log in", email: "dima@gmail.com", password: "password", wantEmail: "dima@gmail.com", wantErr: nil, wantCalls: 1, repoErr: nil},
		{name: "invalid log in: short password", email: "dima@gmail.com", password: "word", wantEmail: "", wantErr: models.ErrInvalidCredentials, wantCalls: 1, repoErr: nil},
		{name: "invalid log in: long password", email: "dima@gmail.com", password: strings.Repeat("a", 73), wantEmail: "", wantErr: models.ErrInvalidCredentials, wantCalls: 1, repoErr: nil},
		{name: "invalid log in: wrong password", email: "dima@gmail.com", password: "pass", wantEmail: "", wantErr: models.ErrInvalidCredentials, wantCalls: 1, repoErr: nil},
		{name: "invalid log in: err no email from repo", email: "dima@gmail.com", password: "password", wantEmail: "", wantErr: models.ErrInvalidCredentials, wantCalls: 1, repoErr: fmt.Errorf("find user by email: %w", models.ErrNotFound)},
		{name: "invalid log in: db down from repo", email: "dima@gmail.com", password: "password", wantEmail: "", wantErr: errDB, wantCalls: 1, repoErr: errDB},
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("create a user: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeUserRepo{errLogin: test.repoErr, returnUser: models.User{Email: "dima@gmail.com", PasswordHash: string(hashedPassword)}}
			srv, err := NewAuthService(fake)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			user, err := srv.Login(context.Background(), test.email, test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if user.Email != test.wantEmail {
				t.Fatalf("unexpected email %q, should be %q", user.Email, test.wantEmail)
			}
			if test.password != "" && test.wantErr == nil {
				err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(test.password))
				if err != nil {
					t.Fatalf("different hashed password and password: %v", err)
				}
			}
			if fake.calls != test.wantCalls {
				t.Fatalf("repo calls = %d, should be %d", fake.calls, test.wantCalls)
			}
		})
	}
}
