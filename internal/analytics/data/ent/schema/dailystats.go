package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type DailyStats struct {
	ent.Schema
}

func (DailyStats) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.Time("stat_date").
			Comment("统计日期"),
		field.Int("total_entries").
			Default(0).
			Comment("总入场数"),
		field.Int("total_exits").
			Default(0).
			Comment("总出场数"),
		field.Int("total_vehicles").
			Default(0).
			Comment("在场车辆数"),
		field.Int64("total_amount").
			Default(0).
			Comment("总收入(分)"),
		field.Int64("total_discount").
			Default(0).
			Comment("总优惠(分)"),
		field.Int64("net_amount").
			Default(0).
			Comment("净收入(分)"),
		field.Float("avg_duration").
			Default(0).
			Comment("平均停车时长(小时)"),
		field.Int("peak_hour").
			Default(0).
			Comment("高峰小时"),
		field.Int("peak_vehicles").
			Default(0).
			Comment("高峰车辆数"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (DailyStats) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lot_id", "stat_date").Unique().StorageKey("idx_daily_stats_lot_date"),
	}
}
