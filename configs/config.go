// Package config загружает конфигурацию сервера из переменных окружения и флагов.
package config

import (
	"errors"
	"flag"
	"os"

	"github.com/caarlos0/env/v11"
)

// Config — конфигурация из окружения.
type Config struct {
	Server ServerConfig
	Db     DbConfig
	Auth   AuthConfig
}

// ServerConfig описывает сетевые параметры сервера.
type ServerConfig struct {
	Address string `env:"RUN_ADDRESS" envDefault:":8080"`
}

// DbConfig описывает подключение к PostgreSQL.
type DbConfig struct {
	DatabaseURI string `env:"DATABASE_URI"`
}

// AuthConfig описывает параметры JWT.
type AuthConfig struct {
	JWTSecret string `env:"JWT_SECRET" envDefault:"gophkeeper-dev-secret"`
	TokenExp  string `env:"TOKEN_EXP" envDefault:"24h"`
}

// loadConfig читает конфигурацию сервера из переменных окружения.
func loadConfig() (*Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Flags — итоговые параметры запуска сервера.
type Flags struct {
	RunAddress  string
	DatabaseURI string
	JWTSecret   string
	TokenExp    string
}

// parseFlags накладывает CLI-флаги поверх значений из окружения.
func parseFlags(conf *Config, fs *flag.FlagSet, args []string) (*Flags, error) {
	flags := &Flags{
		RunAddress:  conf.Server.Address,
		DatabaseURI: conf.Db.DatabaseURI,
		JWTSecret:   conf.Auth.JWTSecret,
		TokenExp:    conf.Auth.TokenExp,
	}

	fs.StringVar(&flags.RunAddress, "a", flags.RunAddress, "service run address")
	fs.StringVar(&flags.DatabaseURI, "d", flags.DatabaseURI, "postgres connection uri")
	fs.StringVar(&flags.JWTSecret, "j", flags.JWTSecret, "jwt signing secret")
	fs.StringVar(&flags.TokenExp, "t", flags.TokenExp, "jwt token ttl")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if flags.DatabaseURI == "" {
		return nil, errors.New("database URI is required")
	}
	return flags, nil
}

// LoadFlags загружает конфигурацию из окружения и CLI-флагов.
func LoadFlags() (*Flags, error) {
	conf, err := loadConfig()
	if err != nil {
		return nil, err
	}
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	return parseFlags(conf, fs, os.Args[1:])
}
