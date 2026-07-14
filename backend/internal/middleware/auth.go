package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/security"
	"github.com/google/uuid"
)

type contextKey struct{}

func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, contextKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(contextKey{}).(uuid.UUID)
	return userID, ok
}

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if authorization == "" {
				http.Error(w, "authorization header is required", http.StatusUnauthorized)
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authorization, bearerPrefix) {
				http.Error(w, "authorization header must use Bearer scheme", http.StatusUnauthorized)
				return
			}

			claims, err := security.ValidateJWT(strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix)), secret)
			if err != nil {
				if errors.Is(err, security.ErrExpiredToken) || errors.Is(err, security.ErrInvalidToken) {
					http.Error(w, "invalid or expired token", http.StatusUnauthorized)
					return
				}

				http.Error(w, "token validation failed", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), claims.UserID)))
		})
	}
}