package secret

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"gophkeeper/pkg/middleware"
)

// Handler обрабатывает HTTP-запросы секретов.
type Handler struct {
	svc *Service
}

// HandlerDeps — зависимости HTTP-обработчика секретов.
type HandlerDeps struct {
	Service *Service
}

// NewHandler создаёт HTTP-обработчик секретов.
func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{svc: deps.Service}
}

// RegisterHTTP регистрирует защищённые маршруты секретов.
func RegisterHTTP(router *http.ServeMux, auth func(http.Handler) http.Handler, h *Handler) {
	router.Handle("POST /api/secrets", auth(http.HandlerFunc(h.Upsert)))
	router.Handle("GET /api/secrets", auth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/secrets/{id}", auth(http.HandlerFunc(h.Get)))
	router.Handle("DELETE /api/secrets/{id}", auth(http.HandlerFunc(h.Delete)))
	router.Handle("POST /api/secrets/sync", auth(http.HandlerFunc(h.Sync)))
}

// Upsert создаёт или обновляет секрет.
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req UpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sec, err := h.svc.Upsert(r.Context(), userID, req)
	if err != nil {
		writeSecretError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(*sec))
}

// List возвращает список секретов пользователя.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	secrets, err := h.svc.List(r.Context(), userID)
	if err != nil {
		slog.ErrorContext(r.Context(), "list secrets", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	dtos := make([]SecretDTO, 0, len(secrets))
	for _, s := range secrets {
		dtos = append(dtos, ToDTO(s))
	}
	writeJSON(w, http.StatusOK, dtos)
}

// Get возвращает один секрет.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	sec, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		writeSecretError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(*sec))
}

// Delete мягко удаляет секрет.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := r.PathValue("id")
	sec, err := h.svc.Delete(r.Context(), userID, id, 0)
	if err != nil {
		writeSecretError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ToDTO(*sec))
}

// Sync синхронизирует изменения клиента и сервера.
func (h *Handler) Sync(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := h.svc.Sync(r.Context(), userID, req)
	if err != nil {
		writeSecretError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// writeSecretError пишет HTTP-статус по типу доменной ошибки.
func writeSecretError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Is(err, ErrInvalidType):
		w.WriteHeader(http.StatusBadRequest)
	default:
		slog.ErrorContext(r.Context(), "secret handler", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// writeJSON отдаёт JSON-ответ с указанным статусом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
