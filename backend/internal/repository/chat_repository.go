package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/google/uuid"
)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) CreateChat(ctx context.Context, chatType models.ChatType, title *string, createdBy *uuid.UUID) (*models.Chat, error) {
	chat := &models.Chat{
		ID:        uuid.New(),
		Type:      chatType,
		Title:     title,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}

	query := `
		INSERT INTO chats (id, type, title, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query, chat.ID, chat.Type, chat.Title, chat.CreatedBy, chat.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert chat: %w", err)
	}

	return chat, nil
}

func (r *ChatRepository) AddMember(ctx context.Context, chatID, userID uuid.UUID, role models.ChatRole) error {
	query := `
		INSERT INTO chat_members (chat_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, user_id)
		DO UPDATE SET role = EXCLUDED.role
	`

	_, err := r.db.ExecContext(ctx, query, chatID, userID, role)
	if err != nil {
		return fmt.Errorf("insert chat member: %w", err)
	}

	return nil
}

func (r *ChatRepository) RemoveMember(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `
		DELETE FROM chat_members
		WHERE chat_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, chatID, userID)
	if err != nil {
		return fmt.Errorf("delete chat member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("chat member not found")
	}

	return nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, id uuid.UUID) (*models.Chat, error) {
	chat := &models.Chat{}

	query := `
		SELECT id, type, title, created_by, direct_key, last_message_at, created_at
		FROM chats
		WHERE id = $1
	`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chat.ID,
		&chat.Type,
		&chat.Title,
		&chat.CreatedBy,
		&chat.DirectKey,
		&chat.LastMessageAt,
		&chat.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("query chat: %w", err)
	}

	return chat, nil
}

func (r *ChatRepository) GetUserChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error) {
	query := `
		SELECT c.id, c.type, c.title, c.created_by, c.direct_key, c.last_message_at, c.created_at
		FROM chats c
		INNER JOIN chat_members cm ON c.id = cm.chat_id
		WHERE cm.user_id = $1
		ORDER BY c.last_message_at DESC NULLS LAST
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query user chats: %w", err)
	}
	defer rows.Close()

	var chats []models.Chat
	for rows.Next() {
		var chat models.Chat
		err := rows.Scan(
			&chat.ID,
			&chat.Type,
			&chat.Title,
			&chat.CreatedBy,
			&chat.DirectKey,
			&chat.LastMessageAt,
			&chat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan chat: %w", err)
		}
		chats = append(chats, chat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return chats, nil
}

func (r *ChatRepository) UpdateChatLastMessage(ctx context.Context, chatID uuid.UUID, timestamp time.Time) error {
	query := `
		UPDATE chats
		SET last_message_at = $1
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, timestamp, chatID)
	if err != nil {
		return fmt.Errorf("update chat last message: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("chat not found")
	}

	return nil
}
