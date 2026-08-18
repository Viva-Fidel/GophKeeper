// Package main — точка входа сервера GophKeeper.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "gophkeeper/configs"
	"gophkeeper/internal/httpserver"
	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
	"gophkeeper/pkg/db"
)

// main загружает конфигурацию, обрабатывает сигналы ОС и запускает HTTP-сервер.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	conf, err := config.LoadFlags()
	if err != nil {
		slog.Error("failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, conf, "migrations"); err != nil {
		slog.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}

// run подключается к БД, применяет миграции, поднимает HTTP API
// и блокируется до отмены ctx либо ошибки Serve.
func run(ctx context.Context, conf *config.Flags, migrationsDir string) error {
	if conf.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}

	if err := db.RunMigrations(ctx, conf.DatabaseURI, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	database, err := db.Open(ctx, conf.DatabaseURI)
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("db close", slog.Any("error", err))
		}
	}()

	tokenTTL, err := time.ParseDuration(conf.TokenExp)
	if err != nil {
		return fmt.Errorf("parse token TTL: %w", err)
	}

	userSvc := user.New(user.NewUserRepository(database), conf.JWTSecret, tokenTTL)
	secretSvc := secret.New(secret.NewSecretRepository(database))
	handler := httpserver.New(userSvc, secretSvc).Router()

	ln, err := net.Listen("tcp", conf.RunAddress)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	httpSrv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	slog.Info("gophkeeper started", slog.String("addr", ln.Addr().String()))

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		<-errCh
		slog.Info("gophkeeper stopped")
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http serve: %w", err)
		}
		return nil
	}
}
