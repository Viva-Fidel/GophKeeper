package config

import (
	"os"
	"testing"
)

func TestLoadFlagsAllowsMissingDB(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	_ = os.Unsetenv("DATABASE_URI")
	_ = os.Setenv("JWT_SECRET", "test-secret")
	t.Cleanup(func() { _ = os.Unsetenv("JWT_SECRET") })
	flags, err := LoadFlags()
	if err != nil {
		t.Fatal(err)
	}
	if flags.DatabaseURI != "" {
		t.Fatal(flags.DatabaseURI)
	}
}

func TestLoadFlagsMissingJWTSecret(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	_ = os.Setenv("DATABASE_URI", "postgres://x")
	_ = os.Unsetenv("JWT_SECRET")
	t.Cleanup(func() { _ = os.Unsetenv("DATABASE_URI") })
	if _, err := LoadFlags(); err == nil {
		t.Fatal("expected error")
	}
}
