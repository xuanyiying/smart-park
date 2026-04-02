package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Device struct {
	ent.Schema
}

func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
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
		field.String("manufacturer").
			MaxLen(64).
			Optional().
			Comment("设备厂商"),
		field.String("model").
			MaxLen(64).
			Optional().
			Comment("设备型号"),
		field.String("firmware_version").
			MaxLen(32).
			Optional().
			Comment("固件版本"),
		field.JSON("vendor_specific_config", map[string]interface{}{}).
			Optional().
			Comment("厂商特定配置"),
		field.String("gate_id").
			MaxLen(64).
			Optional().
			Comment("关联闸机ID"),
		field.Bool("enabled").
			Default(true).
			Comment("是否启用(维修时可禁用)"),
		field.Enum("status").
			Values("active", "offline", "disabled", "upgrading", "fault").
			Default("active").
			Comment("设备状态"),
		field.Time("last_heartbeat").
			Optional().
			Nillable().
			Comment("最后心跳时间"),
		field.Time("last_online").
			Optional().
			Nillable().
			Comment("最后在线时间"),
		field.String("fault_info").
			MaxLen(512).
			Optional().
			Comment("故障信息"),
		field.Int("heartbeat_count").
			Default(0).
			Comment("心跳次数"),
		field.Int("offline_count").
			Default(0).
			Comment("离线次数"),
		field.String("firmware_version").
			MaxLen(32).
			Optional().
			Comment("设备固件版本"),
		field.String("hardware_version").
			MaxLen(32).
			Optional().
			Comment("设备硬件版本"),
		field.JSON("device_config", map[string]interface{}{}).
			Optional().
			Comment("设备配置信息"),
		field.JSON("device_stats", map[string]interface{}{}).
			Optional().
			Comment("设备统计信息"),
		field.String("fault_code").
			MaxLen(32).
			Optional().
			Comment("故障代码"),
		field.String("fault_message").
			MaxLen(256).
			Optional().
			Comment("故障信息"),
		field.Time("last_fault_time").
			Optional().
			Nillable().
			Comment("最后故障时间"),
		field.Time("last_upgrade_time").
			Optional().
			Nillable().
			Comment("最后升级时间"),
		field.String("location").
			MaxLen(256).
			Optional().
			Comment("设备位置"),
		field.String("manufacturer").
			MaxLen(128).
			Optional().
			Comment("设备厂商"),
		field.String("model").
			MaxLen(128).
			Optional().
			Comment("设备型号"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("device_id").Unique().StorageKey("idx_device_id"),
		index.Fields("lot_id").StorageKey("idx_device_lot"),
		index.Fields("lane_id"),
		index.Fields("status"),
	}
}
