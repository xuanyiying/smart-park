package multitenancy

import (
	"context"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

const TenantIDField = "tenant_id"

type TenantHookConfig struct {
	TenantIDField string
	GetTenantID   func(context.Context) *uuid.UUID
}

func DefaultTenantHookConfig() *TenantHookConfig {
	return &TenantHookConfig{
		TenantIDField: TenantIDField,
		GetTenantID:   GetTenantID,
	}
}

type TenantHook struct {
	config *TenantHookConfig
}

func NewTenantHook(config *TenantHookConfig) *TenantHook {
	if config == nil {
		config = DefaultTenantHookConfig()
	}
	return &TenantHook{config: config}
}

func (h *TenantHook) OnCreate() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			tenantID := h.config.GetTenantID(ctx)
			if tenantID == nil {
				return next.Mutate(ctx, m)
			}

			if err := m.SetField(h.config.TenantIDField, *tenantID); err == nil {
				return next.Mutate(ctx, m)
			}

			return next.Mutate(ctx, m)
		})
	}
}

func (h *TenantHook) OnQuery() ent.Hook {
	return func(next ent.Querier) ent.Querier {
		return ent.QueryFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			tenantID := h.config.GetTenantID(ctx)
			if tenantID == nil {
				return next.Query(ctx, q)
			}

			if sq, ok := q.(interface{ WhereP(...interface{}) }); ok {
				sq.WhereP(sql.EQ(h.config.TenantIDField, *tenantID))
			}

			return next.Query(ctx, q)
		})
	}
}

func (h *TenantHook) OnUpdate() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			tenantID := h.config.GetTenantID(ctx)
			if tenantID == nil {
				return next.Mutate(ctx, m)
			}

			if wm, ok := m.(interface{ WhereP(...interface{}) }); ok {
				wm.WhereP(sql.EQ(h.config.TenantIDField, *tenantID))
			}

			return next.Mutate(ctx, m)
		})
	}
}

func (h *TenantHook) OnDelete() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			tenantID := h.config.GetTenantID(ctx)
			if tenantID == nil {
				return next.Mutate(ctx, m)
			}

			if wm, ok := m.(interface{ WhereP(...interface{}) }); ok {
				wm.WhereP(sql.EQ(h.config.TenantIDField, *tenantID))
			}

			return next.Mutate(ctx, m)
		})
	}
}

func (h *TenantHook) Hooks() []ent.Hook {
	return []ent.Hook{
		ent.HookFunc(func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				switch m.Op() {
				case ent.OpCreate:
					return h.OnCreate()(next).Mutate(ctx, m)
				case ent.OpUpdateOne, ent.OpUpdate:
					return h.OnUpdate()(next).Mutate(ctx, m)
				case ent.OpDeleteOne, ent.OpDelete:
					return h.OnDelete()(next).Mutate(ctx, m)
				default:
					return next.Mutate(ctx, m)
				}
			})
		}),
		h.OnQuery(),
	}
}

func RegisterTenantHooks(client *ent.Client, config *TenantHookConfig) {
	hook := NewTenantHook(config)
	client.Use(hook.Hooks()...)
}

type TenantSchemaMixin struct {
	ent.Schema
}

func (TenantSchemaMixin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID(TenantIDField, uuid.UUID{}).
			Comment("租户ID"),
	}
}

func (TenantSchemaMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields(TenantIDField),
	}
}
