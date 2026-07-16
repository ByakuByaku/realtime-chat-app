package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/google/uuid"
)

var ErrChatMemberNotFound = errors.New("chat member not found")

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

func (r *ChatRepository) DeleteChat(ctx context.Context, chatID uuid.UUID) error {
	query := `
		DELETE FROM chats
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("delete chat: %w", err)
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

func (r *ChatRepository) GetMemberRole(ctx context.Context, chatID, userID uuid.UUID) (models.ChatRole, error) {
	var role models.ChatRole

	query := `
		SELECT role
		FROM chat_members
		WHERE chat_id = $1 AND user_id = $2
	`

	err := r.db.QueryRowContext(ctx, query, chatID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrChatMemberNotFound
		}
		return "", fmt.Errorf("query chat member role: %w", err)
	}

	return role, nil
}

func (r *ChatRepository) GetChatMembers(ctx context.Context, chatID uuid.UUID) ([]models.ChatMemberInfo, error) {
	query := `
		WITH chat_users AS (
			SELECT cm.chat_id, cm.user_id, cm.role, cm.joined_at
			FROM chat_members cm
			WHERE cm.chat_id = $1

			UNION

			SELECT c.id, c.created_by, 'admin', c.created_at
			FROM chats c
			WHERE c.id = $1
				AND c.created_by IS NOT NULL
				AND NOT EXISTS (
					SELECT 1
					FROM chat_members cm
					WHERE cm.chat_id = c.id AND cm.user_id = c.created_by
				)
		)
		SELECT cu.chat_id, cu.user_id, u.login, cu.role, cu.joined_at
		FROM chat_users cu
		INNER JOIN users u ON u.id = cu.user_id
		ORDER BY u.login
	`

	rows, err := r.db.QueryContext(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("query chat members: %w", err)
	}
	defer rows.Close()

	var members []models.ChatMemberInfo
	for rows.Next() {
		var member models.ChatMemberInfo
		err := rows.Scan(
			&member.ChatID,
			&member.UserID,
			&member.Login,
			&member.Role,
			&member.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan chat member: %w", err)
		}
		members = append(members, member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return members, nil
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
