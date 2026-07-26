package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Station holds the schema definition for the Station entity.
type Station struct {
	ent.Schema
}

// Fields of the Station.
func (Station) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("充电桩名称"),
		field.Enum("station_type").
			Values("ac", "dc", "fast_dc").
			Default("ac").
			Comment("充电桩类型: ac-交流, dc-直流, fast_dc-快充"),
		field.Enum("status").
			Values("available", "offline", "maintenance").
			Default("available").
			Comment("状态: available-可用, offline-离线, maintenance-维护中"),
		field.Enum("connector_type").
			Values("ac", "dc", "fast_dc").
			Default("ac").
			Comment("连接器类型"),
		field.Float("max_power").
			Default(7.0).
			Comment("最大功率(kW)"),
		field.Float("voltage").
			Default(220.0).
			Comment("电压(V)"),
		field.Int("total_connectors").
			Default(1).
			Min(1).
			Comment("总连接器数"),
		field.Int("available_connectors").
			Default(1).
			Min(0).
			Comment("可用连接器数"),
		field.String("location").
			MaxLen(200).
			Optional().
			Comment("位置描述"),
		field.String("floor").
			MaxLen(50).
			Optional().
			Comment("楼层"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Station.
func (Station) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("connectors", Connector.Type),
		edge.To("prices", Price.Type),
	}
}

// Indexes of the Station.
func (Station) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("lot_id"),
		index.Fields("status"),
		index.Fields("lot_id", "status"),
	}
}
