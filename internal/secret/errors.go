package secret

import "errors"

// ErrNotFound — секрет не найден.
var ErrNotFound = errors.New("secret not found")

// ErrConflict — конфликт версий при обновлении.
var ErrConflict = errors.New("version conflict")

// ErrInvalidType — недопустимый тип секрета.
var ErrInvalidType = errors.New("invalid secret type")

// ErrUnauthorized — нет доступа к секрету.
var ErrUnauthorized = errors.New("unauthorized")
