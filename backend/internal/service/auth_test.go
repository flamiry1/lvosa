package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/flamiry1/lvosa/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

var _ UserRepository = (*fakeUserRepo)(nil)
var _ SessionRepository = (*fakeSessionRepo)(nil)
var _ RateLimiterRepository = (*fakeRateLimiterRepo)(nil)

type fakeUserRepo struct {
	saveUser         models.User
	returnUser       models.User
	createCalls      int
	errCreate        error
	findByEmailCalls int
	errFindByEmail   error
	findByIDCalls    int
	errFindByID      error
	lastID           int
}

type fakeSessionRepo struct {
	createCalls  int
	errCreate    error
	createToken  string
	lastID       int
	getCalls     int
	errGet       error
	getToken     string
	returnUserID int
	deleteCalls  int
	errDelete    error
	deleteToken  string
}

type fakeRateLimiterRepo struct {
	increaseRateCalls int
	dropRateCalls     int
	returnRate        int
	increaseRateErr   error
	dropRateErr       error
}

func (r *fakeUserRepo) Create(ctx context.Context, user models.User) (models.User, error) {
	r.createCalls++
	if r.errCreate != nil {
		return models.User{}, r.errCreate
	}
	r.saveUser = user
	return r.saveUser, nil
}

func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	r.findByEmailCalls++
	if r.errFindByEmail != nil {
		return models.User{}, r.errFindByEmail
	}
	return r.returnUser, nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id int) (models.User, error) {
	r.findByIDCalls++
	r.lastID = id
	if r.errFindByID != nil {
		return models.User{}, r.errFindByID
	}
	return r.returnUser, nil
}

func (r *fakeSessionRepo) Create(ctx context.Context, userID int) (string, error) {
	r.createCalls++
	if r.errCreate != nil {
		return "", r.errCreate
	}
	r.lastID = userID
	return r.createToken, nil
}

func (r *fakeSessionRepo) Get(ctx context.Context, token string) (int, error) {
	r.getCalls++
	r.getToken = token
	if r.errGet != nil {
		return 0, r.errGet
	}
	return r.returnUserID, nil
}

func (r *fakeSessionRepo) Delete(ctx context.Context, token string) error {
	r.deleteCalls++
	if r.errDelete != nil {
		return r.errDelete
	}
	r.deleteToken = token
	return nil
}

func (r *fakeRateLimiterRepo) IncreaseRate(ctx context.Context, email string) (int, error) {
	r.increaseRateCalls++
	if r.increaseRateErr != nil {
		return 0, r.increaseRateErr
	}
	return r.returnRate, nil
}

func (r *fakeRateLimiterRepo) DropRate(ctx context.Context, email string) error {
	r.dropRateCalls++
	if r.dropRateErr != nil {
		return r.dropRateErr
	}
	return nil
}

