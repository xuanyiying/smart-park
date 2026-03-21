package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"github.com/google/uuid"
)

// ParkingLot holds the schema definition for the ParkingLot entity.
// 停车场表
type ParkingLot struct {
	ent.Schema
}

// Fields of the ParkingLot.
func (ParkingLot) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("停车场名称"),
		field.String("address").
			MaxLen(255).
			Optional().
			Comment("地址"),
		field.Int("lanes").
			Default(1).
			Min(1).
			Comment("车道数量"),
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

// Edges of the ParkingLot.
func (ParkingLot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("lanes", Lane.Type),
		edge.To("billing_rules", BillingRule.Type),
		edge.To("devices", Device.Type),
	}
}
