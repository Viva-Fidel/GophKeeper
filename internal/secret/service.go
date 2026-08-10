package secret

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

// Repository — интерфейс хранилища секретов.
type Repository interface {
	Upsert(ctx context.Context, s Secret) (*Secret, error)
	GetByID(ctx context.Context, userID int64, id string) (*Secret, error)
	ListByUser(ctx context.Context, userID int64) ([]Secret, error)
	ListUpdatedSince(ctx context.Context, userID int64, since time.Time) ([]Secret, error)
	SoftDelete(ctx context.Context, userID int64, id string, version int64) (*Secret, error)
}

// Service — бизнес-логика секретов и синхронизации.
type Service struct {
	repo Repository
}

// New создаёт сервис секретов.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// Upsert создаёт или обновляет секрет владельца.
func (s *Service) Upsert(ctx context.Context, userID int64, req UpsertRequest) (*Secret, error) {
	if !ValidType(req.Type) {
		return nil, ErrInvalidType
	}
	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}
	version := req.Version
	if version <= 0 {
		version = 1
	}
	raw, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, Secret{
		ID:         id,
		UserID:     userID,
		Type:       req.Type,
		Title:      req.Title,
		Ciphertext: raw,
		Version:    version,
		Deleted:    false,
	})
}

// Get возвращает секрет по id.
func (s *Service) Get(ctx context.Context, userID int64, id string) (*Secret, error) {
	sec, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if sec.Deleted {
		return nil, ErrNotFound
	}
	return sec, nil
}

// List возвращает активные секреты пользователя.
func (s *Service) List(ctx context.Context, userID int64) ([]Secret, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Delete мягко удаляет секрет.
func (s *Service) Delete(ctx context.Context, userID int64, id string, version int64) (*Secret, error) {
	if version <= 0 {
		existing, err := s.repo.GetByID(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		version = existing.Version + 1
	}
	return s.repo.SoftDelete(ctx, userID, id, version)
}

// Sync применяет локальные изменения клиента и возвращает изменения сервера.
func (s *Service) Sync(ctx context.Context, userID int64, req SyncRequest) (*SyncResponse, error) {
	for _, ch := range req.Changes {
		if ch.ID == "" {
			continue
		}
		if !ValidType(ch.Type) {
			return nil, ErrInvalidType
		}
		raw, err := base64.StdEncoding.DecodeString(ch.Ciphertext)
		if err != nil {
			return nil, err
		}
		version := ch.Version
		if version <= 0 {
			version = 1
		}
		_, err = s.repo.Upsert(ctx, Secret{
			ID:         ch.ID,
			UserID:     userID,
			Type:       ch.Type,
			Title:      ch.Title,
			Ciphertext: raw,
			Version:    version,
			Deleted:    ch.Deleted,
		})
		if err != nil && err != ErrConflict {
			return nil, err
		}
	}

	secrets, err := s.repo.ListUpdatedSince(ctx, userID, req.Since)
	if err != nil {
		return nil, err
	}
	dtos := make([]SecretDTO, 0, len(secrets))
	for _, sec := range secrets {
		dtos = append(dtos, ToDTO(sec))
	}
	return &SyncResponse{Secrets: dtos, ServerTime: time.Now().UTC()}, nil
}
