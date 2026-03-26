package multitenancy

import (
	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

var (
	// ErrTenantNotFound is returned when tenant is not found
	ErrTenantNotFound = biz.ErrTenantNotFound

	// ErrTenantInvalid is returned when tenant is invalid or expired
	ErrTenantInvalid = biz.ErrTenantInvalid

	// ErrTenantDisabled is returned when tenant is disabled
	ErrTenantDisabled = biz.ErrTenantDisabled

	// ErrTenantExpired is returned when tenant subscription has expired
	ErrTenantExpired = biz.ErrTenantExpired

	// ErrQuotaExceeded is returned when tenant exceeds resource quota
	ErrQuotaExceeded = biz.ErrQuotaExceeded

	// ErrFeatureNotAvailable is returned when tenant doesn't have access to a feature
	ErrFeatureNotAvailable = biz.ErrFeatureNotAvailable

	// ErrDuplicateTenantCode is returned when tenant code already exists
	ErrDuplicateTenantCode = biz.ErrDuplicateTenantCode

	// ErrInvalidTenantCode is returned when tenant code is invalid
	ErrInvalidTenantCode = biz.ErrInvalidTenantCode
)
