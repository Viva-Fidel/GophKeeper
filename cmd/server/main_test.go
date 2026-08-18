package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	config "gophkeeper/configs"
)

func TestRunEmptyDatabaseURI(t *testing.T) {
	err := run(context.Background(), &config.Flags{
		RunAddress:  "127.0.0.1:0",
		DatabaseURI: "",
		JWTSecret:   "test-secret",
		TokenExp:    "1h",
	}, "migrations")
	if err == nil {
		t.Fatal("expected empty DATABASE_URI error")
	}
}

func TestRunEmptyJWTSecret(t *testing.T) {
	err := run(context.Background(), &config.Flags{
		RunAddress:  "127.0.0.1:0",
		DatabaseURI: "postgres://x",
		JWTSecret:   "",
		TokenExp:    "1h",
	}, "migrations")
	if err == nil || err.Error() != "JWT_SECRET is required" {
		t.Fatalf("expected JWT_SECRET error, got %v", err)
	}
}

func TestMainExitsWithoutFlags(t *testing.T) {
	if os.Getenv("BE_SERVER") == "1" {
		oldArgs := os.Args
		os.Args = []string{"gophkeeper-server"}
		defer func() { os.Args = oldArgs }()
		main()
		return
	}

	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "DATABASE_URI=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "BE_SERVER=1")

	cmd := exec.Command(os.Args[0], "-test.run=TestMainExitsWithoutFlags")
	cmd.Env = env
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit without DATABASE_URI")
	}
}
