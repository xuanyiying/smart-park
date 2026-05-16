package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type ParkingLot struct {
	ent.Schema
}

func (ParkingLot) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("name").
			MaxLen(100).
			Comment("停车场名称"),
		field.String("address").
			MaxLen(255).
			Comment("停车场地址"),
		field.Int("total_capacity").
			Default(0).
			Comment("总车位数"),
		field.Int("lanes").
			Default(1).
			Comment("车道数"),
		field.Enum("status").
			Values("active", "inactive", "maintenance").
			Default("active").
			Comment("状态"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
