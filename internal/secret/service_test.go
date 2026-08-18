package secret

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

type memRepo struct {
	items map[string]Secret
}

func newMemRepo() *memRepo {
	return &memRepo{items: map[string]Secret{}}
}

func (m *memRepo) Upsert(_ context.Context, s Secret) (*Secret, error) {
	if old, ok := m.items[s.ID]; ok {
		if old.UserID != s.UserID {
			return nil, ErrConflict
		}
		if s.Version < old.Version {
			return nil, ErrConflict
		}
	}
	s.UpdatedAt = time.Now().UTC()
	m.items[s.ID] = s
	out := s
	return &out, nil
}

func (m *memRepo) GetByID(_ context.Context, userID int64, id string) (*Secret, error) {
	s, ok := m.items[id]
	if !ok || s.UserID != userID {
		return nil, ErrNotFound
	}
	out := s
	return &out, nil
}

func (m *memRepo) ListByUser(_ context.Context, userID int64) ([]Secret, error) {
	var out []Secret
	for _, s := range m.items {
		if s.UserID == userID && !s.Deleted {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memRepo) ListUpdatedSince(_ context.Context, userID int64, since time.Time) ([]Secret, error) {
	var out []Secret
	for _, s := range m.items {
		if s.UserID == userID && s.UpdatedAt.After(since) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memRepo) SoftDelete(ctx context.Context, userID int64, id string, version int64) (*Secret, error) {
	s, err := m.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if s.Deleted {
		return s, nil
	}
	if version <= s.Version {
		return nil, ErrConflict
	}
	s.Deleted = true
	s.Version = version
	s.UpdatedAt = time.Now().UTC()
	m.items[id] = *s
	return s, nil
}

func TestServiceCRUDAndSync(t *testing.T) {
	svc := New(newMemRepo())
	ct := base64.StdEncoding.EncodeToString([]byte("cipher"))
	sec, err := svc.Upsert(context.Background(), 1, UpsertRequest{
		Type: TypeText, Title: "note", Ciphertext: ct, Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), 1, sec.ID)
	if err != nil || got.Title != "note" {
		t.Fatal(err, got)
	}
	list, err := svc.List(context.Background(), 1)
	if err != nil || len(list) != 1 {
		t.Fatal(err, list)
	}
	if _, err := svc.Upsert(context.Background(), 1, UpsertRequest{Type: "bad", Ciphertext: ct}); !errors.Is(err, ErrInvalidType) {
		t.Fatal(err)
	}
	del, err := svc.Delete(context.Background(), 1, sec.ID, 0)
	if err != nil || !del.Deleted {
		t.Fatal(err, del)
	}
	if _, err := svc.Get(context.Background(), 1, sec.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}

	resp, err := svc.Sync(context.Background(), 1, SyncRequest{
		Since: time.Time{},
		Changes: []UpsertRequest{{
			ID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", Type: TypeBinary, Title: "bin",
			Ciphertext: ct, Version: 1,
		}},
	})
	if err != nil || len(resp.Secrets) == 0 {
		t.Fatal(err, resp)
	}
}

func TestServiceSyncInvalid(t *testing.T) {
	svc := New(newMemRepo())
	_, err := svc.Sync(context.Background(), 1, SyncRequest{
		Changes: []UpsertRequest{{ID: "x", Type: "bad"}},
	})
	if !errors.Is(err, ErrInvalidType) {
		t.Fatal(err)
	}
}
