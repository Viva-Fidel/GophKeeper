// Package auth реализует JWT и хеширование паролей.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — полезные данные JWT.
type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// NewToken создаёт подписанный JWT для пользователя.
func NewToken(userID int64, secret string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken проверяет подпись и срок действия JWT.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
