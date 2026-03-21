package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Lane holds the schema definition for the Lane entity.
// 车道表
type Lane struct {
	ent.Schema
}

// Fields of the Lane.
func (Lane) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("所属停车场ID"),
		field.Int("lane_no").
			Min(1).
			Comment("车道编号"),
		field.Enum("direction").
			Values("entry", "exit").
			Comment("车道方向: 入口/出口"),
		field.Enum("status").
			Values("active", "inactive", "maintenance").
			Default("active").
			Comment("状态"),
		field.JSON("device_config", map[string]interface{}{}).
			Optional().
			Comment("设备配置JSON"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Lane.
func (Lane) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parking_lot", ParkingLot.Type).
			Ref("lanes").
			Field("lot_id").
			Unique().
			Required(),
		edge.To("devices", Device.Type),
		edge.To("entry_records", ParkingRecord.Type),
		edge.To("exit_records", ParkingRecord.Type),
	}
}

// Indexes of the Lane.
func (Lane) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lot_id", "lane_no").Unique(),
	}
}
