package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
)

type uRepo struct {
	users map[string]*user.User
	next  int64
}

func (m *uRepo) CreateUser(_ context.Context, login, passwordHash string, salt []byte) (int64, error) {
	if _, ok := m.users[login]; ok {
		return 0, errors.New("user exists")
	}
	id := m.next
	m.next++
	m.users[login] = &user.User{ID: id, Login: login, PasswordHash: passwordHash, EncryptionSalt: salt}
	return id, nil
}

func (m *uRepo) GetUserByLogin(_ context.Context, login string) (*user.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

type sRepo struct {
	items map[string]secret.Secret
}

func (m *sRepo) Upsert(_ context.Context, s secret.Secret) (*secret.Secret, error) {
	s.UpdatedAt = time.Now().UTC()
	m.items[s.ID] = s
	out := s
	return &out, nil
}
func (m *sRepo) GetByID(_ context.Context, userID int64, id string) (*secret.Secret, error) {
	s, ok := m.items[id]
	if !ok || s.UserID != userID {
		return nil, secret.ErrNotFound
	}
	out := s
	return &out, nil
}
func (m *sRepo) ListByUser(_ context.Context, userID int64) ([]secret.Secret, error) {
	var out []secret.Secret
	for _, s := range m.items {
		if s.UserID == userID && !s.Deleted {
			out = append(out, s)
		}
	}
	return out, nil
}
func (m *sRepo) ListUpdatedSince(_ context.Context, userID int64, _ time.Time) ([]secret.Secret, error) {
	return m.ListByUser(context.Background(), userID)
}
func (m *sRepo) SoftDelete(ctx context.Context, userID int64, id string, version int64) (*secret.Secret, error) {
	s, err := m.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	s.Deleted = true
	s.Version = version
	m.items[id] = *s
	return s, nil
}

func TestRouterRegister(t *testing.T) {
	us := user.New(&uRepo{users: map[string]*user.User{}, next: 1}, "secret", time.Hour)
	ss := secret.New(&sRepo{items: map[string]secret.Secret{}})
	h := New(us, ss).Router()

	body, _ := json.Marshal(user.RegisterRequest{
		Login: "a", Password: "b",
		Salt: base64.StdEncoding.EncodeToString([]byte("saltsaltsaltsalt")),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code, rr.Body.String())
	}
}
