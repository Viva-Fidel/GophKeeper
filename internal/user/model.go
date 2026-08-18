// Package user реализует регистрацию и аутентификацию пользователей.
package user

// User — учётная запись в хранилище.
type User struct {
	ID             int64
	Login          string
	PasswordHash   string
	EncryptionSalt []byte
}
