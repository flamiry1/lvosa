package service

import (
	"context"
	"errors"
	"fmt"
	"lvosa/backend/internal/models"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
	FindByID(ctx context.Context, id int) (models.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, userID int) (string, error)
	Get(ctx context.Context, token string) (int, error)
	Delete(ctx context.Context, token string) error
}
type AuthService struct {
	usersRepo   UserRepository
	sessionRepo SessionRepository
	stopPass    []byte
}

func NewAuthService(r UserRepository, s SessionRepository) (*AuthService, error) {
	hashedStopPass, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generate stopPass: %w", err)
	}
	return &AuthService{usersRepo: r, sessionRepo: s, stopPass: hashedStopPass}, nil
}

func checkEmail(email string) (string, error) {
	strippedEmail := strings.TrimSpace(email)
	parsedEmail, err := mail.ParseAddress(strippedEmail)
	if err != nil {
		return "", fmt.Errorf("%w: failed to parse email", models.ErrBadRequest)
	}
	if strippedEmail != parsedEmail.Address {
		return "", fmt.Errorf("%w: failed to parse email", models.ErrBadRequest)
	}
	return parsedEmail.Address, nil
}

func (a *AuthService) Register(ctx context.Context, email, password string) (models.User, error) {
	checkedEmail, err := checkEmail(email)
	if err != nil {
		return models.User{}, err
	}
	if utf8.RuneCountInString(password) < 8 {
		return models.User{}, fmt.Errorf("%w: password should not be less than 8 symbols", models.ErrBadRequest)
	}
	if len(password) > 72 {
		return models.User{}, fmt.Errorf("%w: password should not be longer than 72 bytes", models.ErrBadRequest)
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := a.usersRepo.Create(ctx, models.User{Email: checkedEmail, PasswordHash: string(hashedPassword), Role: models.RoleStudent})
	if err != nil {
		return models.User{}, fmt.Errorf("register user: %w", err)
	}
	user.PasswordHash = ""
	return user, nil
}

func (a *AuthService) Login(ctx context.Context, email, password string) (models.User, string, error) {
	checkedEmail, err := checkEmail(email)
	if err != nil {
		return models.User{}, "", err
	}
	user, err := a.usersRepo.FindByEmail(ctx, checkedEmail)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword(a.stopPass, []byte("password"))
			return models.User{}, "", fmt.Errorf("%w: no email", models.ErrInvalidCredentials)
		}
		return models.User{}, "", fmt.Errorf("find user: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return models.User{}, "", models.ErrInvalidCredentials
	}
	user.PasswordHash = ""
	token, err := a.sessionRepo.Create(ctx, user.ID)
	if err != nil {
		return models.User{}, "", fmt.Errorf("create session: %w", err)
	}
	return user, token, nil
}

func (a *AuthService) Logout(ctx context.Context, token string) error {
	err := a.sessionRepo.Delete(ctx, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (a *AuthService) CurrentUser(ctx context.Context, token string) (models.User, error) {
	userID, err := a.sessionRepo.Get(ctx, token)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.User{}, models.ErrUnauthorized
		}
		return models.User{}, fmt.Errorf("get session: %w", err)
	}
	user, err := a.usersRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.User{}, models.ErrUnauthorized
		}
		return models.User{}, fmt.Errorf("get user by ID: %w", err)
	}
	user.PasswordHash = ""
	return user, nil
}
