package httptransport

import (
	"net/http"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.auth == nil {
		writeNotImplemented(w, "auth service is not configured")
		return
	}

	var req AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	user, token, err := s.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse(user, token))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.auth == nil {
		writeNotImplemented(w, "auth service is not configured")
		return
	}

	var req AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	user, token, err := s.auth.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse(user, token))
}

var _ = service.ErrInvalidAuthInput