package tenant

import (
	"context"
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

// fakeQuery stands in for a generated ent query builder. ent.Query is an alias
// for any, so the interceptor passes it through unchanged and the scope only has
// to see something it recognises.
type fakeQuery struct {
	scoped bool
}

// fakeMutation implements just the parts of ent.Mutation the hook touches. The
// embedded interface satisfies the remaining methods; they panic if called,
// which is what we want in a test.
type fakeMutation struct {
	ent.Mutation

	typ    string
	op     ent.Op
	fields map[string]ent.Value
	wheres []func(*sql.Selector)
}

func (m *fakeMutation) Op() ent.Op { return m.op }

func (m *fakeMutation) Type() string { return m.typ }

func (m *fakeMutation) Field(name string) (ent.Value, bool) {
	if m.fields == nil {
		return nil, false
	}
	v, ok := m.fields[name]
	return v, ok
}

func (m *fakeMutation) SetField(name string, value ent.Value) error {
	if m.fields == nil {
		m.fields = make(map[string]ent.Value)
	}
	m.fields[name] = value
	return nil
}

func (m *fakeMutation) WhereP(ps ...func(*sql.Selector)) {
	m.wheres = append(m.wheres, ps...)
}

func TestTenantQueryInterceptorSkipsWithoutTenant(t *testing.T) {
	var scoped int
	interceptor := TenantQueryInterceptor(func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
		scoped++
	})

	next := ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
		return "rows", nil
	})

	got, err := interceptor.Intercept(next).Query(context.Background(), &fakeQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "rows" {
		t.Fatalf("expected the query to pass through, got %v", got)
	}
	if scoped != 0 {
		t.Fatalf("expected no tenant scoping without a tenant in context, got %d", scoped)
	}
}

func TestTenantQueryInterceptorScopesWithTenant(t *testing.T) {
	tenantID := uuid.New()
	var seen uuid.UUID
	var scoped int

	interceptor := TenantQueryInterceptor(func(ctx context.Context, q ent.Query, id uuid.UUID) {
		scoped++
		seen = id
		if fq, ok := q.(*fakeQuery); ok {
			fq.scoped = true
		}
	})

	ctx := WithTenant(context.Background(), &TenantInfo{ID: tenantID, Code: "acme", Name: "Acme"})
	q := &fakeQuery{}

	next := ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
		return q, nil
	})

	got, err := interceptor.Intercept(next).Query(ctx, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scoped != 1 {
		t.Fatalf("expected the scope to run once, got %d", scoped)
	}
	if seen != tenantID {
		t.Fatalf("expected scope to receive %s, got %s", tenantID, seen)
	}
	if fq, ok := got.(*fakeQuery); !ok || !fq.scoped {
		t.Fatalf("expected the scope to receive the underlying query builder, got %#v", got)
	}
}

func TestTenantMutationHookFillsTenantOnCreate(t *testing.T) {
	tenantID := uuid.New()
	hook := TenantMutationHook("ParkingRecord")

	m := &fakeMutation{typ: "ParkingRecord", op: ent.OpCreate}

	next := ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		return "created", nil
	})

	ctx := WithTenant(context.Background(), &TenantInfo{ID: tenantID})
	if _, err := hook(next).Mutate(ctx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := m.Field(TenantIDField)
	if !ok {
		t.Fatal("expected tenant_id to be filled on create")
	}
	if got != tenantID {
		t.Fatalf("expected tenant_id %s, got %v", tenantID, got)
	}
}

func TestTenantMutationHookKeepsExplicitTenant(t *testing.T) {
	tenantID := uuid.New()
	explicit := uuid.New()
	hook := TenantMutationHook("ParkingRecord")

	// A caller that set tenant_id on purpose (cross-tenant provisioning) must not
	// be silently rewritten with the request's tenant.
	m := &fakeMutation{
		typ:    "ParkingRecord",
		op:     ent.OpCreate,
		fields: map[string]ent.Value{TenantIDField: explicit},
	}

	next := ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		return "created", nil
	})

	ctx := WithTenant(context.Background(), &TenantInfo{ID: tenantID})
	if _, err := hook(next).Mutate(ctx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := m.Field(TenantIDField)
	if got != explicit {
		t.Fatalf("expected the explicit tenant %s to be kept, got %v", explicit, got)
	}
}

func TestTenantMutationHookRestrictsUpdates(t *testing.T) {
	tenantID := uuid.New()
	hook := TenantMutationHook("ParkingRecord")

	m := &fakeMutation{typ: "ParkingRecord", op: ent.OpUpdateOne}

	next := ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		return "updated", nil
	})

	ctx := WithTenant(context.Background(), &TenantInfo{ID: tenantID})
	if _, err := hook(next).Mutate(ctx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.wheres) != 1 {
		t.Fatalf("expected one tenant predicate on update, got %d", len(m.wheres))
	}
}

func TestTenantMutationHookIgnoresGlobalTypes(t *testing.T) {
	tenantID := uuid.New()
	// Manufacturer is reference data shared across tenants; scoping it would make
	// it unwritable from any tenant scoped request.
	hook := TenantMutationHook("ParkingRecord")

	m := &fakeMutation{typ: "Manufacturer", op: ent.OpCreate}

	next := ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
		return "created", nil
	})

	ctx := WithTenant(context.Background(), &TenantInfo{ID: tenantID})
	if _, err := hook(next).Mutate(ctx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := m.Field(TenantIDField); ok {
		t.Fatal("did not expect tenant_id to be filled for a global entity type")
	}
}
