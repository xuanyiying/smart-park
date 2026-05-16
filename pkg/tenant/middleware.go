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
