package websocket

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/models"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/security"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const backlogLimit = 200

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

	afterSeq, err := parseAfterSeq(r)
	if err != nil {
		http.Error(w, "after_seq must be an integer", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := NewClient(r.Context(), s.hub, conn, userID, chatID, s.handleInbound)
	s.hub.Subscribe(chatID, userID, client)

	if afterSeq > 0 {
		s.replayBacklog(r.Context(), client, chatID, afterSeq)
	}

	client.Run()
}

func (s *Server) replayBacklog(ctx context.Context, client *Client, chatID uuid.UUID, afterSeq int64) {
	backlog, err := s.messages.GetHistoryAfter(ctx, chatID, afterSeq, backlogLimit)
	if err != nil {
		client.Enqueue(OutboundMessage{Type: OutboundTypeError, Error: "failed to load missed messages"})
		return
	}

	for i := range backlog {
		client.Enqueue(OutboundMessage{Type: OutboundTypeMessage, Message: payloadFromMessage(&backlog[i])})
	}
}

func (s *Server) handleInbound(client *Client, msg InboundMessage) {
	clientMsgID := msg.ClientMsgID
	if clientMsgID == nil || strings.TrimSpace(*clientMsgID) == "" {
		generated := uuid.NewString()
		clientMsgID = &generated
	}

	senderID := client.UserID
	message, duplicate, err := s.messages.SendMessage(client.ctx, client.ChatID, &senderID, msg.Body, clientMsgID)
	if err != nil {
		client.Enqueue(OutboundMessage{Type: OutboundTypeError, Error: err.Error()})
		return
	}

	payload := payloadFromMessage(message)
	client.Enqueue(OutboundMessage{Type: OutboundTypeAck, Message: payload})

	if duplicate {
		return
	}

	s.hub.Broadcast(client.ChatID, OutboundMessage{Type: OutboundTypeMessage, Message: payload})
}

func payloadFromMessage(message *models.Message) *MessagePayload {
	return &MessagePayload{
		ID:          message.ID,
		ChatID:      message.ChatID,
		SenderID:    message.SenderID,
		Body:        message.Body,
		ClientMsgID: message.ClientMsgID,
		Seq:         message.Seq,
		CreatedAt:   message.CreatedAt,
	}
}

func parseAfterSeq(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("after_seq"))
	if raw == "" {
		return 0, nil
	}

	return strconv.ParseInt(raw, 10, 64)
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
