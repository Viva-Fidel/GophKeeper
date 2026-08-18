// Package httpserver собирает HTTP-маршруты GophKeeper.
package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
	"gophkeeper/pkg/middleware"
)

// Server — HTTP API сервера GophKeeper.
type Server struct {
	users   *user.Service
	secrets *secret.Service
}

// New создаёт HTTP-сервер.
func New(users *user.Service, secrets *secret.Service) *Server {
	return &Server{users: users, secrets: secrets}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader сохраняет статус-код ответа для логирования.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// statusCode возвращает записанный HTTP-статус или 200 по умолчанию.
func (rw *responseWriter) statusCode() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Router возвращает корневой HTTP-handler.
func (s *Server) Router() http.Handler {
	router := http.NewServeMux()
	auth := middleware.Auth(s.users)

	user.RegisterHTTP(router, user.NewHandler(user.HandlerDeps{Service: s.users}))
	secret.RegisterHTTP(router, auth, secret.NewHandler(secret.HandlerDeps{Service: s.secrets}))

	return s.logging(router)
}

// logging оборачивает handler и пишет в лог метод, путь, статус и длительность.
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w}
		next.ServeHTTP(wrapped, r)
		slog.InfoContext(r.Context(), "http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", wrapped.statusCode()),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
