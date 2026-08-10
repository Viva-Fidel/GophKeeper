package secret

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gophkeeper/internal/user"
	"gophkeeper/pkg/middleware"
)

func TestHandlerBadBodyAndNotFound(t *testing.T) {
	userSvc := user.New(newUserMemRepo(), "secret", time.Hour)
	authRes, err := userSvc.Register(context.Background(), "u2", "p", []byte("saltsaltsaltsalt"))
	if err != nil {
		t.Fatal(err)
	}
	secSvc := New(newMemRepo())
	mux := http.NewServeMux()
	RegisterHTTP(mux, middleware.Auth(userSvc), NewHandler(HandlerDeps{Service: secSvc}))

	req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader([]byte("{")))
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/secrets/missing", nil)
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatal(rr.Code)
	}

	body, _ := json.Marshal(UpsertRequest{Type: "bad", Ciphertext: base64.StdEncoding.EncodeToString([]byte("x"))})
	req = httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}

	syncBody, _ := json.Marshal(SyncRequest{Changes: []UpsertRequest{{ID: "x", Type: "bad"}}})
	req = httptest.NewRequest(http.MethodPost, "/api/secrets/sync", bytes.NewReader(syncBody))
	req.Header.Set("Authorization", "Bearer "+authRes.Token)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}
}

func TestServiceUpsertBadCiphertext(t *testing.T) {
	svc := New(newMemRepo())
	if _, err := svc.Upsert(context.Background(), 1, UpsertRequest{Type: TypeText, Ciphertext: "%%%"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceSyncSkipEmptyID(t *testing.T) {
	svc := New(newMemRepo())
	resp, err := svc.Sync(context.Background(), 1, SyncRequest{
		Changes: []UpsertRequest{{ID: "", Type: TypeText}},
	})
	if err != nil || resp == nil {
		t.Fatal(err, resp)
	}
}

func TestServiceConflictOnSync(t *testing.T) {
	repo := newMemRepo()
	svc := New(repo)
	ct := base64.StdEncoding.EncodeToString([]byte("c"))
	_, _ = repo.Upsert(context.Background(), Secret{ID: "id", UserID: 1, Type: TypeText, Version: 5, Ciphertext: []byte("c")})
	resp, err := svc.Sync(context.Background(), 1, SyncRequest{
		Changes: []UpsertRequest{{ID: "id", Type: TypeText, Ciphertext: ct, Version: 1}},
	})
	if err != nil || resp == nil {
		t.Fatal(err)
	}
}
