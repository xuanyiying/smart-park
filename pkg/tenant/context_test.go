package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

func TestWithTenant_And_FromTenant(t *testing.T) {
	id := uuid.New()
	info := &TenantInfo{
		ID:   id,
		Code: "test-tenant",
		Name: "Test Tenant",
	}

	ctx := WithTenant(context.Background(), info)

	got, ok := FromTenant(ctx)
	assert.True(t, ok)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "test-tenant", got.Code)
	assert.Equal(t, "Test Tenant", got.Name)
}

func TestFromTenant_Missing(t *testing.T) {
	_, ok := FromTenant(context.Background())
	assert.False(t, ok)
}

func TestFromTenant_NilValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyTenant, nil)
	_, ok := FromTenant(ctx)
	assert.False(t, ok)
}

func TestFromTenant_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyTenant, "not-a-tenant")
	_, ok := FromTenant(ctx)
	assert.False(t, ok)
}

func TestMustFromTenant_Success(t *testing.T) {
	id := uuid.New()
	info := &biz.TenantInfo{ID: id, Code: "acme", Name: "Acme"}
	ctx := WithTenant(context.Background(), info)

	got := MustFromTenant(ctx)
	assert.Equal(t, id, got.ID)
}

func TestMustFromTenant_Panic(t *testing.T) {
	assert.Panics(t, func() {
		MustFromTenant(context.Background())
	})
}

func TestTenantIDFromCtx_Present(t *testing.T) {
	id := uuid.New()
	info := &TenantInfo{ID: id, Code: "acme"}
	ctx := WithTenant(context.Background(), info)

	got := TenantIDFromCtx(ctx)
	assert.Equal(t, id, got)
}

func TestTenantIDFromCtx_Missing(t *testing.T) {
	got := TenantIDFromCtx(context.Background())
	assert.Equal(t, uuid.Nil, got)
}
