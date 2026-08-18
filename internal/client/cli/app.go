// Package cli реализует команды CLI-клиента GophKeeper.
package cli

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"gophkeeper/internal/client/api"
	"gophkeeper/internal/client/local"
	"gophkeeper/internal/crypto"
	"gophkeeper/internal/secret"
)

// App — CLI-приложение клиента.
type App struct {
	Store    *local.Store
	Password string
}

// Payload — расшифрованное содержимое секрета.
type Payload struct {
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	Text     string `json:"text,omitempty"`
	Binary   string `json:"binary,omitempty"` // base64
	CardNumber string `json:"card_number,omitempty"`
	CardHolder string `json:"card_holder,omitempty"`
	CardExpiry string `json:"card_expiry,omitempty"`
	CardCVV    string `json:"card_cvv,omitempty"`
	Meta       string `json:"meta,omitempty"`
}

// Register регистрирует пользователя и сохраняет локальную сессию.
func (a *App) Register(serverURL, login, password string) error {
	client := api.New(strings.TrimRight(serverURL, "/"))
	salt, err := crypto.NewSalt()
	if err != nil {
		return err
	}
	resp, err := client.Register(login, password, crypto.EncodeBase64(salt))
	if err != nil {
		return err
	}
	return a.Store.SaveSession(local.Session{
		ServerURL: client.BaseURL,
		Login:     login,
		Token:     resp.Token,
		SaltB64:   resp.Salt,
		LastSync:  time.Time{},
	})
}

// Login аутентифицирует пользователя и сохраняет сессию.
func (a *App) Login(serverURL, login, password string) error {
	client := api.New(strings.TrimRight(serverURL, "/"))
	resp, err := client.Login(login, password)
	if err != nil {
		return err
	}
	lastSync := time.Time{}
	if sess, err := a.Store.LoadSession(); err == nil && sess != nil && sess.Login == login {
		lastSync = sess.LastSync
	}
	return a.Store.SaveSession(local.Session{
		ServerURL: client.BaseURL,
		Login:     login,
		Token:     resp.Token,
		SaltB64:   resp.Salt,
		LastSync:  lastSync,
	})
}

// Add добавляет секрет локально (и помечает для sync).
func (a *App) Add(typ secret.Type, title string, payload Payload) (string, error) {
	if !secret.ValidType(typ) {
		return "", secret.ErrInvalidType
	}
	key, err := a.deriveKey()
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	ct, err := crypto.Encrypt(key, raw)
	if err != nil {
		return "", err
	}
	items, err := a.Store.LoadVault()
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	items = append(items, local.LocalSecret{
		ID:         id,
		Type:       typ,
		Title:      title,
		Ciphertext: crypto.EncodeBase64(ct),
		Version:    1,
		UpdatedAt:  time.Now().UTC(),
		Deleted:    false,
		Dirty:      true,
	})
	if err := a.Store.SaveVault(items); err != nil {
		return "", err
	}
	return id, nil
}

// List возвращает активные локальные секреты.
func (a *App) List() ([]local.LocalSecret, error) {
	items, err := a.Store.LoadVault()
	if err != nil {
		return nil, err
	}
	out := make([]local.LocalSecret, 0, len(items))
	for _, it := range items {
		if !it.Deleted {
			out = append(out, it)
		}
	}
	return out, nil
}

// Get расшифровывает секрет по id.
func (a *App) Get(id string) (*Payload, *local.LocalSecret, error) {
	items, err := a.Store.LoadVault()
	if err != nil {
		return nil, nil, err
	}
	for i := range items {
		if items[i].ID == id && !items[i].Deleted {
			key, err := a.deriveKey()
			if err != nil {
				return nil, nil, err
			}
			raw, err := crypto.DecodeBase64(items[i].Ciphertext)
			if err != nil {
				return nil, nil, err
			}
			plain, err := crypto.Decrypt(key, raw)
			if err != nil {
				return nil, nil, err
			}
			var p Payload
			if err := json.Unmarshal(plain, &p); err != nil {
				return nil, nil, err
			}
			return &p, &items[i], nil
		}
	}
	return nil, nil, secret.ErrNotFound
}

