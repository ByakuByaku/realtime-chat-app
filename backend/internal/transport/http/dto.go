package httptransport

import (
	"time"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/google/uuid"
)

type AuthRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

type TokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateChatRequest struct {
	Type  models.ChatType `json:"type"`
	Title *string         `json:"title,omitempty"`
}

type ChatResponse struct {
	ID            uuid.UUID       `json:"id"`
	Type          models.ChatType `json:"type"`
	Title         *string         `json:"title,omitempty"`
	CreatedBy     *uuid.UUID      `json:"created_by,omitempty"`
	LastMessageAt *time.Time      `json:"last_message_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ChatListResponse struct {
	Items []ChatResponse `json:"items"`
}

type AddMemberRequest struct {
	UserID uuid.UUID       `json:"user_id"`
	Role   models.ChatRole `json:"role,omitempty"`
}

type ChatMemberResponse struct {
	ChatID uuid.UUID       `json:"chat_id"`
	UserID uuid.UUID       `json:"user_id"`
	Role   models.ChatRole `json:"role"`
}

type SendMessageRequest struct {
	Body        string  `json:"body"`
	ClientMsgID *string `json:"client_msg_id"`
}

type MessageResponse struct {
	ID          uuid.UUID  `json:"id"`
	ChatID      uuid.UUID  `json:"chat_id"`
	SenderID    *uuid.UUID `json:"sender_id,omitempty"`
	Body        string     `json:"body"`
	ClientMsgID *string    `json:"client_msg_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type MessageListResponse struct {
	Items  []MessageResponse `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func authResponse(user *models.User, accessToken, refreshToken string) AuthResponse {
	return AuthResponse{
		User: UserResponse{
			ID:        user.ID,
			Login:     user.Login,
			CreatedAt: user.CreatedAt,
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

func chatResponse(chat *models.Chat) ChatResponse {
	return ChatResponse{
		ID:            chat.ID,
		Type:          chat.Type,
		Title:         chat.Title,
		CreatedBy:     chat.CreatedBy,
		LastMessageAt: chat.LastMessageAt,
		CreatedAt:     chat.CreatedAt,
	}
}

func messageResponse(message *models.Message) MessageResponse {
	return MessageResponse{
		ID:          message.ID,
		ChatID:      message.ChatID,
		SenderID:    message.SenderID,
		Body:        message.Body,
		ClientMsgID: message.ClientMsgID,
		CreatedAt:   message.CreatedAt,
	}
}
