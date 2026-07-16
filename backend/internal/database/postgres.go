package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

	if err := runMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migrationsDir, err := migrationsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, fileName := range files {
		var applied bool
		if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, fileName).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", fileName, err)
		}
		if applied {
			continue
		}

		path := filepath.Join(migrationsDir, fileName)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fileName, err)
		}

		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", fileName, err)
		}

		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, fileName); err != nil {
			return fmt.Errorf("record migration %s: %w", fileName, err)
		}
	}

	return nil
}

func migrationsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve migrations path")
	}
	return filepath.Join(filepath.Dir(currentFile), "migrations"), nil
}
