package mq

import "errors"

var (
	ErrNotImplemented = errors.New("operation not implemented")
	ErrInvalidConfig  = errors.New("invalid config")
	ErrConnectionLost = errors.New("connection lost")
)
