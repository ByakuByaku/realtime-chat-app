package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/config"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/database"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/middleware"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
	httptransport "github.com/ByakuByaku/realtime-chat-app/backend/internal/transport/http"
	wstransport "github.com/ByakuByaku/realtime-chat-app/backend/internal/transport/websocket"
)

const (
	shutdownTimeout = 10 * time.Second
)

type App struct {
	config config.Config
	db     *sql.DB
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	db, err := database.OpenPostgres(cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	usersRepo := repository.NewUserRepository(db)
	refreshTokensRepo := repository.NewRefreshTokenRepository(db)
	chatsRepo := repository.NewChatRepository(db)
	messagesRepo := repository.NewMessageRepository(db)

	authService := service.NewAuthService(usersRepo, refreshTokensRepo, cfg.JWTSecret, cfg.TokenTTL, cfg.RefreshTokenTTL)
	chatService := service.NewChatService(chatsRepo)
	messageService := service.NewMessageService(messagesRepo)

	httpServer := httptransport.NewServer(authService, chatService, messageService)

	hub := wstransport.NewHub()
	go hub.Run()
	wsServer := wstransport.NewServer(hub, chatService, messageService, cfg.JWTSecret)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/api/v1/auth/", httpServer.PublicHandler())
	mux.HandleFunc("GET /api/v1/chats/{chat_id}/ws", wsServer.HandleChatSocket)
	mux.Handle("/api/v1/", middleware.Auth(cfg.JWTSecret)(httpServer.ProtectedHandler()))

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	return &App{
		config: cfg,
		db:     db,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := a.server.Shutdown(shutdownCtx); err != nil {
			_ = a.db.Close()
			return fmt.Errorf("shutdown server: %w", err)
		}

		_ = a.db.Close()
		return nil
	case err := <-serverErr:
		_ = a.db.Close()
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("run server: %w", err)
		}
		return nil
	}
}
