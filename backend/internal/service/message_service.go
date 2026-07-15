package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/google/uuid"
)

const (
	defaultMessageLimit = 50
	maxMessageLimit     = 100
)

type MessageService struct {
	messages *repository.MessageRepository
}

func NewMessageService(messages *repository.MessageRepository) *MessageService {
	return &MessageService{messages: messages}
}

func (s *MessageService) SendMessage(ctx context.Context, chatID uuid.UUID, senderID *uuid.UUID, body string, clientMsgID *string) (*models.Message, bool, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, false, fmt.Errorf("message body is required")
	}
	if clientMsgID == nil || strings.TrimSpace(*clientMsgID) == "" {
		return nil, false, fmt.Errorf("client message id is required")
	}

	message, duplicate, err := s.messages.CreateMessage(ctx, chatID, senderID, body, clientMsgID)
	if err != nil {
		return nil, false, fmt.Errorf("send message: %w", err)
	}

	return message, duplicate, nil
}

func (s *MessageService) GetHistoryAfter(ctx context.Context, chatID uuid.UUID, afterSeq int64, limit int) ([]models.Message, error) {
	limit, _ = normalizePaging(limit, 0)

	messages, err := s.messages.GetChatMessagesAfterSeq(ctx, chatID, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("get history after seq: %w", err)
	}

	return messages, nil
}

func (s *MessageService) GetHistory(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]models.Message, error) {
	limit, offset = normalizePaging(limit, offset)

	messages, err := s.messages.GetChatMessages(ctx, chatID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	return messages, nil
}

func (s *MessageService) Search(ctx context.Context, chatID uuid.UUID, query string, limit, offset int) ([]models.Message, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	limit, offset = normalizePaging(limit, offset)

	messages, err := s.messages.SearchChatMessages(ctx, chatID, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}

	return messages, nil
}

func normalizePaging(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultMessageLimit
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
