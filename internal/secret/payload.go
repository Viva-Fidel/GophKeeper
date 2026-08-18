package secret

import (
	"encoding/base64"
	"time"
)

// SecretDTO — представление секрета в API.
type SecretDTO struct {
	ID         string    `json:"id"`
	Type       Type      `json:"type"`
	Title      string    `json:"title"`
	Ciphertext string    `json:"ciphertext"`
	Version    int64     `json:"version"`
	UpdatedAt  time.Time `json:"updated_at"`
	Deleted    bool      `json:"deleted"`
}

// UpsertRequest — создание или обновление секрета.
type UpsertRequest struct {
	ID         string `json:"id"`
	Type       Type   `json:"type"`
	Title      string `json:"title"`
	Ciphertext string `json:"ciphertext"`
	Version    int64  `json:"version"`
	Deleted    bool   `json:"deleted"`
}

// SyncRequest — запрос синхронизации.
type SyncRequest struct {
	Since   time.Time      `json:"since"`
	Changes []UpsertRequest `json:"changes"`
}

// SyncResponse — ответ синхронизации с изменениями сервера.
type SyncResponse struct {
	Secrets   []SecretDTO `json:"secrets"`
	ServerTime time.Time  `json:"server_time"`
}

// ToDTO преобразует Secret в DTO.
func ToDTO(s Secret) SecretDTO {
	return SecretDTO{
		ID:         s.ID,
		Type:       s.Type,
		Title:      s.Title,
		Ciphertext: base64.StdEncoding.EncodeToString(s.Ciphertext),
		Version:    s.Version,
		UpdatedAt:  s.UpdatedAt,
		Deleted:    s.Deleted,
	}
}
