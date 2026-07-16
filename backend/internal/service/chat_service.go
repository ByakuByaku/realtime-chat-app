package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/google/uuid"
)

const defaultMemberRole = models.ChatRoleMember

var ErrForbidden = errors.New("forbidden")

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

	if createdBy != nil {
		if err := s.chats.AddMember(ctx, chat.ID, *createdBy, models.ChatRoleAdmin); err != nil {
			return nil, fmt.Errorf("add chat creator as admin: %w", err)
		}
	}

	return chat, nil
}

func (s *ChatService) AddMember(ctx context.Context, actorID, chatID, userID uuid.UUID, role models.ChatRole) error {
	if err := s.requireAdmin(ctx, chatID, actorID); err != nil {
		return err
	}

	if role == "" {
		role = defaultMemberRole
	}

	chat, err := s.chats.GetChatByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("get chat: %w", err)
	}
	if chat.CreatedBy != nil && *chat.CreatedBy == userID {
		role = models.ChatRoleAdmin
	}

	if err := s.chats.AddMember(ctx, chatID, userID, role); err != nil {
		return fmt.Errorf("add chat member: %w", err)
	}

	return nil
}

func (s *ChatService) RemoveMember(ctx context.Context, actorID, chatID, userID uuid.UUID) error {
	if actorID != userID {
		if err := s.requireAdmin(ctx, chatID, actorID); err != nil {
			return err
		}
	}

	if err := s.chats.RemoveMember(ctx, chatID, userID); err != nil {
		return fmt.Errorf("remove chat member: %w", err)
	}

	return nil
}

func (s *ChatService) DeleteChat(ctx context.Context, actorID, chatID uuid.UUID) error {
	chat, err := s.chats.GetChatByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("get chat: %w", err)
	}

	if chat.Type == models.ChatTypeDirect {
		member, err := s.IsMember(ctx, chatID, actorID)
		if err != nil {
			return err
		}
		if !member && (chat.CreatedBy == nil || *chat.CreatedBy != actorID) {
			return ErrForbidden
		}
	} else if err := s.requireAdmin(ctx, chatID, actorID); err != nil {
		return err
	}

	if err := s.chats.DeleteChat(ctx, chatID); err != nil {
		return fmt.Errorf("delete chat: %w", err)
	}

	return nil
}

func (s *ChatService) requireAdmin(ctx context.Context, chatID, actorID uuid.UUID) error {
	role, err := s.chats.GetMemberRole(ctx, chatID, actorID)
	if err != nil {
		if errors.Is(err, repository.ErrChatMemberNotFound) {
			chat, chatErr := s.chats.GetChatByID(ctx, chatID)
			if chatErr != nil {
				return fmt.Errorf("get chat: %w", chatErr)
			}
			if chat.CreatedBy != nil && *chat.CreatedBy == actorID {
				return nil
			}
			return ErrForbidden
		}
		return fmt.Errorf("get member role: %w", err)
	}

	if role != models.ChatRoleAdmin {
		return ErrForbidden
	}

	return nil
}

func (s *ChatService) IsMember(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	_, err := s.chats.GetMemberRole(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrChatMemberNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("check chat membership: %w", err)
	}

	return true, nil
}

func (s *ChatService) GetChats(ctx context.Context, userID uuid.UUID) ([]models.Chat, error) {
	chats, err := s.chats.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get chats: %w", err)
	}

	return chats, nil
}

func (s *ChatService) EnsureMember(ctx context.Context, actorID, chatID uuid.UUID) error {
	member, err := s.IsMember(ctx, chatID, actorID)
	if err != nil {
		return err
	}
	if member {
		return nil
	}

	chat, err := s.chats.GetChatByID(ctx, chatID)
	if err != nil {
		return fmt.Errorf("get chat: %w", err)
	}
	if chat.CreatedBy == nil || *chat.CreatedBy != actorID {
		return ErrForbidden
	}

	return nil
}

func (s *ChatService) GetMembers(ctx context.Context, actorID, chatID uuid.UUID) ([]models.ChatMemberInfo, error) {
	if err := s.EnsureMember(ctx, actorID, chatID); err != nil {
		return nil, err
	}

	members, err := s.chats.GetChatMembers(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get chat members: %w", err)
	}

	return members, nil
}