func TestAuthService_Register(t *testing.T) {
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
		{name: "valid registration", email: "marjana@gmail.com", password: "password", wantEmail: "marjana@gmail.com", wantErr: nil, wantCalls: 1, repoErr: nil},
		{name: "valid registration: email with spaces", email: "  dima@gmail.com  ", password: "password", wantEmail: "dima@gmail.com", wantErr: nil, wantCalls: 1, repoErr: nil},
		{name: "invalid registration: email with name", email: "Dima <dima@gmail.com>", password: "password", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: wrong email format", email: "Dima", password: "password", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: short password", email: "marjana@gmail.com", password: "word", wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: long password", email: "marjana@gmail.com", password: strings.Repeat("a", 73), wantEmail: "", wantErr: models.ErrBadRequest, wantCalls: 0, repoErr: nil},
		{name: "invalid registration: conflict from repo", email: "marjana@gmail.com", password: "password", wantEmail: "", wantErr: models.ErrConflict, wantCalls: 1, repoErr: fmt.Errorf("register user: %w", models.ErrConflict)},
		{name: "invalid registration: db down", email: "marjana@gmail.com", password: "password", wantEmail: "", wantErr: errDB, wantCalls: 1, repoErr: errDB},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeURepo := &fakeUserRepo{errCreate: test.repoErr}
			fakeSRepo := &fakeSessionRepo{}
			fakeRateRepo := &fakeRateLimiterRepo{}
			srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, 5)
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
				err = bcrypt.CompareHashAndPassword([]byte(fakeURepo.saveUser.PasswordHash), []byte(test.password))
				if err != nil {
					t.Fatalf("different hashed password and password: %v", err)
				}
				if user.Role != models.RoleStudent {
					t.Fatalf("expected role: %q, received: %q", models.RoleStudent, user.Role)
				}
			}
			if user.PasswordHash != "" {
				t.Fatalf("expected empty hash password, received: %s", user.PasswordHash)
			}
			if fakeURepo.createCalls != test.wantCalls {
				t.Fatalf("repo calls = %d, should be %d", fakeURepo.createCalls, test.wantCalls)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	errDB := errors.New("db down")
	errRDB := errors.New("rdb down")
	tests := []struct {
		name                 string
		email                string
		password             string
		wantEmail            string
		findByEmailWantCalls int
		createWantCalls      int
		wantToken            string
		returnID             int
		repoPGErr            error
		wantErr              error
		repoRedisErr         error
	}{
		{name: "valid log in", email: "dima@gmail.com", password: "password", wantEmail: "dima@gmail.com", wantErr: nil, findByEmailWantCalls: 1, createWantCalls: 1, repoPGErr: nil, wantToken: "test-token", returnID: 7},
		{name: "invalid log in: short password", email: "dima@gmail.com", password: "word", wantEmail: "", wantErr: models.ErrInvalidCredentials, findByEmailWantCalls: 1, repoPGErr: nil},
		{name: "invalid log in: err no email from repo", email: "dima@gmail.com", password: "password", wantEmail: "", wantErr: models.ErrInvalidCredentials, findByEmailWantCalls: 1, repoPGErr: fmt.Errorf("find user by email: %w", models.ErrNotFound)},
		{name: "invalid log in: db down from repo", email: "dima@gmail.com", password: "password", wantEmail: "", wantErr: errDB, findByEmailWantCalls: 1, repoPGErr: errDB},
		{name: "valid log in (session): valid return token", email: "dima@gmail.com", password: "password", wantEmail: "dima@gmail.com", wantErr: nil, findByEmailWantCalls: 1, createWantCalls: 1, repoPGErr: nil, wantToken: "test-token", returnID: 7},
		{name: "invalid log in (session): invalid password", email: "dima@gmail.com", password: "pass", wantEmail: "", wantErr: models.ErrInvalidCredentials, findByEmailWantCalls: 1, createWantCalls: 0, repoPGErr: nil, wantToken: ""},
		{name: "invalid log in (session): no email", email: "", password: "password", wantEmail: "", wantErr: models.ErrBadRequest, findByEmailWantCalls: 0, createWantCalls: 0, repoPGErr: nil, wantToken: ""},
		{name: "invalid log in (session): db down", email: "dima@gmail.com", password: "password", wantEmail: "", wantErr: errRDB, findByEmailWantCalls: 1, createWantCalls: 1, repoPGErr: nil, wantToken: "", repoRedisErr: errRDB},
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("create a user: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeSRepo := &fakeSessionRepo{createToken: "test-token", errCreate: test.repoRedisErr, returnUserID: test.returnID}
			fakeURepo := &fakeUserRepo{errFindByEmail: test.repoPGErr, returnUser: models.User{Email: "dima@gmail.com", PasswordHash: string(hashedPassword), ID: 7}}
			fakeRateRepo := &fakeRateLimiterRepo{returnRate: 1}
			srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, 5)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			user, token, err := srv.Login(context.Background(), test.email, test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if user.Email != test.wantEmail {
				t.Fatalf("unexpected email %q, should be %q", user.Email, test.wantEmail)
			}
			if user.PasswordHash != "" {
				t.Fatalf("expected empty hash password, received: %s", user.PasswordHash)
			}
			if fakeURepo.findByEmailCalls != test.findByEmailWantCalls {
				t.Fatalf("users repo calls = %d, should be %d", fakeURepo.findByEmailCalls, test.findByEmailWantCalls)
			}
			if fakeSRepo.createCalls != test.createWantCalls {
				t.Fatalf("session repo calls = %d, should be %d", fakeSRepo.createCalls, test.createWantCalls)
			}
			if token != test.wantToken {
				t.Fatalf("create made different token, received: %s, wanted: %s", fakeSRepo.createToken, test.wantToken)
			}
			if user.ID != fakeSRepo.lastID {
				t.Fatalf("expected ID: %d, received: %d", fakeURepo.lastID, user.ID)
			}
		})
	}
}

