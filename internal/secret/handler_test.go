package secret

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

	"gophkeeper/internal/user"
	"gophkeeper/pkg/middleware"
)

type userMemRepo struct {
	users map[string]*user.User
	next  int64
}

func newUserMemRepo() *userMemRepo {
	return &userMemRepo{users: map[string]*user.User{}, next: 1}
}

func (m *userMemRepo) CreateUser(_ context.Context, login, passwordHash string, salt []byte) (int64, error) {
	if _, ok := m.users[login]; ok {
		return 0, errors.New("user exists")
	}
	id := m.next
	m.next++
	m.users[login] = &user.User{ID: id, Login: login, PasswordHash: passwordHash, EncryptionSalt: salt}
	return id, nil
}

func (m *userMemRepo) GetUserByLogin(_ context.Context, login string) (*user.User, error) {
	u, ok := m.users[login]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

func TestHandlerSecretsFlow(t *testing.T) {
	userSvc := user.New(newUserMemRepo(), "secret", time.Hour)
	salt := []byte("saltsaltsaltsalt")
	authRes, err := userSvc.Register(context.Background(), "u", "p", salt)
	if err != nil {
		t.Fatal(err)
	}
	secSvc := New(newMemRepo())
	mux := http.NewServeMux()
	RegisterHTTP(mux, middleware.Auth(userSvc), NewHandler(HandlerDeps{Service: secSvc}))

	ct := base64.StdEncoding.EncodeToString([]byte("data"))
	body, _ := json.Marshal(UpsertRequest{Type: TypeText, Title: "t", Ciphertext: ct})
	req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var dto SecretDTO
	_ = json.Unmarshal(rr.Body.Bytes(), &dto)

	req = httptest.NewRequest(http.MethodGet, "/api/secrets", nil)
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/secrets/"+dto.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	syncBody, _ := json.Marshal(SyncRequest{Since: time.Time{}})
	req = httptest.NewRequest(http.MethodPost, "/api/secrets/sync", bytes.NewReader(syncBody))
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/secrets/"+dto.ID, nil)
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}
}

func TestHandlerUnauthorized(t *testing.T) {
	secSvc := New(newMemRepo())
	mux := http.NewServeMux()
	userSvc := user.New(newUserMemRepo(), "secret", time.Hour)
	RegisterHTTP(mux, middleware.Auth(userSvc), NewHandler(HandlerDeps{Service: secSvc}))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/secrets", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatal(rr.Code)
	}
}
