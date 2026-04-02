package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Vehicle struct {
	ent.Schema
}

func (Vehicle) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
		field.String("plate_number").
			MaxLen(20).
			Unique().
			Comment("车牌号"),
		field.Enum("vehicle_type").
			Values("temporary", "monthly", "vip").
			Default("temporary").
			Comment("车辆类型: 临时车/月卡车/VIP"),
		field.String("owner_name").
			MaxLen(100).
			Optional().
			Comment("车主姓名"),
		field.String("owner_phone").
			MaxLen(20).
			Optional().
			Sensitive().
			Comment("车主电话(加密存储)"),
		field.Time("monthly_valid_until").
			Optional().
			Nillable().
			Comment("月卡有效期"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Vehicle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plate_number").Unique(),
		index.Fields("vehicle_type"),
		index.Fields("monthly_valid_until"),
	}
}
