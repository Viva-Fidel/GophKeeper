package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestClientVersionBinary(t *testing.T) {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "client")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v: %s", err, out)
	}
	out, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	if len(out) == 0 {
		t.Fatal("empty version output")
	}
}

func TestClientUsageExit(t *testing.T) {
	if os.Getenv("BE_CLIENT") == "1" {
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestClientUsageExit")
	cmd.Env = append(os.Environ(), "BE_CLIENT=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
}
