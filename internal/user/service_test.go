package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type mockRepo struct {
	users map[string]*User
	next  int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: map[string]*User{}, next: 1}
}

func (m *mockRepo) CreateUser(_ context.Context, login, passwordHash string, salt []byte) (int64, error) {
	if _, ok := m.users[login]; ok {
		return 0, errors.New("user exists")
	}
	id := m.next
	m.next++
	m.users[login] = &User{ID: id, Login: login, PasswordHash: passwordHash, EncryptionSalt: salt}
	return id, nil
}

func (m *mockRepo) GetUserByLogin(_ context.Context, login string) (*User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

func TestServiceRegisterLogin(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	res, err := svc.Register(context.Background(), "alice", "pass", []byte("saltsaltsaltsalt"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Token == "" || len(res.Salt) == 0 {
		t.Fatal(res)
	}
	uid, err := svc.ParseToken(res.Token)
	if err != nil || uid != 1 {
		t.Fatal(uid, err)
	}
	loginRes, err := svc.Login(context.Background(), "alice", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if loginRes.Token == "" {
		t.Fatal("empty token")
	}
}

func TestServiceRegisterExists(t *testing.T) {
	svc := New(newMockRepo(), "secret", 0)
	salt := []byte("saltsaltsaltsalt")
	if _, err := svc.Register(context.Background(), "alice", "pass", salt); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(context.Background(), "alice", "pass", salt); !errors.Is(err, ErrUserExists) {
		t.Fatal(err)
	}
}

func TestServiceRegisterNeedsSalt(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	if _, err := svc.Register(context.Background(), "a", "b", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceLoginInvalid(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	if _, err := svc.Login(context.Background(), "no", "pass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal(err)
	}
	salt := []byte("saltsaltsaltsalt")
	if _, err := svc.Register(context.Background(), "alice", "pass", salt); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "alice", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal(err)
	}
}

func TestTokenTTL(t *testing.T) {
	svc := New(newMockRepo(), "secret", 2*time.Hour)
	if svc.TokenTTL() != 2*time.Hour {
		t.Fatal(svc.TokenTTL())
	}
}
