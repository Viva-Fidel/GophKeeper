package user

import (
	"context"
	"errors"
	"time"

	"gophkeeper/internal/auth"
)

// Repository — интерфейс хранилища пользователей.
type Repository interface {
	CreateUser(ctx context.Context, login, passwordHash string, salt []byte) (int64, error)
	GetUserByLogin(ctx context.Context, login string) (*User, error)
}

// AuthResult — результат успешной регистрации или входа.
type AuthResult struct {
	Token string
	Salt  []byte
}

// Service — бизнес-логика регистрации и входа.
type Service struct {
	repo        Repository
	jwtSecret   string
	jwtTokenTTL time.Duration
}

// New создаёт сервис пользователей.
func New(repo Repository, secret string, tokenTTL time.Duration) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &Service{repo: repo, jwtSecret: secret, jwtTokenTTL: tokenTTL}
}

// TokenTTL возвращает срок жизни JWT.
func (s *Service) TokenTTL() time.Duration {
	return s.jwtTokenTTL
}

// Register регистрирует пользователя и возвращает JWT с солью шифрования.
func (s *Service) Register(ctx context.Context, login, password string, salt []byte) (*AuthResult, error) {
	if len(salt) == 0 {
		return nil, errors.New("encryption salt is required")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	id, err := s.repo.CreateUser(ctx, login, hash, salt)
	if err != nil {
		return nil, err
	}
	token, err := auth.NewToken(id, s.jwtSecret, s.jwtTokenTTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, Salt: salt}, nil
}

// Login аутентифицирует пользователя и возвращает JWT с солью шифрования.
func (s *Service) Login(ctx context.Context, login, password string) (*AuthResult, error) {
	u, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !auth.VerifyPassword(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	token, err := auth.NewToken(u.ID, s.jwtSecret, s.jwtTokenTTL)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Token: token, Salt: u.EncryptionSalt}, nil
}

// ParseToken извлекает userID из JWT.
func (s *Service) ParseToken(token string) (int64, error) {
	claims, err := auth.ParseToken(token, s.jwtSecret)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
