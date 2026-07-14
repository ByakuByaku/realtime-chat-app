package httptransport

import (
	"net/http"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
)

const apiPrefix = "/api/v1"

type Server struct {
	mux      *http.ServeMux
	auth     *service.AuthService
	chats    *service.ChatService
	messages *service.MessageService
}

func NewServer(auth *service.AuthService, chats *service.ChatService, messages *service.MessageService) *Server {
	s := &Server{
		mux:      http.NewServeMux(),
		auth:     auth,
		chats:    chats,
		messages: messages,
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST "+apiPrefix+"/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST "+apiPrefix+"/auth/login", s.handleLogin)

	s.mux.HandleFunc("GET "+apiPrefix+"/chats", s.handleGetChats)
	s.mux.HandleFunc("POST "+apiPrefix+"/chats", s.handleCreateChat)
	s.mux.HandleFunc("POST "+apiPrefix+"/chats/{chat_id}/members", s.handleAddMember)
	s.mux.HandleFunc("DELETE "+apiPrefix+"/chats/{chat_id}/members/{user_id}", s.handleRemoveMember)
	s.mux.HandleFunc("GET "+apiPrefix+"/chats/{chat_id}/messages", s.handleGetMessages)
	s.mux.HandleFunc("POST "+apiPrefix+"/chats/{chat_id}/messages", s.handleSendMessage)
	s.mux.HandleFunc("GET "+apiPrefix+"/chats/{chat_id}/search", s.handleSearchMessages)
}

func (s *Server) ensureMethod(w http.ResponseWriter, r *http.Request, expected string) bool {
	if r.Method == expected {
		return true
	}
	writeMethodNotAllowed(w, expected)
	return false
}