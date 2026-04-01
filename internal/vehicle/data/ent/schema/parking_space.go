package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type ParkingSpace struct {
	ent.Schema
}

func (ParkingSpace) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("space_id").
			MaxLen(64).
			Unique().
			NotEmpty().
			Comment("车位唯一标识"),
		field.UUID("lot_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("所属停车场ID"),
		field.String("device_id").
			MaxLen(64).
			Comment("关联设备ID"),
		field.Enum("status").
			Values("available", "occupied", "reserved", "maintenance").
			Default("available").
			Comment("车位状态"),
		field.String("vehicle_plate").
			MaxLen(20).
			Optional().
			Nillable().
			Comment("当前车辆车牌"),
		field.Time("last_update").
			Default(time.Now).
			Comment("最后更新时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (ParkingSpace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("space_id").Unique().StorageKey("idx_space_id"),
		index.Fields("lot_id").StorageKey("idx_space_lot"),
		index.Fields("device_id").StorageKey("idx_space_device"),
		index.Fields("status").StorageKey("idx_space_status"),
		index.Fields("last_update").StorageKey("idx_space_last_update"),
	}
}
