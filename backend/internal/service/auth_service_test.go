package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/security"
	"github.com/google/uuid"
)

type testStore struct {
	mu           sync.Mutex
	usersByID    map[uuid.UUID]testUserRow
	usersByLogin map[string]testUserRow
	refreshByID  map[uuid.UUID]testRefreshRow
	refreshByHash map[string]testRefreshRow
}

type testUserRow struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	CreatedAt    time.Time
}

type testRefreshRow struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func newTestStore() *testStore {
	return &testStore{
		usersByID:     make(map[uuid.UUID]testUserRow),
		usersByLogin:  make(map[string]testUserRow),
		refreshByID:   make(map[uuid.UUID]testRefreshRow),
		refreshByHash: make(map[string]testRefreshRow),
	}
}

func openTestDB(t *testing.T, store *testStore) (*sql.DB, func()) {
	t.Helper()

	driverName := fmt.Sprintf("auth-test-%d", time.Now().UnixNano())
	sql.Register(driverName, &testDriver{store: store})

	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup
}

type testDriver struct {
	store *testStore
}

func (d *testDriver) Open(name string) (driver.Conn, error) {
	return &testConn{store: d.store}, nil
}

type testConn struct {
	store *testStore
}

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (c *testConn) Close() error { return nil }

func (c *testConn) Begin() (driver.Tx, error) { return nil, errors.New("transactions not supported") }

func (c *testConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.exec(query, args)
}

func (c *testConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.query(query, args)
}

func (c *testConn) exec(query string, args []driver.NamedValue) (driver.Result, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	switch {
	case strings.Contains(query, "INSERT INTO users"):
		row := testUserRow{
			ID:           mustUUID(args[0].Value),
			Login:        args[1].Value.(string),
			PasswordHash: args[2].Value.(string),
			CreatedAt:    args[3].Value.(time.Time),
		}
		c.store.usersByID[row.ID] = row
		c.store.usersByLogin[row.Login] = row
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "INSERT INTO refresh_tokens"):
		row := testRefreshRow{
			ID:        mustUUID(args[0].Value),
			UserID:    mustUUID(args[1].Value),
			TokenHash: args[2].Value.(string),
			ExpiresAt: args[3].Value.(time.Time),
			Revoked:   args[4].Value.(bool),
			CreatedAt: args[5].Value.(time.Time),
		}
		c.store.refreshByID[row.ID] = row
		c.store.refreshByHash[row.TokenHash] = row
		return driver.RowsAffected(1), nil
	case strings.Contains(query, "UPDATE refresh_tokens"):
		id := mustUUID(args[0].Value)
		row, ok := c.store.refreshByID[id]
		if !ok {
			return driver.RowsAffected(0), nil
		}
		row.Revoked = true
		c.store.refreshByID[id] = row
		c.store.refreshByHash[row.TokenHash] = row
		return driver.RowsAffected(1), nil
	default:
		return nil, fmt.Errorf("unsupported exec query: %s", query)
	}
}

func (c *testConn) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	c.store.mu.Lock()
	defer c.store.mu.Unlock()

	switch {
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE login = $1"):
		login := args[0].Value.(string)
		row, ok := c.store.usersByLogin[login]
		if !ok {
			return &testRows{cols: []string{"id", "login", "password_hash", "created_at"}}, nil
		}
		return rowsFromUser(row), nil
	case strings.Contains(query, "FROM users") && strings.Contains(query, "WHERE id = $1"):
		id := mustUUID(args[0].Value)
		row, ok := c.store.usersByID[id]
		if !ok {
			return &testRows{cols: []string{"id", "login", "password_hash", "created_at"}}, nil
		}
		return rowsFromUser(row), nil
	case strings.Contains(query, "FROM refresh_tokens") && strings.Contains(query, "WHERE token_hash = $1 AND revoked = false AND expires_at > now()"):
		hash := args[0].Value.(string)
		row, ok := c.store.refreshByHash[hash]
		if !ok || row.Revoked || !row.ExpiresAt.After(time.Now().UTC()) {
			return &testRows{cols: []string{"id", "user_id", "token_hash", "expires_at", "revoked", "created_at"}}, nil
		}
		return rowsFromRefresh(row), nil
	case strings.Contains(query, "FROM refresh_tokens") && strings.Contains(query, "WHERE token_hash = $1"):
		hash := args[0].Value.(string)
		row, ok := c.store.refreshByHash[hash]
		if !ok {
			return &testRows{cols: []string{"id", "user_id", "token_hash", "expires_at", "revoked", "created_at"}}, nil
		}
		return rowsFromRefresh(row), nil
	default:
		return nil, fmt.Errorf("unsupported query: %s", query)
	}
}

func mustUUID(value any) uuid.UUID {
	switch typed := value.(type) {
	case uuid.UUID:
		return typed
	case string:
		parsed, err := uuid.Parse(typed)
		if err != nil {
			panic(err)
		}
		return parsed
	default:
		panic(fmt.Sprintf("unsupported uuid value type %T", value))
	}
}

