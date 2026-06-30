package passenger

import "errors"

var (
	ErrPassengerNotFound   = errors.New("passenger not found")
	ErrPassengerBlocked    = errors.New("passenger blocked")
	ErrInvalidCode         = errors.New("invalid code")
	ErrCodeExpired         = errors.New("code expired")
	ErrCodeAlreadyUsed     = errors.New("code already used")
	ErrTooManyAttempts     = errors.New("too many attempts")
	ErrInvalidToken        = errors.New("invalid passenger token")
	ErrInvalidRefreshToken = errors.New("invalid passenger refresh token")
)
