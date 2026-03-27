package biz

import "errors"

var (
	// ErrStationNotFound is returned when charging station is not found
	ErrStationNotFound = errors.New("charging station not found")

	// ErrConnectorNotFound is returned when charging connector is not found
	ErrConnectorNotFound = errors.New("charging connector not found")

	// ErrSessionNotFound is returned when charging session is not found
	ErrSessionNotFound = errors.New("charging session not found")

	// ErrConnectorNotAvailable is returned when connector is not available
	ErrConnectorNotAvailable = errors.New("connector not available")

	// ErrConnectorInUse is returned when connector is already in use
	ErrConnectorInUse = errors.New("connector already in use")

	// ErrSessionNotActive is returned when session is not active
	ErrSessionNotActive = errors.New("charging session not active")

	// ErrInvalidEnergyValue is returned when energy value is invalid
	ErrInvalidEnergyValue = errors.New("invalid energy value")

	// ErrInvalidPowerValue is returned when power value is invalid
	ErrInvalidPowerValue = errors.New("invalid power value")

	// ErrPriceNotConfigured is returned when price is not configured
	ErrPriceNotConfigured = errors.New("charging price not configured")

	// ErrStationNotAvailable is returned when station is not available
	ErrStationNotAvailable = errors.New("station not available")

	// ErrInvalidDuration is returned when duration is invalid
	ErrInvalidDuration = errors.New("invalid charging duration")

	// ErrPaymentFailed is returned when payment processing failed
	ErrPaymentFailed = errors.New("payment processing failed")

	// ErrUnauthorizedAccess is returned when user is not authorized
	ErrUnauthorizedAccess = errors.New("unauthorized access to charging session")
)