// Update обновляет секрет локально.
func (a *App) Update(id string, title string, payload Payload) error {
	items, err := a.Store.LoadVault()
	if err != nil {
		return err
	}
	key, err := a.deriveKey()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ct, err := crypto.Encrypt(key, raw)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id && !items[i].Deleted {
			if title != "" {
				items[i].Title = title
			}
			items[i].Ciphertext = crypto.EncodeBase64(ct)
			items[i].Version++
			items[i].UpdatedAt = time.Now().UTC()
			items[i].Dirty = true
			return a.Store.SaveVault(items)
		}
	}
	return secret.ErrNotFound
}

// Delete мягко удаляет секрет локально.
func (a *App) Delete(id string) error {
	items, err := a.Store.LoadVault()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id && !items[i].Deleted {
			items[i].Deleted = true
			items[i].Version++
			items[i].UpdatedAt = time.Now().UTC()
			items[i].Dirty = true
			return a.Store.SaveVault(items)
		}
	}
	return secret.ErrNotFound
}

// Sync синхронизирует локальное хранилище с сервером.
func (a *App) Sync() error {
	sess, err := a.Store.LoadSession()
	if err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}
	client := api.New(sess.ServerURL)
	client.Token = sess.Token

	items, err := a.Store.LoadVault()
	if err != nil {
		return err
	}
	changes := make([]secret.UpsertRequest, 0)
	for _, it := range items {
		if !it.Dirty {
			continue
		}
		changes = append(changes, secret.UpsertRequest{
			ID:         it.ID,
			Type:       it.Type,
			Title:      it.Title,
			Ciphertext: it.Ciphertext,
			Version:    it.Version,
			Deleted:    it.Deleted,
		})
	}

	resp, err := client.Sync(secret.SyncRequest{Since: sess.LastSync, Changes: changes})
	if err != nil {
		return err
	}

	byID := map[string]int{}
	for i := range items {
		byID[items[i].ID] = i
		items[i].Dirty = false
	}
	for _, remote := range resp.Secrets {
		idx, ok := byID[remote.ID]
		if !ok {
			items = append(items, local.LocalSecret{
				ID:         remote.ID,
				Type:       remote.Type,
				Title:      remote.Title,
				Ciphertext: remote.Ciphertext,
				Version:    remote.Version,
				UpdatedAt:  remote.UpdatedAt,
				Deleted:    remote.Deleted,
				Dirty:      false,
			})
			continue
		}
		if remote.Version >= items[idx].Version {
			items[idx].Type = remote.Type
			items[idx].Title = remote.Title
			items[idx].Ciphertext = remote.Ciphertext
			items[idx].Version = remote.Version
			items[idx].UpdatedAt = remote.UpdatedAt
			items[idx].Deleted = remote.Deleted
			items[idx].Dirty = false
		}
	}
	if err := a.Store.SaveVault(items); err != nil {
		return err
	}
	sess.LastSync = resp.ServerTime
	return a.Store.SaveSession(*sess)
}

// deriveKey получает ключ AES из мастер-пароля и соли сессии.
func (a *App) deriveKey() ([]byte, error) {
	if a.Password == "" {
		return nil, errors.New("master password is required")
	}
	sess, err := a.Store.LoadSession()
	if err != nil {
		return nil, fmt.Errorf("not logged in: %w", err)
	}
	salt, err := crypto.DecodeBase64(sess.SaltB64)
	if err != nil {
		return nil, err
	}
	return crypto.DeriveKey(a.Password, salt, crypto.DefaultIterations), nil
}

// ReadBinaryFile читает файл и кодирует в base64 для payload.
func ReadBinaryFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
