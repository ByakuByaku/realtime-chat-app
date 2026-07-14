package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/security"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAuthInput   = errors.New("invalid auth input")
)

type AuthService struct {
	users     *repository.UserRepository
	jwtSecret string
	tokenTTL  time.Duration
}

func NewAuthService(users *repository.UserRepository, jwtSecret string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		users:     users,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*models.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.TrimSpace(password) == "" {
		return nil, "", ErrInvalidAuthInput
	}

	_, err := s.users.GetUserByLogin(ctx, login)
	if err == nil {
		return nil, "", ErrUserAlreadyExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, "", fmt.Errorf("get user by login: %w", err)
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, login, passwordHash)
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := security.GenerateJWT(user.ID, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*models.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.TrimSpace(password) == "" {
		return nil, "", ErrInvalidAuthInput
	}

	user, err := s.users.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("get user by login: %w", err)
	}

	if err := security.CheckPassword(password, user.PasswordHash); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := security.GenerateJWT(user.ID, s.jwtSecret, s.tokenTTL)
	if err != nil {
		return nil, "", fmt.Errorf("generate access token: %w", err)
	}

	return user, token, nil
}
