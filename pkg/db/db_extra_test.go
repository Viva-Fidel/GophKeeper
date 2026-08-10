package db

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPingWithRetrySuccess(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectPing()
	if err := pingWithRetry(db, 3, time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestPingWithRetryFail(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectPing().WillReturnError(errPing)
	mock.ExpectPing().WillReturnError(errPing)
	if err := pingWithRetry(db, 2, time.Millisecond); err == nil {
		t.Fatal("expected error")
	}
}

type pingErr struct{}

func (pingErr) Error() string { return "ping fail" }

var errPing = pingErr{}

func TestOpenPingFail(t *testing.T) {
	// Open uses real sql.Open; empty already tested. Ensure ensureSchemaMigrationsTable error path.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).WillReturnError(errPing)
	if err := ensureSchemaMigrationsTable(context.Background(), db); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadUpMigrationsBadDir(t *testing.T) {
	if _, err := loadUpMigrations("/no/such/dir/gophkeeper"); err == nil {
		t.Fatal("expected error")
	}
}
