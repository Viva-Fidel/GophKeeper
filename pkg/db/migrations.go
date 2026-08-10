package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var migrationUpPattern = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

type migration struct {
	version int
	name    string
	path    string
}

// RunMigrations применяет все новые .up.sql миграции из указанной директории.
func RunMigrations(ctx context.Context, db *sql.DB, dir string) error {
	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		return err
	}

	files, err := loadUpMigrations(dir)
	if err != nil {
		return err
	}

	for _, m := range files {
		var alreadyApplied bool
		if err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			m.version,
		).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("failed to check migration %d: %w", m.version, err)
		}
		if alreadyApplied {
			continue
		}

		sqlBytes, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("failed to read migration %q: %w", m.path, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to start tx for migration %d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.ErrorContext(ctx, "migration rollback after exec error",
					slog.Int("version", m.version),
					slog.Any("rollback_error", rbErr),
					slog.Any("exec_error", err),
				)
			}
			return fmt.Errorf("failed to apply migration %d: %w", m.version, err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, name) VALUES ($1, $2)`,
			m.version,
			m.name,
		); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.ErrorContext(ctx, "migration rollback after version insert error",
					slog.Int("version", m.version),
					slog.Any("rollback_error", rbErr),
					slog.Any("exec_error", err),
				)
			}
			return fmt.Errorf("failed to save migration version %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.version, err)
		}
	}

	return nil
}

// ensureSchemaMigrationsTable создаёт таблицу учёта миграций при необходимости.
func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}
	return nil
}

// loadUpMigrations читает и сортирует файлы *.up.sql из директории.
func loadUpMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations dir %q: %w", dir, err)
	}

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		match := migrationUpPattern.FindStringSubmatch(name)
		if len(match) != 2 {
			continue
		}
		version, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("bad migration version in %q: %w", name, err)
		}
		migrations = append(migrations, migration{
			version: version,
			name:    name,
			path:    filepath.Join(dir, name),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}
