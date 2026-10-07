package auth

import (
	"errors"
	"time"
)

var (
	ErrWrongIdentificator = errors.New("wrong identificator given")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExist   = errors.New("user with this username or email already exists")
	ErrPasswordMismatch   = errors.New("password mismatch")
	ErrTooLongPassword    = errors.New("too long password")
	ErrEmptyUsername      = errors.New("empty username given")
	ErrTooShortPassword   = errors.New("too short password")
)

type User struct {
	Id           int64
	Name         string
	Surname      string
	Email        string
	PasswordHash string
	IsActive     bool
	balance      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type LoginResult struct {
	user        User
	AccessToken string
	ExpiresAt   time.Time
}
