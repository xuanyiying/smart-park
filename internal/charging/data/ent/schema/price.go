package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Price holds the schema definition for the Price entity.
type Price struct {
	ent.Schema
}

// Fields of the Price.
func (Price) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("station_id", uuid.UUID{}).
			Comment("充电桩ID"),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("价格名称"),
		field.Int("start_hour").
			Min(0).
			Max(23).
			Comment("开始小时(0-23)"),
		field.Int("end_hour").
			Min(0).
			Max(23).
			Comment("结束小时(0-23)"),
		field.Float("price_per_kwh").
			Default(1.0).
			Comment("电价(元/kWh)"),
		field.Float("service_fee").
			Default(0.5).
			Comment("服务费(元/kWh)"),
		field.Float("peak_load").
			Default(1.5).
			Comment("高峰负载系数"),
		field.Float("off_peak_load").
			Default(0.8).
			Comment("低谷负载系数"),
		field.Bool("is_peak_hours").
			Default(false).
			Comment("是否高峰时段"),
		field.Time("effective_at").
			Default(time.Now).
			Comment("生效时间"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("过期时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Price.
func (Price) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("station", Station.Type).
			Ref("prices").
			Field("station_id").
			Unique().
			Required(),
	}
}

// Indexes of the Price.
func (Price) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("station_id"),
		index.Fields("effective_at", "expires_at"),
		index.Fields("station_id", "is_peak_hours"),
	}
}
