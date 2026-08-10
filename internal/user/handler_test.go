package user

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerRegisterLogin(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	h := NewHandler(HandlerDeps{Service: svc})
	mux := http.NewServeMux()
	RegisterHTTP(mux, h)

	salt := base64.StdEncoding.EncodeToString([]byte("saltsaltsaltsalt"))
	body, _ := json.Marshal(RegisterRequest{Login: "bob", Password: "pass", Salt: salt})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code, rr.Body.String())
	}
	var auth AuthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &auth); err != nil || auth.Token == "" {
		t.Fatal(err, auth)
	}

	body, _ = json.Marshal(LoginRequest{Login: "bob", Password: "pass"})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}
}

func TestHandlerBadRequests(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	h := NewHandler(HandlerDeps{Service: svc})
	mux := http.NewServeMux()
	RegisterHTTP(mux, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader([]byte(`{}`))))
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader([]byte(`{}`))))
	if rr.Code != http.StatusBadRequest {
		t.Fatal(rr.Code)
	}
}

func TestHandlerConflictAndUnauthorized(t *testing.T) {
	svc := New(newMockRepo(), "secret", time.Hour)
	h := NewHandler(HandlerDeps{Service: svc})
	mux := http.NewServeMux()
	RegisterHTTP(mux, h)
	salt := base64.StdEncoding.EncodeToString([]byte("saltsaltsaltsalt"))
	body, _ := json.Marshal(RegisterRequest{Login: "bob", Password: "pass", Salt: salt})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body)))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body)))
	if rr.Code != http.StatusConflict {
		t.Fatal(rr.Code)
	}
	body, _ = json.Marshal(LoginRequest{Login: "bob", Password: "wrong"})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatal(rr.Code)
	}
}
