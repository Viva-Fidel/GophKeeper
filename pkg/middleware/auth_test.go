package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubParser struct {
	uid int64
	err error
}

func (s stubParser) ParseToken(token string) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	if token == "ok" {
		return s.uid, nil
	}
	return 0, errors.New("bad token")
}

func TestAuthBearer(t *testing.T) {
	h := Auth(stubParser{uid: 7})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := UserIDFromContext(r.Context())
		if !ok || uid != 7 {
			t.Fatalf("uid=%v ok=%v", uid, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatal(rr.Code)
	}
}

func TestAuthCookie(t *testing.T) {
	h := Auth(stubParser{uid: 3})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: AuthCookieName, Value: "ok"})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatal(rr.Code)
	}
}

func TestAuthUnauthorized(t *testing.T) {
	h := Auth(stubParser{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatal(rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatal(rr.Code)
	}
}

func TestWithUserID(t *testing.T) {
	ctx := WithUserID(httptest.NewRequest(http.MethodGet, "/", nil).Context(), 9)
	uid, ok := UserIDFromContext(ctx)
	if !ok || uid != 9 {
		t.Fatal(uid, ok)
	}
}
