// Package biz provides business logic for the charging service.
package biz

import "errors"

// Common errors for charging service
var (
	// Station errors
	ErrStationNotFound     = errors.New("station not found")
	ErrStationNotAvailable = errors.New("station not available")
	
	// Connector errors
	ErrConnectorNotFound     = errors.New("connector not found")
	ErrConnectorNotAvailable = errors.New("connector not available")
	ErrConnectorInUse        = errors.New("connector in use")
	
	// Session errors
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionNotActive   = errors.New("session not active")
	ErrUnauthorizedAccess = errors.New("unauthorized access")
	
	// Price errors
	ErrPriceNotConfigured = errors.New("price not configured")
	
	// Validation errors
	ErrInvalidPowerValue  = errors.New("invalid power value")
	ErrInvalidEnergyValue = errors.New("invalid energy value")
)