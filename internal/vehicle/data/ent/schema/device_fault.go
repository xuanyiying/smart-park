package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DeviceFault holds the schema definition for the DeviceFault entity.
type DeviceFault struct {
	ent.Schema
}

// Fields of the DeviceFault.
func (DeviceFault) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("device_id").
			MaxLen(64).
			NotEmpty().
			Comment("设备ID"),
		field.String("fault_type").
			MaxLen(64).
			NotEmpty().
			Comment("故障类型"),
		field.String("fault_code").
			MaxLen(32).
			NotEmpty().
			Comment("故障代码"),
		field.String("description").
			MaxLen(512).
			Comment("故障描述"),
		field.Enum("severity").
			Values("critical", "error", "warning", "info").
			Default("error").
			Comment("严重程度"),
		field.Enum("status").
			Values("detected", "processing", "resolved").
			Default("detected").
			Comment("故障状态"),
		field.String("suggestion").
			MaxLen(1024).
			Comment("处理建议"),
		field.Time("detected_at").
			Default(time.Now).
			Comment("检测时间"),
		field.Time("resolved_at").
			Optional().
			Comment("解决时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the DeviceFault.
func (DeviceFault) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id").StorageKey("idx_device_id"),
		index.Fields("status").StorageKey("idx_status"),
		index.Fields("severity").StorageKey("idx_severity"),
		index.Fields("detected_at").StorageKey("idx_detected_at"),
		index.Fields("device_id", "status").StorageKey("idx_device_status"),
	}
}
