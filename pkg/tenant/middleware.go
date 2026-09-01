package tenant

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
	"github.com/xuanyiying/smart-park/pkg/auth"
	authmiddleware "github.com/xuanyiying/smart-park/pkg/middleware"
)

type TenantExtractor interface {
	Extract(ctx context.Context) (string, error)
}

type HeaderExtractor struct {
	HeaderName string
}

func NewHeaderExtractor(headerName string) *HeaderExtractor {
	if headerName == "" {
		headerName = "X-Tenant-ID"
	}
	return &HeaderExtractor{HeaderName: headerName}
}

func (e *HeaderExtractor) Extract(ctx context.Context) (string, error) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", nil
	}
	val := tr.RequestHeader().Get(e.HeaderName)
	return val, nil
}

type SubdomainExtractor struct{}

func NewSubdomainExtractor() *SubdomainExtractor {
	return &SubdomainExtractor{}
}

func (e *SubdomainExtractor) Extract(ctx context.Context) (string, error) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", nil
	}
	ht, ok := tr.(*khttp.Transport)
	if !ok {
		return "", nil
	}
	host := ht.Request().Host
	host = strings.Split(host, ":")[0]
	parts := strings.Split(host, ".")
	if len(parts) > 2 {
		return parts[0], nil
	}
	return "", nil
}

type TokenExtractor struct{}

func NewTokenExtractor() *TokenExtractor {
	return &TokenExtractor{}
}

func (e *TokenExtractor) Extract(ctx context.Context) (string, error) {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return "", nil
	}
	authHeader := tr.RequestHeader().Get("Authorization")
	if authHeader == "" {
		return "", nil
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1], nil
	}
	return "", nil
}

type TenantLoader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*biz.Tenant, error)
	GetByCode(ctx context.Context, code string) (*biz.Tenant, error)
}

// tenantFromClaims reads the tenant bound to the authenticated session.
//
// It reads the claims the JWT middleware stores under the same context key, so this works
// whether or not the tenant middleware is chained behind it.
func tenantFromClaims(ctx context.Context) string {
	claims, ok := ctx.Value(authmiddleware.JWTClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		return ""
	}
	return claims.TenantID
}

// sameTenant compares two tenant identifiers, tolerating UUID case differences.
func sameTenant(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func TenantMiddleware(loader TenantLoader, extractor TenantExtractor, logger log.Logger) middleware.Middleware {
	helper := log.NewHelper(logger)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			identifier, err := extractor.Extract(ctx)
			if err != nil {
				helper.WithContext(ctx).Warnf("tenant extractor error: %v", err)
				return nil, errors.Forbidden("TENANT_INVALID", "failed to extract tenant identifier")
			}
			if identifier == "" {
				return nil, errors.Forbidden("TENANT_REQUIRED", "tenant identifier is required")
			}

			// The identifier usually comes from a request header, which the client controls.
			// When the caller is authenticated, the token already binds them to a tenant;
			// trusting a different header value would let any logged-in user read another
			// tenant's data, so the two must agree.
			if authenticatedTenant := tenantFromClaims(ctx); authenticatedTenant != "" {
				if !sameTenant(authenticatedTenant, identifier) {
					helper.WithContext(ctx).Warnf("cross-tenant access attempt: token tenant %q vs requested tenant %q",
						authenticatedTenant, identifier)
					return nil, errors.Forbidden("TENANT_MISMATCH", "tenant does not match the authenticated session")
				}
			}

			var tenant *biz.Tenant

			id, parseErr := uuid.Parse(identifier)
			if parseErr == nil {
				tenant, err = loader.GetByID(ctx, id)
			} else {
				tenant, err = loader.GetByCode(ctx, identifier)
			}

			if err != nil {
				helper.WithContext(ctx).Warnf("tenant not found: %s, error: %v", identifier, err)
				return nil, errors.Forbidden("TENANT_NOT_FOUND", "tenant not found")
			}

			if !tenant.IsValid() {
				if tenant.Status != "active" {
					return nil, errors.Forbidden("TENANT_DISABLED", "tenant is disabled")
				}
				if tenant.ExpiredAt != nil && time.Now().After(*tenant.ExpiredAt) {
					return nil, errors.Forbidden("TENANT_EXPIRED", "tenant subscription has expired")
				}
				return nil, errors.Forbidden("TENANT_INVALID", "tenant is invalid")
			}

			ctx = WithTenant(ctx, tenant.ToInfo())
			return handler(ctx, req)
		}
	}
}
