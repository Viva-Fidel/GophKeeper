package user

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewUserRepository(db)

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("alice", "hash", []byte("salt")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	id, err := repo.CreateUser(context.Background(), "alice", "hash", []byte("salt"))
	if err != nil || id != 1 {
		t.Fatal(err, id)
	}

	mock.ExpectQuery(`SELECT id, login, password_hash, encryption_salt`).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"id", "login", "password_hash", "encryption_salt"}).
			AddRow(1, "alice", "hash", []byte("salt")))
	u, err := repo.GetUserByLogin(context.Background(), "alice")
	if err != nil || u.Login != "alice" {
		t.Fatal(err, u)
	}

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("alice", "hash", []byte("salt")).
		WillReturnError(errDup)
	if _, err := repo.CreateUser(context.Background(), "alice", "hash", []byte("salt")); err == nil {
		t.Fatal("expected exists")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type dupErr struct{}

func (dupErr) Error() string { return "duplicate key value violates unique constraint" }

var errDup = dupErr{}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(errDup) {
		t.Fatal("expected unique")
	}
	if isUniqueViolation(nil) {
		t.Fatal("nil")
	}
}
