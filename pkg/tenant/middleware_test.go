package tenant

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"time"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

type mockHeader struct {
	values map[string]string
}

func newMockHeader() *mockHeader {
	return &mockHeader{values: make(map[string]string)}
}

func (h *mockHeader) Get(key string) string      { return h.values[key] }
func (h *mockHeader) Set(key, value string)       { h.values[key] = value }
func (h *mockHeader) Add(key, value string)       { h.values[key] = value }
func (h *mockHeader) Keys() []string              { return nil }
func (h *mockHeader) Values(key string) []string  { return nil }

type mockTransporter struct {
	kind      transport.Kind
	operation string
	header    *mockHeader
}

func newMockTransporter() *mockTransporter {
	return &mockTransporter{
		kind:   transport.KindHTTP,
		header: newMockHeader(),
	}
}

func (t *mockTransporter) Kind() transport.Kind      { return t.kind }
func (t *mockTransporter) Endpoint() string           { return "http://localhost:8000" }
func (t *mockTransporter) Operation() string          { return t.operation }
func (t *mockTransporter) RequestHeader() transport.Header { return t.header }
func (t *mockTransporter) ReplyHeader() transport.Header   { return t.header }

type mockTenantLoader struct {
	tenants map[uuid.UUID]*biz.Tenant
	codes   map[string]*biz.Tenant
}

func newMockTenantLoader() *mockTenantLoader {
	return &mockTenantLoader{
		tenants: make(map[uuid.UUID]*biz.Tenant),
		codes:   make(map[string]*biz.Tenant),
	}
}

func (m *mockTenantLoader) addTenant(t *biz.Tenant) {
	m.tenants[t.ID] = t
	m.codes[t.Code] = t
}

func (m *mockTenantLoader) GetByID(ctx context.Context, id uuid.UUID) (*biz.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, biz.ErrTenantNotFound
	}
	return t, nil
}

func (m *mockTenantLoader) GetByCode(ctx context.Context, code string) (*biz.Tenant, error) {
	t, ok := m.codes[code]
	if !ok {
		return nil, biz.ErrTenantNotFound
	}
	return t, nil
}

func TestHeaderExtractor_Extract(t *testing.T) {
	extractor := NewHeaderExtractor("X-Tenant-ID")
	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", "acme-corp")
	ctx := transport.NewServerContext(context.Background(), tr)

	val, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "acme-corp", val)
}

func TestHeaderExtractor_Extract_Missing(t *testing.T) {
	extractor := NewHeaderExtractor("X-Tenant-ID")
	tr := newMockTransporter()
	ctx := transport.NewServerContext(context.Background(), tr)

	val, err := extractor.Extract(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestHeaderExtractor_DefaultHeader(t *testing.T) {
	extractor := NewHeaderExtractor("")
	assert.Equal(t, "X-Tenant-ID", extractor.HeaderName)
}

func TestHeaderExtractor_NoTransport(t *testing.T) {
	extractor := NewHeaderExtractor("X-Tenant-ID")
	val, err := extractor.Extract(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestTenantMiddleware_Success(t *testing.T) {
	loader := newMockTenantLoader()
	tenantID := uuid.New()
	loader.addTenant(&biz.Tenant{
		ID:     tenantID,
		Name:   "Acme Corp",
		Code:   "acme",
		Status: "active",
		Config: biz.DefaultTenantConfig(),
	})

	extractor := NewHeaderExtractor("X-Tenant-ID")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", tenantID.String())
	ctx := transport.NewServerContext(context.Background(), tr)

	var capturedCtx context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedCtx = ctx
		return "ok", nil
	}

	wrapped := mw(handler)
	reply, err := wrapped(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, "ok", reply)

	info, ok := FromTenant(capturedCtx)
	assert.True(t, ok)
	assert.Equal(t, tenantID, info.ID)
	assert.Equal(t, "acme", info.Code)
}

func TestTenantMiddleware_ByCode(t *testing.T) {
	loader := newMockTenantLoader()
	tenantID := uuid.New()
	loader.addTenant(&biz.Tenant{
		ID:     tenantID,
		Name:   "Acme Corp",
		Code:   "acme",
		Status: "active",
		Config: biz.DefaultTenantConfig(),
	})

	extractor := NewHeaderExtractor("X-Tenant-Code")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	tr.header.Set("X-Tenant-Code", "acme")
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	wrapped := mw(handler)
	reply, err := wrapped(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, "ok", reply)
}

func TestTenantMiddleware_MissingIdentifier(t *testing.T) {
	loader := newMockTenantLoader()
	extractor := NewHeaderExtractor("X-Tenant-ID")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	wrapped := mw(handler)
	_, err := wrapped(ctx, nil)
	assert.Error(t, err)
}

func TestTenantMiddleware_TenantNotFound(t *testing.T) {
	loader := newMockTenantLoader()
	extractor := NewHeaderExtractor("X-Tenant-ID")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", uuid.New().String())
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	wrapped := mw(handler)
	_, err := wrapped(ctx, nil)
	assert.Error(t, err)
}

func TestTenantMiddleware_DisabledTenant(t *testing.T) {
	loader := newMockTenantLoader()
	tenantID := uuid.New()
	loader.addTenant(&biz.Tenant{
		ID:     tenantID,
		Name:   "Disabled Corp",
		Code:   "disabled",
		Status: "disabled",
		Config: biz.DefaultTenantConfig(),
	})

	extractor := NewHeaderExtractor("X-Tenant-ID")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", tenantID.String())
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	wrapped := mw(handler)
	_, err := wrapped(ctx, nil)
	assert.Error(t, err)
}

func TestTenantMiddleware_ExpiredTenant(t *testing.T) {
	loader := newMockTenantLoader()
	tenantID := uuid.New()
	expiredAt := time.Now().Add(-24 * time.Hour)
	loader.addTenant(&biz.Tenant{
		ID:        tenantID,
		Name:      "Expired Corp",
		Code:      "expired",
		Status:    "active",
		Config:    biz.DefaultTenantConfig(),
		ExpiredAt: &expiredAt,
	})

	extractor := NewHeaderExtractor("X-Tenant-ID")
	mw := TenantMiddleware(loader, extractor, log.DefaultLogger)

	tr := newMockTransporter()
	tr.header.Set("X-Tenant-ID", tenantID.String())
	ctx := transport.NewServerContext(context.Background(), tr)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	wrapped := mw(handler)
	_, err := wrapped(ctx, nil)
	assert.Error(t, err)
}