func TestAuthService_LogoutValid(t *testing.T) {
	fakeSRepo := &fakeSessionRepo{}
	fakeURepo := &fakeUserRepo{}
	fakeRateRepo := &fakeRateLimiterRepo{}
	srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, 5)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	err = srv.Logout(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fakeSRepo.deleteToken != "test-token" {
		t.Fatalf("expected token: test-token, received: %s", fakeSRepo.deleteToken)
	}

}

func TestAuthService_LogoutInValid(t *testing.T) {
	errRDB := errors.New("rdb down")
	fakeSRepo := &fakeSessionRepo{errDelete: errRDB}
	fakeURepo := &fakeUserRepo{}
	fakeRateRepo := &fakeRateLimiterRepo{}
	srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, 5)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	err = srv.Logout(context.Background(), "test-token")
	if !errors.Is(err, errRDB) {
		t.Fatalf("expected error: %v, received error: %v", errRDB, err)
	}
}

func TestAuthService_CurrentUser(t *testing.T) {
	rDBErr := errors.New("rdb down")
	dbErr := errors.New("db down")
	tests := []struct {
		name              string
		getErr            error
		findByIDErr       error
		token             string
		wantErr           error
		wantFindByIDCalls int
		wantUserID        int
	}{
		{name: "valid get current user", token: "test-token", findByIDErr: nil, wantErr: nil, wantFindByIDCalls: 1, wantUserID: 77, getErr: nil},
		{name: "get returned not found", token: "test-token", findByIDErr: nil, wantErr: models.ErrUnauthorized, wantFindByIDCalls: 0, wantUserID: 0, getErr: models.ErrNotFound},
		{name: "findByID return not found", token: "test-token", findByIDErr: models.ErrNotFound, wantErr: models.ErrUnauthorized, wantFindByIDCalls: 1, wantUserID: 0, getErr: nil},
		{name: "redis error", token: "test-token", findByIDErr: nil, wantErr: rDBErr, wantFindByIDCalls: 0, wantUserID: 0, getErr: rDBErr},
		{name: "postgre error", token: "test-token", findByIDErr: dbErr, wantErr: dbErr, wantFindByIDCalls: 1, wantUserID: 0, getErr: nil},
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("create a user: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeSRepo := &fakeSessionRepo{returnUserID: 77, errGet: test.getErr}
			fakeURepo := &fakeUserRepo{returnUser: models.User{ID: 77, Email: "dimas@bog.mail", PasswordHash: string(hashedPassword)}, errFindByID: test.findByIDErr}
			fakeRateRepo := &fakeRateLimiterRepo{}
			srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, 5)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			user, err := srv.CurrentUser(context.Background(), test.token)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("expected error: %v, received error: %v", test.wantErr, err)
			}
			if user.ID != test.wantUserID {
				t.Fatalf("expected ID: %d, received: %d", test.wantUserID, user.ID)
			}
			if user.PasswordHash != "" {
				t.Fatalf("expected empty hash password, received: %s", user.PasswordHash)
			}
			if fakeSRepo.getToken != test.token {
				t.Fatalf("expected token: %s, received: %s", test.token, fakeSRepo.getToken)
			}
			if fakeURepo.findByIDCalls != test.wantFindByIDCalls {
				t.Fatalf("expected FindBy call: %d, received: %d", test.wantFindByIDCalls, fakeURepo.findByIDCalls)
			}
			if test.wantFindByIDCalls > 0 {
				if fakeURepo.lastID != 77 {
					t.Fatalf("expected lastID: 77, received: %d", fakeURepo.lastID)
				}
			}
		})
	}
}