type testRows struct {
	cols []string
	data [][]driver.Value
	idx  int
}

func rowsFromUser(row testUserRow) driver.Rows {
	return &testRows{
		cols: []string{"id", "login", "password_hash", "created_at"},
		data: [][]driver.Value{{row.ID.String(), row.Login, row.PasswordHash, row.CreatedAt}},
	}
}

func rowsFromRefresh(row testRefreshRow) driver.Rows {
	return &testRows{
		cols: []string{"id", "user_id", "token_hash", "expires_at", "revoked", "created_at"},
		data: [][]driver.Value{{row.ID.String(), row.UserID.String(), row.TokenHash, row.ExpiresAt, row.Revoked, row.CreatedAt}},
	}
}

func (r *testRows) Columns() []string { return r.cols }

func (r *testRows) Close() error { return nil }

func (r *testRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	row := r.data[r.idx]
	for i := range row {
		dest[i] = row[i]
	}
	r.idx++
	return nil
}

func newTestAuthService(t *testing.T, store *testStore) (*AuthService, func()) {
	t.Helper()

	db, cleanup := openTestDB(t, store)
	usersRepo := repository.NewUserRepository(db)
	refreshRepo := repository.NewRefreshTokenRepository(db)
	service := NewAuthService(usersRepo, refreshRepo, "test-secret", time.Hour, 24*time.Hour)

	return service, cleanup
}

func TestAuthServiceRegister(t *testing.T) {
	store := newTestStore()
	svc, cleanup := newTestAuthService(t, store)
	defer cleanup()

	user, accessToken, refreshToken, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if user.Login != "alice" {
		t.Fatalf("expected login alice, got %q", user.Login)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("expected tokens to be returned")
	}

	claims, err := security.ValidateJWT(accessToken, "test-secret")
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected access token user id %s, got %s", user.ID, claims.UserID)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	hash := security.HashRefreshToken(refreshToken)
	row, ok := store.refreshByHash[hash]
	if !ok {
		t.Fatal("expected refresh token to be stored")
	}
	if row.UserID != user.ID {
		t.Fatalf("expected refresh token user id %s, got %s", user.ID, row.UserID)
	}
	if row.Revoked {
		t.Fatal("expected refresh token to be active")
	}
}

func TestAuthServiceLogin(t *testing.T) {
	store := newTestStore()
	passwordHash, err := security.HashPassword("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := testUserRow{
		ID:           uuid.New(),
		Login:        "alice",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	store.usersByID[user.ID] = user
	store.usersByLogin[user.Login] = user

	svc, cleanup := newTestAuthService(t, store)
	defer cleanup()

	result, accessToken, refreshToken, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.ID != user.ID {
		t.Fatalf("expected user id %s, got %s", user.ID, result.ID)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("expected tokens to be returned")
	}

	claims, err := security.ValidateJWT(accessToken, "test-secret")
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected access token user id %s, got %s", user.ID, claims.UserID)
	}
}

func TestAuthServiceRefreshAndLogout(t *testing.T) {
	store := newTestStore()
	user := testUserRow{
		ID:           uuid.New(),
		Login:        "alice",
		PasswordHash: "hashed",
		CreatedAt:    time.Now().UTC(),
	}
	store.usersByID[user.ID] = user
	store.usersByLogin[user.Login] = user

	rawRefreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	refreshRow := testRefreshRow{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: security.HashRefreshToken(rawRefreshToken),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
	store.refreshByID[refreshRow.ID] = refreshRow
	store.refreshByHash[refreshRow.TokenHash] = refreshRow

	svc, cleanup := newTestAuthService(t, store)
	defer cleanup()

	result, accessToken, nextRefreshToken, err := svc.Refresh(context.Background(), rawRefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if result.ID != user.ID {
		t.Fatalf("expected user id %s, got %s", user.ID, result.ID)
	}
	if accessToken == "" || nextRefreshToken == "" {
		t.Fatal("expected new tokens to be returned")
	}
	if accessToken == "" {
		t.Fatal("expected access token")
	}
	claims, err := security.ValidateJWT(accessToken, "test-secret")
	if err != nil {
		t.Fatalf("validate refreshed access token: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("expected access token user id %s, got %s", user.ID, claims.UserID)
	}

	store.mu.Lock()
	oldRow := store.refreshByID[refreshRow.ID]
	newHash := security.HashRefreshToken(nextRefreshToken)
	newRow, ok := store.refreshByHash[newHash]
	store.mu.Unlock()

	if !oldRow.Revoked {
		t.Fatal("expected old refresh token to be revoked")
	}
	if !ok {
		t.Fatal("expected new refresh token to be stored")
	}
	if newRow.UserID != user.ID {
		t.Fatalf("expected new refresh token user id %s, got %s", user.ID, newRow.UserID)
	}

	if err := svc.Logout(context.Background(), nextRefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.refreshByID[newRow.ID].Revoked {
		t.Fatal("expected logout to revoke refresh token")
	}
}
