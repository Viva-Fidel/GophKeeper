package cli

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gophkeeper/internal/client/local"
	"gophkeeper/internal/httpserver"
	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
)

type syncUserRepo struct {
	users map[string]*user.User
	next  int64
}

func newSyncUserRepo() *syncUserRepo {
	return &syncUserRepo{users: map[string]*user.User{}, next: 1}
}

func (m *syncUserRepo) CreateUser(_ context.Context, login, passwordHash string, salt []byte) (int64, error) {
	if _, ok := m.users[login]; ok {
		return 0, errors.New("user exists")
	}
	id := m.next
	m.next++
	m.users[login] = &user.User{ID: id, Login: login, PasswordHash: passwordHash, EncryptionSalt: salt}
	return id, nil
}

func (m *syncUserRepo) GetUserByLogin(_ context.Context, login string) (*user.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

type syncSecretRepo struct {
	items map[string]secret.Secret
}

func newSyncSecretRepo() *syncSecretRepo {
	return &syncSecretRepo{items: map[string]secret.Secret{}}
}

func (m *syncSecretRepo) Upsert(_ context.Context, s secret.Secret) (*secret.Secret, error) {
	if old, ok := m.items[s.ID]; ok {
		if old.UserID != s.UserID || s.Version < old.Version {
			return nil, secret.ErrConflict
		}
	}
	s.UpdatedAt = time.Now().UTC()
	m.items[s.ID] = s
	out := s
	return &out, nil
}

func (m *syncSecretRepo) GetByID(_ context.Context, userID int64, id string) (*secret.Secret, error) {
	s, ok := m.items[id]
	if !ok || s.UserID != userID {
		return nil, secret.ErrNotFound
	}
	out := s
	return &out, nil
}

func (m *syncSecretRepo) ListByUser(_ context.Context, userID int64) ([]secret.Secret, error) {
	var out []secret.Secret
	for _, s := range m.items {
		if s.UserID == userID && !s.Deleted {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *syncSecretRepo) ListUpdatedSince(_ context.Context, userID int64, since time.Time) ([]secret.Secret, error) {
	var out []secret.Secret
	for _, s := range m.items {
		if s.UserID == userID && s.UpdatedAt.After(since) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *syncSecretRepo) SoftDelete(ctx context.Context, userID int64, id string, version int64) (*secret.Secret, error) {
	s, err := m.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if s.Deleted {
		return s, nil
	}
	if version <= s.Version {
		return nil, secret.ErrConflict
	}
	s.Deleted = true
	s.Version = version
	s.UpdatedAt = time.Now().UTC()
	m.items[id] = *s
	return s, nil
}

func TestAppLocalCRUD(t *testing.T) {
	store, err := local.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveSession(local.Session{
		ServerURL: "http://localhost",
		Login:     "u",
		Token:     "t",
		SaltB64:   base64.StdEncoding.EncodeToString([]byte("saltsaltsaltsalt")),
	}); err != nil {
		t.Fatal(err)
	}
	app := &App{Store: store, Password: "master"}
	id, err := app.Add(secret.TypeLoginPassword, "site", Payload{Login: "a", Password: "b", Meta: "m"})
	if err != nil {
		t.Fatal(err)
	}
	list, err := app.List()
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}
	p, meta, err := app.Get(id)
	if err != nil || p.Login != "a" || meta.Title != "site" {
		t.Fatal(err, p, meta)
	}
	if err := app.Update(id, "site2", Payload{Login: "c", Password: "d", Meta: "m2"}); err != nil {
		t.Fatal(err)
	}
	p, _, err = app.Get(id)
	if err != nil || p.Login != "c" {
		t.Fatal(err, p)
	}
	if err := app.Delete(id); err != nil {
		t.Fatal(err)
	}
	list, err = app.List()
	if err != nil || len(list) != 0 {
		t.Fatal(err, list)
	}
}

func TestReadBinaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.bin")
	if err := os.WriteFile(path, []byte{1, 2, 3}, 0o600); err != nil {
		t.Fatal(err)
	}
	b64, err := ReadBinaryFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(raw) != 3 {
		t.Fatal(err, raw)
	}
}

func TestAppSyncWithServer(t *testing.T) {
	userSvc := user.New(newSyncUserRepo(), "secret", time.Hour)
	secSvc := secret.New(newSyncSecretRepo())
	srv := httptest.NewServer(httpserver.New(userSvc, secSvc).Router())
	defer srv.Close()

	store, err := local.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app := &App{Store: store, Password: "master"}
	if err := app.Register(srv.URL, "alice", "pass"); err != nil {
		t.Fatal(err)
	}
	id, err := app.Add(secret.TypeText, "note", Payload{Text: "hello", Meta: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Sync(); err != nil {
		t.Fatal(err)
	}

	store2, err := local.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app2 := &App{Store: store2, Password: "master"}
	if err := app2.Login(srv.URL, "alice", "pass"); err != nil {
		t.Fatal(err)
	}
	if err := app2.Sync(); err != nil {
		t.Fatal(err)
	}
	p, _, err := app2.Get(id)
	if err != nil || p.Text != "hello" {
		t.Fatal(err, p)
	}
}
