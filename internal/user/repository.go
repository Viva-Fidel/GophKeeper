package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepository хранит пользователей в PostgreSQL.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository создаёт репозиторий пользователей.
func NewUserRepository(database *sql.DB) *UserRepository {
	return &UserRepository{db: database}
}

// isUniqueViolation определяет ошибку нарушения UNIQUE в PostgreSQL.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateUser создаёт пользователя и возвращает его id.
func (repo *UserRepository) CreateUser(ctx context.Context, login, passwordHash string, salt []byte) (int64, error) {
	var id int64
	err := repo.db.QueryRowContext(ctx,
		`INSERT INTO users (login, password_hash, encryption_salt) VALUES ($1, $2, $3) RETURNING id`,
		login, passwordHash, salt,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrUserExists
		}
		return 0, err
	}
	return id, nil
}

// GetUserByLogin возвращает пользователя по логину.
func (repo *UserRepository) GetUserByLogin(ctx context.Context, login string) (*User, error) {
	u := &User{}
	err := repo.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash, encryption_salt FROM users WHERE login = $1`, login,
	).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.EncryptionSalt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}
