package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInactiveUser       = errors.New("inactive user")
)
