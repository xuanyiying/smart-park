package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// BlacklistEntry holds plates that must be denied entry.
type BlacklistEntry struct {
	ent.Schema
}

func (BlacklistEntry) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
		field.String("plate_number").
			MaxLen(20).
			NotEmpty().
			Comment("车牌号"),
		field.String("reason").
			MaxLen(255).
			Optional().
			Comment("拉黑原因"),
		field.String("created_by").
			MaxLen(100).
			Optional().
			Comment("操作人"),
		field.Bool("active").
			Default(true).
			Comment("是否生效，移除时置 false 以保留执行历史"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (BlacklistEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("plate_number"),
	}
}
