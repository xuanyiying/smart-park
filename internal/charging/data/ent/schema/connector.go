package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Connector holds the schema definition for the Connector entity.
type Connector struct {
	ent.Schema
}

// Fields of the Connector.
func (Connector) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("station_id", uuid.UUID{}).
			Comment("充电桩ID"),
		field.Int("number").
			Min(1).
			Comment("连接器编号"),
		field.Enum("type").
			Values("ac", "dc", "fast_dc").
			Default("ac").
			Comment("类型"),
		field.Enum("status").
			Values("available", "charging", "faulted", "offline").
			Default("available").
			Comment("状态: available-可用, charging-充电中, faulted-故障, offline-离线"),
		field.Float("max_power").
			Default(7.0).
			Comment("最大功率(kW)"),
		field.Float("voltage").
			Default(220.0).
			Comment("电压(V)"),
		field.Float("current").
			Default(0).
			Comment("电流(A)"),
		field.String("fault_code").
			MaxLen(100).
			Optional().
			Comment("故障代码"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Connector.
func (Connector) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("station", Station.Type).
			Ref("connectors").
			Field("station_id").
			Unique().
			Required(),
		edge.To("sessions", Session.Type),
	}
}

// Indexes of the Connector.
func (Connector) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("station_id"),
		index.Fields("status"),
		index.Fields("station_id", "number").Unique(),
	}
}
