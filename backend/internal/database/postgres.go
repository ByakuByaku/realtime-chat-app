package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const pingTimeout = 5 * time.Second

func OpenPostgres(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DBAddress)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(25)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}