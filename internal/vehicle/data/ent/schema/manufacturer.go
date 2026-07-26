package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Manufacturer struct {
	ent.Schema
}

func (Manufacturer) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("name").
			MaxLen(64).
			Unique().
			NotEmpty().
			Comment("厂商名称"),
		field.String("website").
			MaxLen(255).
			Optional().
			Comment("厂商网站"),
		field.String("contact_info").
			MaxLen(255).
			Optional().
			Comment("联系信息"),
		field.String("description").
			MaxLen(512).
			Optional().
			Comment("厂商描述"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Manufacturer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique().StorageKey("idx_manufacturer_name"),
	}
}
