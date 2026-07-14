package httptransport

import (
	"net/http"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/middleware"
)

func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodGet) {
		return
	}
	if s.messages == nil {
		writeNotImplemented(w, "message service is not configured")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	limit, offset, err := pagingFromQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	items, err := s.messages.GetHistory(r.Context(), chatID, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := MessageListResponse{Items: make([]MessageResponse, 0, len(items)), Limit: limit, Offset: offset}
	for i := range items {
		response.Items = append(response.Items, messageResponse(&items[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.messages == nil {
		writeNotImplemented(w, "message service is not configured")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	var req SendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	message, err := s.messages.SendMessage(r.Context(), chatID, &userID, req.Body, req.ClientMsgID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, messageResponse(message))
}

func (s *Server) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodGet) {
		return
	}
	if s.messages == nil {
		writeNotImplemented(w, "message service is not configured")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeBadRequest(w, "query parameter q is required")
		return
	}

	limit, offset, err := pagingFromQuery(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	items, err := s.messages.Search(r.Context(), chatID, query, limit, offset)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := MessageListResponse{Items: make([]MessageResponse, 0, len(items)), Limit: limit, Offset: offset}
	for i := range items {
		response.Items = append(response.Items, messageResponse(&items[i]))
	}

	writeJSON(w, http.StatusOK, response)
}