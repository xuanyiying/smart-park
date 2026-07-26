package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Firmware struct {
	ent.Schema
}

func (Firmware) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("firmware_id").
			MaxLen(64).
			Unique().
			NotEmpty().
			Comment("固件唯一标识"),
		field.String("manufacturer").
			MaxLen(64).
			NotEmpty().
			Comment("厂商名称"),
		field.String("model").
			MaxLen(64).
			NotEmpty().
			Comment("设备型号"),
		field.String("version").
			MaxLen(32).
			NotEmpty().
			Comment("固件版本"),
		field.String("url").
			MaxLen(512).
			NotEmpty().
			Comment("固件下载地址"),
		field.Int64("size").
			Comment("固件大小(字节)"),
		field.String("md5").
			MaxLen(32).
			Comment("固件MD5校验值"),
		field.String("description").
			MaxLen(512).
			Optional().
			Comment("固件描述"),
		field.Enum("status").
			Values("draft", "published", "deprecated").
			Default("draft").
			Comment("固件状态"),
		field.Time("release_date").
			Default(time.Now).
			Comment("发布日期"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Firmware) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("firmware_id").Unique().StorageKey("idx_firmware_id"),
		index.Fields("manufacturer", "model", "version").Unique().StorageKey("idx_firmware_version"),
		index.Fields("status"),
		index.Fields("release_date"),
	}
}
