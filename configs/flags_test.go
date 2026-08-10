package config

import (
	"os"
	"testing"
)

func TestLoadFlagsMissingDB(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	_ = os.Unsetenv("DATABASE_URI")
	if _, err := LoadFlags(); err == nil {
		t.Fatal("expected error")
	}
}
