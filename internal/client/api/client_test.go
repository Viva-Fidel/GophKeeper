package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
)

func TestClientAuthAndSecrets(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/user/register", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(user.AuthResponse{Token: "tok", Salt: "c2FsdA=="})
	})
	mux.HandleFunc("POST /api/user/login", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(user.AuthResponse{Token: "tok2", Salt: "c2FsdA=="})
	})
	mux.HandleFunc("POST /api/secrets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(secret.SecretDTO{ID: "1", Type: secret.TypeText, Version: 1})
	})
	mux.HandleFunc("GET /api/secrets", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]secret.SecretDTO{{ID: "1"}})
	})
	mux.HandleFunc("GET /api/secrets/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(secret.SecretDTO{ID: r.PathValue("id")})
	})
	mux.HandleFunc("DELETE /api/secrets/{id}", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(secret.SecretDTO{ID: r.PathValue("id"), Deleted: true})
	})
	mux.HandleFunc("POST /api/secrets/sync", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(secret.SyncResponse{Secrets: []secret.SecretDTO{{ID: "1"}}, ServerTime: time.Now().UTC()})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := New(srv.URL)
	salt := base64.StdEncoding.EncodeToString([]byte("salt"))
	if _, err := c.Register("u", "p", salt); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Login("u", "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Upsert(secret.UpsertRequest{Type: secret.TypeText, Ciphertext: "YQ=="}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get("1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Delete("1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Sync(secret.SyncRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestClientAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	}))
	defer srv.Close()
	c := New(srv.URL)
	if _, err := c.Login("u", "p"); err == nil {
		t.Fatal("expected error")
	}
}
