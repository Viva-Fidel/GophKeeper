package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
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
	if _, err := repo.CreateUser(context.Background(), "alice", "hash", []byte("salt")); !errors.Is(err, ErrUserExists) {
		t.Fatal(err)
	}

	mock.ExpectQuery(`SELECT id, login, password_hash, encryption_salt`).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetUserByLogin(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

var errDup = &pgconn.PgError{Code: "23505"}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(errDup) {
		t.Fatal("expected unique")
	}
	if !isUniqueViolation(errors.Join(errors.New("wrap"), errDup)) {
		t.Fatal("expected unique when wrapped")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("foreign key is not unique")
	}
	if isUniqueViolation(errors.New("duplicate key")) {
		t.Fatal("text match must not succeed")
	}
	if isUniqueViolation(nil) {
		t.Fatal("nil")
	}
}
