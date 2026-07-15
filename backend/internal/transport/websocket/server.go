package websocket

import (
	"net/http"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/security"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Server struct {
	hub       *Hub
	chats     *service.ChatService
	messages  *service.MessageService
	jwtSecret string
}

func NewServer(hub *Hub, chats *service.ChatService, messages *service.MessageService, jwtSecret string) *Server {
	return &Server{hub: hub, chats: chats, messages: messages, jwtSecret: jwtSecret}
}

func (s *Server) HandleChatSocket(w http.ResponseWriter, r *http.Request) {
	chatID, err := uuid.Parse(r.PathValue("chat_id"))
	if err != nil {
		http.Error(w, "chat_id must be a uuid", http.StatusBadRequest)
		return
	}

	userID, err := authenticate(r, s.jwtSecret)
	if err != nil {
		http.Error(w, "invalid or missing token", http.StatusUnauthorized)
		return
	}

	isMember, err := s.chats.IsMember(r.Context(), chatID, userID)
	if err != nil {
		http.Error(w, "failed to verify chat membership", http.StatusInternalServerError)
		return
	}
	if !isMember {
		http.Error(w, "not a member of this chat", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(r.Context(), s.hub, conn, userID, chatID, s.handleInbound)
	s.hub.Subscribe(chatID, client)
	client.Run()
}

func (s *Server) handleInbound(client *Client, msg InboundMessage) {
	clientMsgID := msg.ClientMsgID
	if clientMsgID == nil || strings.TrimSpace(*clientMsgID) == "" {
		generated := uuid.NewString()
		clientMsgID = &generated
	}

	senderID := client.UserID
	message, err := s.messages.SendMessage(client.ctx, client.ChatID, &senderID, msg.Body, clientMsgID)
	if err != nil {
		client.Enqueue(OutboundMessage{Type: OutboundTypeError, Error: err.Error()})
		return
	}

	s.hub.Broadcast(client.ChatID, OutboundMessage{
		Type:    OutboundTypeMessage,
		Message: payloadFromMessage(message),
	})
}

func payloadFromMessage(message *models.Message) *MessagePayload {
	return &MessagePayload{
		ID:          message.ID,
		ChatID:      message.ChatID,
		SenderID:    message.SenderID,
		Body:        message.Body,
		ClientMsgID: message.ClientMsgID,
		CreatedAt:   message.CreatedAt,
	}
}

func authenticate(r *http.Request, secret string) (uuid.UUID, error) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		return uuid.Nil, security.ErrInvalidToken
	}

	claims, err := security.ValidateJWT(token, secret)
	if err != nil {
		return uuid.Nil, err
	}

	return claims.UserID, nil
}