func TestAuthService_LoginRateLimiter(t *testing.T) {
	const limit = 5
	rDBErr := errors.New("rdb down")
	tests := []struct {
		name                   string
		email                  string
		password               string
		rate                   int
		increaseRateErr        error
		dropRateErr            error
		findByEmailErr         error
		wantErr                error
		wantIncreaseRateCalls  int
		wantFindByEmailCalls   int
		wantDropRateCalls      int
		wantSessionCreateCalls int
	}{
		{name: "valid login: first attempt", email: "dima@gmail.com", password: "password", rate: 1, wantErr: nil, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 1, wantSessionCreateCalls: 1},
		{name: "valid login: last allowed attempt", email: "dima@gmail.com", password: "password", rate: limit, wantErr: nil, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 1, wantSessionCreateCalls: 1},
		{name: "invalid login: last allowed attempt, password invalid", email: "dima@gmail.com", password: "pass", rate: limit, wantErr: models.ErrInvalidCredentials, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "blocked: first attempt over limit, password valid", email: "dima@gmail.com", password: "password", rate: limit + 1, wantErr: models.ErrTooManyRequests, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 0, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "blocked: first attempt over limit, password invalid", email: "dima@gmail.com", password: "pass", rate: limit + 1, wantErr: models.ErrTooManyRequests, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 0, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "blocked: far over limit after burst", email: "dima@gmail.com", password: "password", rate: 50, wantErr: models.ErrTooManyRequests, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 0, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "invalid login: password invalid", email: "dima@gmail.com", password: "pass", rate: 1, wantErr: models.ErrInvalidCredentials, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "invalid login: no such email", email: "d@gmail.com", password: "password", rate: 1, findByEmailErr: models.ErrNotFound, wantErr: models.ErrInvalidCredentials, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "invalid login: bad email format not counted", email: "dima", password: "password", rate: 1, wantErr: models.ErrBadRequest, wantIncreaseRateCalls: 0, wantFindByEmailCalls: 0, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "failed login: rdb down while increasing rate", email: "dima@gmail.com", password: "password", increaseRateErr: rDBErr, wantErr: rDBErr, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 0, wantDropRateCalls: 0, wantSessionCreateCalls: 0},
		{name: "valid login: rdb down while dropping rate", email: "dima@gmail.com", password: "password", rate: 1, dropRateErr: rDBErr, wantErr: nil, wantIncreaseRateCalls: 1, wantFindByEmailCalls: 1, wantDropRateCalls: 1, wantSessionCreateCalls: 1},
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("create a user: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeSRepo := &fakeSessionRepo{}
			fakeURepo := &fakeUserRepo{returnUser: models.User{ID: 77, Email: "dima@gmail.com", PasswordHash: string(hashedPassword)}, errFindByEmail: test.findByEmailErr}
			fakeRateRepo := &fakeRateLimiterRepo{returnRate: test.rate, increaseRateErr: test.increaseRateErr, dropRateErr: test.dropRateErr}
			srv, err := NewAuthService(fakeURepo, fakeSRepo, fakeRateRepo, limit)
			if err != nil {
				t.Fatalf("create service: %v", err)
			}
			_, _, err = srv.Login(context.Background(), test.email, test.password)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("want error: %v, received: %v", test.wantErr, err)
			}
			if fakeRateRepo.increaseRateCalls != test.wantIncreaseRateCalls {
				t.Fatalf("expected increase rate calls: %d, received: %d", test.wantIncreaseRateCalls, fakeRateRepo.increaseRateCalls)
			}
			if fakeURepo.findByEmailCalls != test.wantFindByEmailCalls {
				t.Fatalf("expected find by email calls: %d, received: %d", test.wantFindByEmailCalls, fakeURepo.findByEmailCalls)
			}
			if fakeRateRepo.dropRateCalls != test.wantDropRateCalls {
				t.Fatalf("expected drop rate calls: %d, received: %d", test.wantDropRateCalls, fakeRateRepo.dropRateCalls)
			}
			if fakeSRepo.createCalls != test.wantSessionCreateCalls {
				t.Fatalf("expected session create calls: %d, received: %d", test.wantSessionCreateCalls, fakeSRepo.createCalls)
			}
		})
	}
}
