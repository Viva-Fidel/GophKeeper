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
	if err := pingWithRetry(context.Background(), db, 3, time.Millisecond); err != nil {
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
	if err := pingWithRetry(context.Background(), db, 2, time.Millisecond); err == nil {
		t.Fatal("expected error")
	}
}

type pingErr struct{}

func (pingErr) Error() string { return "ping fail" }

var errPing = pingErr{}
