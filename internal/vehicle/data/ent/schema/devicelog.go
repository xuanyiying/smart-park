package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DeviceLog 设备日志记录
type DeviceLog struct {
	ent.Schema
}

func (DeviceLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("device_id").
			MaxLen(64).
			NotEmpty().
			Comment("设备唯一标识"),
		field.Enum("log_type").
			Values("info", "warning", "error", "debug").
			Default("info").
			Comment("日志类型"),
		field.String("log_level").
			MaxLen(10).
			Default("info").
			Comment("日志级别"),
		field.String("message").
			MaxLen(1024).
			NotEmpty().
			Comment("日志消息"),
		field.String("fault_code").
			MaxLen(32).
			Optional().
			Comment("故障代码"),
		field.JSON("details", map[string]interface{}{}).
			Optional().
			Comment("日志详细信息"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (DeviceLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id").StorageKey("idx_device_log_device"),
		index.Fields("log_type").StorageKey("idx_device_log_type"),
		index.Fields("created_at").StorageKey("idx_device_log_time"),
	}
}
