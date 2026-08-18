// Package local хранит локальное состояние CLI-клиента.
package local

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"gophkeeper/internal/secret"
)

// Session — сохранённая сессия пользователя.
type Session struct {
	ServerURL string `json:"server_url"`
	Login     string `json:"login"`
	Token     string `json:"token"`
	SaltB64   string `json:"salt_b64"`
	LastSync  time.Time `json:"last_sync"`
}

// LocalSecret — локальная копия секрета.
type LocalSecret struct {
	ID         string      `json:"id"`
	Type       secret.Type `json:"type"`
	Title      string      `json:"title"`
	Ciphertext string      `json:"ciphertext"`
	Version    int64       `json:"version"`
	UpdatedAt  time.Time   `json:"updated_at"`
	Deleted    bool        `json:"deleted"`
	Dirty      bool        `json:"dirty"`
}

// Store — файловое хранилище клиента.
type Store struct {
	Dir string
}

// DefaultDir возвращает каталог данных клиента (~/.gophkeeper).
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gophkeeper"), nil
}

// NewStore создаёт локальное хранилище в dir.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Store{Dir: dir}, nil
}

// sessionPath возвращает путь к файлу сессии.
func (s *Store) sessionPath() string {
	return filepath.Join(s.Dir, "session.json")
}

// vaultPath возвращает путь к локальному vault.
func (s *Store) vaultPath() string {
	return filepath.Join(s.Dir, "vault.json")
}

// SaveSession сохраняет сессию на диск.
func (s *Store) SaveSession(sess Session) error {
	return writeJSON(s.sessionPath(), sess)
}

// LoadSession загружает сессию с диска.
func (s *Store) LoadSession() (*Session, error) {
	var sess Session
	if err := readJSON(s.sessionPath(), &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// ClearSession удаляет файл сессии.
func (s *Store) ClearSession() error {
	err := os.Remove(s.sessionPath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// LoadVault загружает локальные секреты.
func (s *Store) LoadVault() ([]LocalSecret, error) {
	var items []LocalSecret
	err := readJSON(s.vaultPath(), &items)
	if errors.Is(err, os.ErrNotExist) {
		return []LocalSecret{}, nil
	}
	if err != nil {
		return nil, err
	}
	return items, nil
}

// SaveVault сохраняет локальные секреты.
func (s *Store) SaveVault(items []LocalSecret) error {
	return writeJSON(s.vaultPath(), items)
}

// writeJSON атомарно сохраняет значение в JSON-файл с правами 0600.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// readJSON читает JSON-файл в переданную структуру.
func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
