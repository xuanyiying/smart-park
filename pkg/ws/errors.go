package ws

import "errors"

var (
	ErrMissingToken = errors.New("missing authentication token")
	ErrInvalidToken = errors.New("invalid authentication token")
	ErrRateLimited  = errors.New("connection rate limit exceeded")
)
