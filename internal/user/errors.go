package user

import "errors"

// ErrUserExists — пользователь с таким логином уже есть.
var ErrUserExists = errors.New("user already exists")

// ErrInvalidCredentials — неверный логин или пароль.
var ErrInvalidCredentials = errors.New("invalid credentials")
