package httptransport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ByakuByaku/realtime-chat-app/backend/internal/repository"
	"github.com/ByakuByaku/realtime-chat-app/backend/internal/service"
	"github.com/google/uuid"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeBadRequest(w http.ResponseWriter, details string) {
	writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "bad_request", Details: details})
}

func writeUnauthorized(w http.ResponseWriter, details string) {
	writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized", Details: details})
}

func writeNotImplemented(w http.ResponseWriter, details string) {
	writeJSON(w, http.StatusNotImplemented, ErrorResponse{Error: "not_implemented", Details: details})
}

func writeMethodNotAllowed(w http.ResponseWriter, expected string) {
	w.Header().Set("Allow", expected)
	writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method_not_allowed", Details: fmt.Sprintf("expected %s", expected)})
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	if decoder.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func pathUUID(r *http.Request, key string) (uuid.UUID, error) {
	value := r.PathValue(key)
	if value == "" {
		return uuid.Nil, fmt.Errorf("path parameter %s is required", key)
	}

	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("path parameter %s must be a uuid", key)
	}

	return parsed, nil
}

func pagingFromQuery(r *http.Request) (int, int, error) {
	limit := 50
	offset := 0

	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return 0, 0, fmt.Errorf("limit must be an integer")
		}
		limit = parsed
	}

	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil {
			return 0, 0, fmt.Errorf("offset must be an integer")
		}
		offset = parsed
	}

	return limit, offset, nil
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case errors.Is(err, service.ErrUserAlreadyExists):
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: "user_already_exists", Details: err.Error()})
	case errors.Is(err, service.ErrInvalidCredentials):
		writeUnauthorized(w, err.Error())
	case errors.Is(err, service.ErrInvalidAuthInput):
		writeBadRequest(w, err.Error())
	case errors.Is(err, service.ErrInvalidRefreshToken):
		writeUnauthorized(w, err.Error())
	case errors.Is(err, repository.ErrUserNotFound):
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Details: err.Error()})
	case strings.Contains(err.Error(), "not found"):
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "not_found", Details: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal_error", Details: err.Error()})
	}
}
