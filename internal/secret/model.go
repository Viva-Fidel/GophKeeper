// Package secret реализует хранение и синхронизацию приватных данных.
package secret

import "time"

// Type — тип хранимой записи.
type Type string

const (
	// TypeLoginPassword — пара логин/пароль.
	TypeLoginPassword Type = "login_password"
	// TypeText — произвольный текст.
	TypeText Type = "text"
	// TypeBinary — произвольные бинарные данные.
	TypeBinary Type = "binary"
	// TypeBankCard — данные банковской карты.
	TypeBankCard Type = "bank_card"
)

// ValidType проверяет допустимость типа секрета.
func ValidType(t Type) bool {
	switch t {
	case TypeLoginPassword, TypeText, TypeBinary, TypeBankCard:
		return true
	default:
		return false
	}
}

// Secret — запись хранилища (на сервере ciphertext непрозрачен).
type Secret struct {
	ID         string
	UserID     int64
	Type       Type
	Title      string
	Ciphertext []byte
	Version    int64
	UpdatedAt  time.Time
	Deleted    bool
}
