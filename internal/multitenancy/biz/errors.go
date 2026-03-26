package biz

import "errors"

var (
	// ErrTenantNotFound is returned when tenant is not found
	ErrTenantNotFound = errors.New("tenant not found")

	// ErrTenantInvalid is returned when tenant is invalid or expired
	ErrTenantInvalid = errors.New("tenant is invalid or expired")

	// ErrTenantDisabled is returned when tenant is disabled
	ErrTenantDisabled = errors.New("tenant is disabled")

	// ErrTenantExpired is returned when tenant subscription has expired
	ErrTenantExpired = errors.New("tenant subscription has expired")

	// ErrQuotaExceeded is returned when tenant exceeds resource quota
	ErrQuotaExceeded = errors.New("tenant quota exceeded")

	// ErrFeatureNotAvailable is returned when tenant doesn't have access to a feature
	ErrFeatureNotAvailable = errors.New("feature not available for this tenant")

	// ErrDuplicateTenantCode is returned when tenant code already exists
	ErrDuplicateTenantCode = errors.New("tenant code already exists")

	// ErrInvalidTenantCode is returned when tenant code is invalid
	ErrInvalidTenantCode = errors.New("invalid tenant code")
)
