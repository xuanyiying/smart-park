package tenant

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
	"github.com/xuanyiying/smart-park/pkg/auth"
	authmiddleware "github.com/xuanyiying/smart-park/pkg/middleware"
)

func activeTestTenant(id uuid.UUID, code string) *biz.Tenant {
	return &biz.Tenant{
		ID:     id,
		Code:   code,
		Name:   "Tenant " + code,
		Status: "active",
	}
}

// tenantContext builds a server context carrying the given tenant header and, when
// claims is non-nil, an authenticated session.
func tenantContext(headerTenant string, claims *auth.Claims) context.Context {
	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", headerTenant)
	ctx := transport.NewServerContext(context.Background(), tr)
	if claims != nil {
		ctx = context.WithValue(ctx, authmiddleware.JWTClaimsKey, claims)
	}
	return ctx
}

// TestTenantMiddlewareRejectsCrossTenantHeader guards the vulnerability the audit found:
// the tenant identifier came from a client-controlled header, and the middleware only
// checked that the tenant existed. Any authenticated user could therefore read another
// tenant's data by editing X-Tenant-ID.
func TestTenantMiddlewareRejectsCrossTenantHeader(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	loader := newMockTenantLoader()
	loader.addTenant(activeTestTenant(tenantA, "acme"))
	loader.addTenant(activeTestTenant(tenantB, "globex"))

	mw := TenantMiddleware(loader, NewHeaderExtractor("X-Tenant-ID"), log.NewStdLogger(io.Discard))

	// Authenticated as a member of tenant A, but requesting tenant B's data.
	ctx := tenantContext(tenantB.String(), &auth.Claims{UserID: "user-1", TenantID: tenantA.String()})

	handlerCalled := false
	_, err := mw(func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "ok", nil
	})(ctx, nil)

	require.Error(t, err, "a cross-tenant request must be rejected")
	assert.False(t, handlerCalled, "the handler must not run for another tenant's data")
}

// TestTenantMiddlewareAllowsMatchingTenant keeps the legitimate path working.
func TestTenantMiddlewareAllowsMatchingTenant(t *testing.T) {
	tenantA := uuid.New()

	loader := newMockTenantLoader()
	loader.addTenant(activeTestTenant(tenantA, "acme"))

	mw := TenantMiddleware(loader, NewHeaderExtractor("X-Tenant-ID"), log.NewStdLogger(io.Discard))

	ctx := tenantContext(tenantA.String(), &auth.Claims{UserID: "user-1", TenantID: tenantA.String()})

	var seenTenant uuid.UUID
	_, err := mw(func(ctx context.Context, req interface{}) (interface{}, error) {
		seenTenant = TenantIDFromCtx(ctx)
		return "ok", nil
	})(ctx, nil)

	require.NoError(t, err)
	assert.Equal(t, tenantA, seenTenant, "the resolved tenant must be available to the handler")
}

// TestTenantMiddlewareResolvesByCodeWhenUnauthenticated covers service-to-service callers
// that carry no user session: they still get a validated tenant.
func TestTenantMiddlewareResolvesByCodeWhenUnauthenticated(t *testing.T) {
	tenantA := uuid.New()

	loader := newMockTenantLoader()
	loader.addTenant(activeTestTenant(tenantA, "acme"))

	mw := TenantMiddleware(loader, NewHeaderExtractor("X-Tenant-ID"), log.NewStdLogger(io.Discard))

	ctx := tenantContext("acme", nil)

	var seenTenant uuid.UUID
	_, err := mw(func(ctx context.Context, req interface{}) (interface{}, error) {
		seenTenant = TenantIDFromCtx(ctx)
		return "ok", nil
	})(ctx, nil)

	require.NoError(t, err)
	assert.Equal(t, tenantA, seenTenant)
}

// TestTenantMiddlewareRejectsUnknownTenant ensures a fabricated tenant id is refused
// rather than silently falling through to an unscoped query.
func TestTenantMiddlewareRejectsUnknownTenant(t *testing.T) {
	loader := newMockTenantLoader()

	mw := TenantMiddleware(loader, NewHeaderExtractor("X-Tenant-ID"), log.NewStdLogger(io.Discard))

	ctx := tenantContext(uuid.New().String(), nil)

	_, err := mw(func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	})(ctx, nil)

	require.Error(t, err, "an unknown tenant must be rejected")
}
