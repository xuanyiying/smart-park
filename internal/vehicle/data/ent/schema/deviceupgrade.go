package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DeviceUpgrade 设备升级记录
type DeviceUpgrade struct {
	ent.Schema
}

func (DeviceUpgrade) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("device_id").
			MaxLen(64).
			NotEmpty().
			Comment("设备唯一标识"),
		field.String("from_version").
			MaxLen(32).
			NotEmpty().
			Comment("升级前版本"),
		field.String("to_version").
			MaxLen(32).
			NotEmpty().
			Comment("升级后版本"),
		field.String("firmware_url").
			MaxLen(512).
			Optional().
			Comment("固件下载地址"),
		field.Enum("status").
			Values("pending", "in_progress", "success", "failed").
			Default("pending").
			Comment("升级状态"),
		field.String("error_message").
			MaxLen(512).
			Optional().
			Comment("错误信息"),
		field.Int64("duration").
			Optional().
			Comment("升级耗时(秒)"),
		field.Time("start_time").
			Default(time.Now).
			Comment("开始升级时间"),
		field.Time("end_time").
			Optional().
			Nillable().
			Comment("结束升级时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (DeviceUpgrade) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id").StorageKey("idx_device_upgrade_device"),
		index.Fields("status").StorageKey("idx_device_upgrade_status"),
		index.Fields("start_time").StorageKey("idx_device_upgrade_time"),
	}
}
