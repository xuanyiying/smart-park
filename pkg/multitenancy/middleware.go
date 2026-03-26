package multitenancy

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// TenantResolver resolves tenant from incoming requests
type TenantResolver interface {
	Resolve(ctx context.Context, req interface{}) (*TenantInfo, error)
}

// HeaderTenantResolver resolves tenant from HTTP header
type HeaderTenantResolver struct {
	HeaderName string
	Repo       TenantRepo
}

// Resolve implements TenantResolver
func (r *HeaderTenantResolver) Resolve(ctx context.Context, req interface{}) (*TenantInfo, error) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		header := tr.RequestHeader()
		tenantCode := header.Get(r.HeaderName)
		if tenantCode == "" {
			return nil, ErrTenantNotFound
		}
		return r.Repo.GetTenantByCode(ctx, tenantCode)
	}
	return nil, ErrTenantNotFound
}

// DomainTenantResolver resolves tenant from request domain
type DomainTenantResolver struct {
	Repo TenantRepo
}

// Resolve implements TenantResolver
func (r *DomainTenantResolver) Resolve(ctx context.Context, req interface{}) (*TenantInfo, error) {
	if tr, ok := transport.FromServerContext(ctx); ok {
		// Try to get host from HTTP transport
		if ht, ok := tr.(*http.Transport); ok {
			host := ht.Request().Host
			// Extract tenant from subdomain
			parts := strings.Split(host, ".")
			if len(parts) > 2 {
				tenantCode := parts[0]
				return r.Repo.GetTenantByCode(ctx, tenantCode)
			}
		}
	}
	return nil, ErrTenantNotFound
}

// Middleware creates a middleware that extracts tenant info and adds it to context
func Middleware(resolver TenantResolver) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tenant, err := resolver.Resolve(ctx, req)
			if err != nil {
				// Allow requests without tenant for public endpoints
				if IsPublicEndpoint(ctx) {
					return handler(ctx, req)
				}
				return nil, err
			}

			if tenant == nil {
				return nil, ErrTenantInvalid
			}

			ctx = ContextWithTenant(ctx, tenant)
			return handler(ctx, req)
		}
	}
}

// IsPublicEndpoint checks if the current endpoint is public (no tenant required)
func IsPublicEndpoint(ctx context.Context) bool {
	// Check for public path patterns
	if tr, ok := transport.FromServerContext(ctx); ok {
		path := tr.Operation()
		publicPaths := []string{
			"/api/v1/public/",
			"/api/v1/auth/",
			"/api/v1/health",
		}
		for _, p := range publicPaths {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
	}
	return false
}

// RequireTenant middleware ensures tenant is present in context
func RequireTenant() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			_, ok := TenantFromContext(ctx)
			if !ok {
				return nil, ErrTenantNotFound
			}
			return handler(ctx, req)
		}
	}
}

// TenantRepo interface for tenant data access
type TenantRepo interface {
	GetTenantByCode(ctx context.Context, code string) (*TenantInfo, error)
	GetTenantByID(ctx context.Context, id string) (*TenantInfo, error)
}
