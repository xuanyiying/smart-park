package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type HourlyStats struct {
	ent.Schema
}

func (HourlyStats) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.Time("stat_hour").
			Comment("统计小时"),
		field.Int("entries").
			Default(0).
			Comment("入场数"),
		field.Int("exits").
			Default(0).
			Comment("出场数"),
		field.Int("occupied_spaces").
			Default(0).
			Comment("占用车位数"),
		field.Float("occupancy_rate").
			Default(0).
			Comment("占用率"),
		field.Int64("revenue").
			Default(0).
			Comment("收入(分)"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (HourlyStats) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lot_id", "stat_hour").Unique().StorageKey("idx_hourly_stats_lot_hour"),
	}
}
