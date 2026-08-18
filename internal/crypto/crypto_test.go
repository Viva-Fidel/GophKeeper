package crypto

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatal(err)
	}
	key := DeriveKey("master", salt, 1000)
	ct, err := Encrypt(key, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	pt, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, []byte("hello")) {
		t.Fatalf("got %q", pt)
	}
}

func TestEncryptInvalidKey(t *testing.T) {
	if _, err := Encrypt([]byte("short"), []byte("x")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Decrypt([]byte("short"), []byte("x")); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := DeriveKey("p", make([]byte, SaltSize), 1)
	if _, err := Decrypt(key, []byte("x")); err == nil {
		t.Fatal("expected error")
	}
}

func TestBase64(t *testing.T) {
	s := EncodeBase64([]byte("ab"))
	b, err := DecodeBase64(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ab" {
		t.Fatal(string(b))
	}
}

func TestDeriveKeyDefaultIterations(t *testing.T) {
	k := DeriveKey("p", make([]byte, SaltSize), 0)
	if len(k) != KeySize {
		t.Fatal(len(k))
	}
}
