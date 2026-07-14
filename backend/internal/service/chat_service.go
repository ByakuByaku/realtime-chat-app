package service

import (
	"context"
	"fmt"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/google/uuid"
)

const defaultMemberRole = models.ChatRoleMember

type ChatService struct {
	chats *repository.ChatRepository
}

func NewChatService(chats *repository.ChatRepository) *ChatService {
	return &ChatService{chats: chats}
}

func (s *ChatService) CreateChat(ctx context.Context, chatType models.ChatType, title *string, createdBy *uuid.UUID) (*models.Chat, error) {
	if chatType != models.ChatTypeDirect && chatType != models.ChatTypeGroup {
		return nil, fmt.Errorf("unsupported chat type")
	}

	chat, err := s.chats.CreateChat(ctx, chatType, title, createdBy)
	if err != nil {
		return nil, fmt.Errorf("create chat: %w", err)
	}

	return chat, nil
}

func (s *ChatService) AddMember(ctx context.Context, chatID, userID uuid.UUID, role models.ChatRole) error {
	if role == "" {
		role = defaultMemberRole
	}

	if err := s.chats.AddMember(ctx, chatID, userID, role); err != nil {
		return fmt.Errorf("add chat member: %w", err)
	}

	return nil
}

func (s *ChatService) RemoveMember(ctx context.Context, chatID, userID uuid.UUID) error {
	if err := s.chats.RemoveMember(ctx, chatID, userID); err != nil {
		return fmt.Errorf("remove chat member: %w", err)
	}

	return nil
}

func (s *ChatService) GetChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error) {
	chats, err := s.chats.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get chats: %w", err)
	}

	return chats, nil
}
