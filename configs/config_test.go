package config

import (
	"flag"
	"testing"
)

func TestParseFlags(t *testing.T) {
	conf := &Config{
		Server: ServerConfig{Address: ":8080"},
		Db:     DbConfig{DatabaseURI: "postgres://x"},
		Auth:   AuthConfig{JWTSecret: "s", TokenExp: "1h"},
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flags, err := parseFlags(conf, fs, []string{"-a", ":9090"})
	if err != nil {
		t.Fatal(err)
	}
	if flags.RunAddress != ":9090" {
		t.Fatal(flags.RunAddress)
	}
}

func TestParseFlagsRequiresDB(t *testing.T) {
	conf := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	if _, err := parseFlags(conf, fs, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadConfig(t *testing.T) {
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Address == "" {
		t.Fatal("empty address")
	}
}
