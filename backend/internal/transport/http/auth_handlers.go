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

	user, accessToken, refreshToken, err := s.auth.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, authResponse(user, accessToken, refreshToken))
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

	user, accessToken, refreshToken, err := s.auth.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse(user, accessToken, refreshToken))
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.auth == nil {
		writeNotImplemented(w, "auth service is not configured")
		return
	}

	var req TokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	user, accessToken, refreshToken, err := s.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authResponse(user, accessToken, refreshToken))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.auth == nil {
		writeNotImplemented(w, "auth service is not configured")
		return
	}

	var req TokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var _ = service.ErrInvalidAuthInput
