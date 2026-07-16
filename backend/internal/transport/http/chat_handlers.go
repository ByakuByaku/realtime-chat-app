package httptransport

import (
	"net/http"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/middleware"
)

func (s *Server) handleGetChats(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodGet) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	items, err := s.chats.GetChats(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := ChatListResponse{Items: make([]ChatResponse, 0, len(items))}
	for i := range items {
		response.Items = append(response.Items, chatResponse(&items[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateChat(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	var req CreateChatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	createdBy := userID
	chat, err := s.chats.CreateChat(r.Context(), req.Type, req.Title, &createdBy)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, chatResponse(chat))
}

func (s *Server) handleDeleteChat(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodDelete) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.chats.DeleteChat(r.Context(), actorID, chatID); err != nil {
		handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodGet) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	items, err := s.chats.GetMembers(r.Context(), actorID, chatID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response := ChatMemberListResponse{Items: make([]ChatMemberResponse, 0, len(items))}
	for i := range items {
		response.Items = append(response.Items, chatMemberResponse(&items[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodPost) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	var req AddMemberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.chats.AddMember(r.Context(), actorID, chatID, req.UserID, req.Role); err != nil {
		handleServiceError(w, err)
		return
	}

	members, err := s.chats.GetMembers(r.Context(), actorID, chatID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	for i := range members {
		if members[i].UserID == req.UserID {
			writeJSON(w, http.StatusCreated, chatMemberResponse(&members[i]))
			return
		}
	}

	writeJSON(w, http.StatusCreated, ChatMemberResponse{ChatID: chatID, UserID: req.UserID, Role: req.Role})
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	if !s.ensureMethod(w, r, http.MethodDelete) {
		return
	}
	if s.chats == nil {
		writeNotImplemented(w, "chat service is not configured")
		return
	}

	actorID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "missing user context")
		return
	}

	chatID, err := pathUUID(r, "chat_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	userID, err := pathUUID(r, "user_id")
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	if err := s.chats.RemoveMember(r.Context(), actorID, chatID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	if s.hub != nil {
		s.hub.DisconnectUser(chatID, userID)
	}

	w.WriteHeader(http.StatusNoContent)
}
