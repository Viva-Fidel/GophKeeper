package secret

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSecretRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSecretRepository(db)
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO secrets`).
		WithArgs("id1", int64(1), TypeText, "t", []byte("c"), int64(1), false).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "ciphertext", "version", "updated_at", "deleted",
		}).AddRow("id1", 1, TypeText, "t", []byte("c"), 1, now, false))
	sec, err := repo.Upsert(context.Background(), Secret{
		ID: "id1", UserID: 1, Type: TypeText, Title: "t", Ciphertext: []byte("c"), Version: 1,
	})
	if err != nil || sec.ID != "id1" {
		t.Fatal(err, sec)
	}

	mock.ExpectQuery(`SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted`).
		WithArgs("id1", int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "ciphertext", "version", "updated_at", "deleted",
		}).AddRow("id1", 1, TypeText, "t", []byte("c"), 1, now, false))
	got, err := repo.GetByID(context.Background(), 1, "id1")
	if err != nil || got.Title != "t" {
		t.Fatal(err, got)
	}

	mock.ExpectQuery(`SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "ciphertext", "version", "updated_at", "deleted",
		}).AddRow("id1", 1, TypeText, "t", []byte("c"), 1, now, false))
	list, err := repo.ListByUser(context.Background(), 1)
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}

	mock.ExpectQuery(`SELECT id, user_id, type, title, ciphertext, version, updated_at, deleted`).
		WithArgs(int64(1), time.Time{}).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "ciphertext", "version", "updated_at", "deleted",
		}).AddRow("id1", 1, TypeText, "t", []byte("c"), 1, now, false))
	upd, err := repo.ListUpdatedSince(context.Background(), 1, time.Time{})
	if err != nil || len(upd) != 1 {
		t.Fatal(err, upd)
	}

	mock.ExpectQuery(`UPDATE secrets`).
		WithArgs("id1", int64(1), int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "type", "title", "ciphertext", "version", "updated_at", "deleted",
		}).AddRow("id1", 1, TypeText, "t", []byte("c"), 2, now, true))
	del, err := repo.SoftDelete(context.Background(), 1, "id1", 2)
	if err != nil || !del.Deleted {
		t.Fatal(err, del)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
