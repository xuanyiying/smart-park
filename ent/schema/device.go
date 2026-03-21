package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Device holds the schema definition for the Device entity.
// 设备注册表
type Device struct {
	ent.Schema
}

// Fields of the Device.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("device_id").
			MaxLen(64).
			Unique().
			NotEmpty().
			Comment("设备唯一标识"),
		field.UUID("lot_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("所属停车场ID"),
		field.UUID("lane_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("所属车道ID"),
		field.String("device_secret").
			MaxLen(128).
			Sensitive().
			Comment("HMAC密钥(加密存储)"),
		field.Enum("device_type").
			Values("camera", "gate", "display", "payment_kiosk", "sensor").
			Comment("设备类型"),
		field.String("gate_id").
			MaxLen(64).
			Optional().
			Comment("关联闸机ID"),
		field.Bool("enabled").
			Default(true).
			Comment("是否启用(维修时可禁用)"),
		field.Enum("status").
			Values("active", "offline", "disabled").
			Default("active").
			Comment("设备状态"),
		field.Time("last_heartbeat").
			Optional().
			Nillable().
			Comment("最后心跳时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Device.
func (Device) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parking_lot", ParkingLot.Type).
			Ref("devices").
			Field("lot_id").
			Unique(),
		edge.From("lane", Lane.Type).
			Ref("devices").
			Field("lane_id").
			Unique(),
	}
}

// Indexes of the Device.
func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id").Unique().StorageKey("idx_device_id"),
		index.Fields("lot_id").StorageKey("idx_device_lot"),
		index.Fields("lane_id"),
		index.Fields("status"),
	}
}
