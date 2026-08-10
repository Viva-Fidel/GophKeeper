package secret

import (
	"context"
	"database/sql"
	"time"
)

// SecretRepository хранит секреты в PostgreSQL.
type SecretRepository struct {
	db *sql.DB
}

// NewSecretRepository создаёт репозиторий секретов.
func NewSecretRepository(database *sql.DB) *SecretRepository {
	return &SecretRepository{db: database}
}

// Upsert создаёт или обновляет секрет (last-write-wins по version).
func (repo *SecretRepository) Upsert(ctx context.Context, s Secret) (*Secret, error) {
	row := repo.db.QueryRowContext(ctx, `
		INSERT INTO secrets (id, user_id, type, title, ciphertext, version, updated_at, deleted)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), $7)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			ciphertext = EXCLUDED.ciphertext,
			version = EXCLUDED.version,
			updated_at = NOW(),
			deleted = EXCLUDED.deleted
		WHERE secrets.user_id = EXCLUDED.user_id
		  AND EXCLUDED.version >= secrets.version
		RETURNING id, user_id, type, title, ciphertext, version, updated_at, deleted
	`, s.ID, s.UserID, s.Type, s.Title, s.Ciphertext, s.Version, s.Deleted)

	out := &Secret{}
	err := row.Scan(&out.ID, &out.UserID, &out.Type, &out.Title, &out.Ciphertext, &out.Version, &out.UpdatedAt, &out.Deleted)
	if err == sql.ErrNoRows {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetByID возвращает секрет владельца по id.
func (repo *SecretRepository) GetByID(ctx context.Context, userID int64, id string) (*Secret, error) {
	out := &Secret{}
	err := repo.db.QueryRowContext(ctx, `
		SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted
		FROM secrets WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&out.ID, &out.UserID, &out.Type, &out.Title, &out.Ciphertext, &out.Version, &out.UpdatedAt, &out.Deleted)
	if errorsIsNoRows(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListByUser возвращает все незакрытые секреты пользователя.
func (repo *SecretRepository) ListByUser(ctx context.Context, userID int64) ([]Secret, error) {
	rows, err := repo.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted
		FROM secrets WHERE user_id = $1 AND deleted = FALSE
		ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSecrets(rows)
}

// ListUpdatedSince возвращает секреты, изменённые после since (включая удалённые).
func (repo *SecretRepository) ListUpdatedSince(ctx context.Context, userID int64, since time.Time) ([]Secret, error) {
	rows, err := repo.db.QueryContext(ctx, `
		SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted
		FROM secrets WHERE user_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
	`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSecrets(rows)
}

// SoftDelete помечает секрет удалённым и увеличивает version.
func (repo *SecretRepository) SoftDelete(ctx context.Context, userID int64, id string, version int64) (*Secret, error) {
	out := &Secret{}
	err := repo.db.QueryRowContext(ctx, `
		UPDATE secrets
		SET deleted = TRUE, version = $3, updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND deleted = FALSE AND version < $3
		RETURNING id, user_id, type, title, ciphertext, version, updated_at, deleted
	`, id, userID, version).Scan(&out.ID, &out.UserID, &out.Type, &out.Title, &out.Ciphertext, &out.Version, &out.UpdatedAt, &out.Deleted)
	if errorsIsNoRows(err) {
		existing, getErr := repo.GetByID(ctx, userID, id)
		if getErr != nil {
			return nil, getErr
		}
		if existing.Deleted {
			return existing, nil
		}
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// errorsIsNoRows проверяет, что ошибка — sql.ErrNoRows.
func errorsIsNoRows(err error) bool {
	return err == sql.ErrNoRows
}

// scanSecrets читает все строки выборки в слайс Secret.
func scanSecrets(rows *sql.Rows) ([]Secret, error) {
	var result []Secret
	for rows.Next() {
		var s Secret
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Title, &s.Ciphertext, &s.Version, &s.UpdatedAt, &s.Deleted); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
