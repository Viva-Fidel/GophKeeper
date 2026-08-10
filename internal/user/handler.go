package user

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Handler обрабатывает HTTP-запросы пользователей.
type Handler struct {
	svc *Service
}

// HandlerDeps — зависимости HTTP-обработчика.
type HandlerDeps struct {
	Service *Service
}

// NewHandler создаёт HTTP-обработчик пользователей.
func NewHandler(deps HandlerDeps) *Handler {
	return &Handler{svc: deps.Service}
}

// RegisterHTTP регистрирует публичные маршруты пользователей.
func RegisterHTTP(router *http.ServeMux, h *Handler) {
	router.HandleFunc("POST /api/user/register", h.Register)
	router.HandleFunc("POST /api/user/login", h.Login)
}

// Register регистрирует нового пользователя.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Login == "" || req.Password == "" || req.Salt == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	salt, err := base64.StdEncoding.DecodeString(req.Salt)
	if err != nil || len(salt) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := h.svc.Register(r.Context(), req.Login, req.Password, salt)
	if err != nil {
		if errors.Is(err, ErrUserExists) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		slog.ErrorContext(r.Context(), "register", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuthResponse{
		Token: result.Token,
		Salt:  base64.StdEncoding.EncodeToString(result.Salt),
	})
}

// Login аутентифицирует пользователя.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Login == "" || req.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		slog.ErrorContext(r.Context(), "login", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(AuthResponse{
		Token: result.Token,
		Salt:  base64.StdEncoding.EncodeToString(result.Salt),
	})
}
