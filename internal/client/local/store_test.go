package local

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gophkeeper/internal/secret"
)

func TestStoreSessionAndVault(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	sess := Session{ServerURL: "http://x", Login: "u", Token: "t", SaltB64: "c2FsdA==", LastSync: time.Now().UTC()}
	if err := store.SaveSession(sess); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadSession()
	if err != nil || got.Login != "u" {
		t.Fatal(err, got)
	}
	items := []LocalSecret{{ID: "1", Type: secret.TypeText, Title: "n", Ciphertext: "YQ==", Version: 1}}
	if err := store.SaveVault(items); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadVault()
	if err != nil || len(loaded) != 1 {
		t.Fatal(err, loaded)
	}
	if err := store.ClearSession(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadSession(); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatal(dir)
	}
}

func TestLoadVaultMissing(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	items, err := store.LoadVault()
	if err != nil || len(items) != 0 {
		t.Fatal(err, items)
	}
}
