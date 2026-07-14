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
	"github.com/google/uuid"
)

var (
	ErrUserAlreadyExists   = errors.New("user already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidAuthInput    = errors.New("invalid auth input")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type AuthService struct {
	users           *repository.UserRepository
	refreshTokens   *repository.RefreshTokenRepository
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthService(users *repository.UserRepository, refreshTokens *repository.RefreshTokenRepository, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *AuthService {
	return &AuthService{
		users:           users,
		refreshTokens:   refreshTokens,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, login, password string) (*models.User, string, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.TrimSpace(password) == "" {
		return nil, "", "", ErrInvalidAuthInput
	}

	_, err := s.users.GetUserByLogin(ctx, login)
	if err == nil {
		return nil, "", "", ErrUserAlreadyExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, "", "", fmt.Errorf("get user by login: %w", err)
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.CreateUser(ctx, login, passwordHash)
	if err != nil {
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*models.User, string, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || strings.TrimSpace(password) == "" {
		return nil, "", "", ErrInvalidAuthInput
	}

	user, err := s.users.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("get user by login: %w", err)
	}

	if err := security.CheckPassword(password, user.PasswordHash); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	accessToken, refreshToken, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.User, string, string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, "", "", ErrInvalidRefreshToken
	}
	if s.refreshTokens == nil {
		return nil, "", "", fmt.Errorf("refresh token repository is not configured")
	}

	tokenHash := security.HashRefreshToken(refreshToken)
	storedToken, err := s.refreshTokens.GetValidRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, "", "", ErrInvalidRefreshToken
	}

	user, err := s.users.GetUserByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, "", "", fmt.Errorf("get user by id: %w", err)
	}

	if err := s.refreshTokens.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return nil, "", "", fmt.Errorf("revoke refresh token: %w", err)
	}

	accessToken, nextRefreshToken, err := s.issueTokens(ctx, user.ID)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, nextRefreshToken, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ErrInvalidRefreshToken
	}
	if s.refreshTokens == nil {
		return fmt.Errorf("refresh token repository is not configured")
	}

	tokenHash := security.HashRefreshToken(refreshToken)
	storedToken, err := s.refreshTokens.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return ErrInvalidRefreshToken
	}

	if err := s.refreshTokens.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, userID uuid.UUID) (string, string, error) {
	if s.refreshTokens == nil {
		return "", "", fmt.Errorf("refresh token repository is not configured")
	}

	accessToken, err := security.GenerateJWT(userID, s.jwtSecret, s.accessTokenTTL)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	refreshTokenHash := security.HashRefreshToken(refreshToken)
	if _, err := s.refreshTokens.CreateRefreshToken(ctx, userID, refreshTokenHash, time.Now().UTC().Add(s.refreshTokenTTL)); err != nil {
		return "", "", fmt.Errorf("create refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}
