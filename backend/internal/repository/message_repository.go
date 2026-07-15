package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/google/uuid"
)

type MessageRepository struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) CreateMessage(ctx context.Context, chatID uuid.UUID, senderID *uuid.UUID, body string, clientMsgID *string) (*models.Message, bool, error) {
	msg := &models.Message{
		ID:          uuid.New(),
		ChatID:      chatID,
		SenderID:    senderID,
		Body:        body,
		ClientMsgID: clientMsgID,
		CreatedAt:   time.Now(),
	}

	query := `
		INSERT INTO messages (id, chat_id, sender_id, body, client_msg_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chat_id, client_msg_id) DO NOTHING
		RETURNING seq
	`

	err := r.db.QueryRowContext(ctx, query, msg.ID, msg.ChatID, msg.SenderID, msg.Body, msg.ClientMsgID, msg.CreatedAt).Scan(&msg.Seq)
	if err == nil {
		return msg, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, fmt.Errorf("insert message: %w", err)
	}

	existing, err := r.getMessageByClientID(ctx, chatID, clientMsgID)
	if err != nil {
		return nil, false, fmt.Errorf("fetch duplicate message: %w", err)
	}

	return existing, true, nil
}

func (r *MessageRepository) getMessageByClientID(ctx context.Context, chatID uuid.UUID, clientMsgID *string) (*models.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, body, client_msg_id, seq, created_at
		FROM messages
		WHERE chat_id = $1 AND client_msg_id = $2
	`

	msg := &models.Message{}
	err := r.db.QueryRowContext(ctx, query, chatID, clientMsgID).Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderID,
		&msg.Body,
		&msg.ClientMsgID,
		&msg.Seq,
		&msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query message by client id: %w", err)
	}

	return msg, nil
}

func (r *MessageRepository) GetChatMessagesAfterSeq(ctx context.Context, chatID uuid.UUID, afterSeq int64, limit int) ([]models.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, body, client_msg_id, seq, created_at
		FROM messages
		WHERE chat_id = $1 AND seq > $2
		ORDER BY seq ASC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("query chat messages after seq: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Body,
			&msg.ClientMsgID,
			&msg.Seq,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return messages, nil
}

func (r *MessageRepository) GetChatMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, body, client_msg_id, seq, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY seq DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Body,
			&msg.ClientMsgID,
			&msg.Seq,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *MessageRepository) SearchChatMessages(ctx context.Context, chatID uuid.UUID, queryText string, limit, offset int) ([]models.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, body, client_msg_id, seq, created_at
		FROM messages
		WHERE chat_id = $1 AND body ILIKE '%' || $2 || '%'
		ORDER BY seq DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, chatID, queryText, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query chat messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.Body,
			&msg.ClientMsgID,
			&msg.Seq,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
