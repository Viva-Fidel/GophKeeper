package auth

import (
	"testing"
	"time"
)

func TestNewAndParseToken(t *testing.T) {
	token, err := NewToken(42, "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 {
		t.Fatalf("got uid %d", claims.UserID)
	}
}

func TestParseTokenInvalid(t *testing.T) {
	if _, err := ParseToken("bad", "secret"); err == nil {
		t.Fatal("expected error")
	}
	token, err := NewToken(1, "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken(token, "other"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("pass")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "pass") {
		t.Fatal("verify failed")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
}
