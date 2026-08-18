package config

import (
	"os"
	"testing"
)

func TestLoadFlagsAllowsMissingDB(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	t.Setenv("DATABASE_URI", "")
	flags, err := LoadFlags()
	if err != nil {
		t.Fatal(err)
	}
	if flags.DatabaseURI != "" {
		t.Fatal(flags.DatabaseURI)
	}
}

func TestLoadFlagsAllowsMissingJWTSecret(t *testing.T) {
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = []string{"gophkeeper"}
	t.Setenv("JWT_SECRET", "")
	flags, err := LoadFlags()
	if err != nil {
		t.Fatal(err)
	}
	if flags.JWTSecret != "" {
		t.Fatal(flags.JWTSecret)
	}
}
