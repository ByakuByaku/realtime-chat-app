package httptransport

import (
	"net/http"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
	wstransport "github.com/ByakuByaku/realtime-chat-app/backend/internal/transport/websocket"
)

const apiPrefix = "/api/v1"

type Server struct {
	public    *http.ServeMux
	protected *http.ServeMux
	auth      *service.AuthService
	chats     *service.ChatService
	messages  *service.MessageService
	hub       *wstransport.Hub
}

func NewServer(auth *service.AuthService, chats *service.ChatService, messages *service.MessageService, hub *wstransport.Hub) *Server {
	s := &Server{
		public:    http.NewServeMux(),
		protected: http.NewServeMux(),
		auth:      auth,
		chats:     chats,
		messages:  messages,
		hub:       hub,
	}
	s.registerPublicRoutes()
	s.registerProtectedRoutes()
	return s
}

func (s *Server) PublicHandler() http.Handler {
	return s.public
}

func (s *Server) ProtectedHandler() http.Handler {
	return s.protected
}

func (s *Server) registerPublicRoutes() {
	s.public.HandleFunc("POST "+apiPrefix+"/auth/register", s.handleRegister)
	s.public.HandleFunc("POST "+apiPrefix+"/auth/login", s.handleLogin)
	s.public.HandleFunc("POST "+apiPrefix+"/auth/refresh", s.handleRefresh)
	s.public.HandleFunc("POST "+apiPrefix+"/auth/logout", s.handleLogout)
}

func (s *Server) registerProtectedRoutes() {
	s.protected.HandleFunc("GET "+apiPrefix+"/chats", s.handleGetChats)
	s.protected.HandleFunc("POST "+apiPrefix+"/chats", s.handleCreateChat)
	s.protected.HandleFunc("DELETE "+apiPrefix+"/chats/{chat_id}", s.handleDeleteChat)
	s.protected.HandleFunc("GET "+apiPrefix+"/chats/{chat_id}/members", s.handleGetMembers)
	s.protected.HandleFunc("POST "+apiPrefix+"/chats/{chat_id}/members", s.handleAddMember)
	s.protected.HandleFunc("DELETE "+apiPrefix+"/chats/{chat_id}/members/{user_id}", s.handleRemoveMember)
	s.protected.HandleFunc("GET "+apiPrefix+"/chats/{chat_id}/messages", s.handleGetMessages)
	s.protected.HandleFunc("POST "+apiPrefix+"/chats/{chat_id}/messages", s.handleSendMessage)
	s.protected.HandleFunc("GET "+apiPrefix+"/chats/{chat_id}/search", s.handleSearchMessages)
}

func (s *Server) ensureMethod(w http.ResponseWriter, r *http.Request, expected string) bool {
	if r.Method == expected {
		return true
	}
	writeMethodNotAllowed(w, expected)
	return false
}
