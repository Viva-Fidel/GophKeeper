// Package middleware содержит HTTP-middleware авторизации.
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

type ctxUserIDKey struct{}

var userIDKey = ctxUserIDKey{}

// AuthCookieName — имя cookie с JWT.
const AuthCookieName = "token"

// TokenParser проверяет сырой токен и возвращает id пользователя.
type TokenParser interface {
	ParseToken(token string) (int64, error)
}

// UserIDFromContext возвращает id пользователя, установленный Auth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	uid, ok := ctx.Value(userIDKey).(int64)
	return uid, ok
}

// WithUserID задаёт userID в контексте так же, как middleware Auth.
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// tokenFromRequest извлекает JWT из Authorization Bearer или cookie.
func tokenFromRequest(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	c, err := r.Cookie(AuthCookieName)
	if err == nil && c.Value != "" {
		return c.Value
	}
	return ""
}

// Auth парсит JWT и кладёт userID в контекст запроса.
func Auth(p TokenParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := tokenFromRequest(r)
			if raw == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			uid, err := p.ParseToken(raw)
			if err != nil {
				slog.WarnContext(r.Context(), "auth parse token", slog.Any("error", err))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
