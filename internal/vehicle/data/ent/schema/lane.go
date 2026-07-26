package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Lane struct {
	ent.Schema
}

func (Lane) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.Int("lane_no").
			Min(1).
			Comment("车道编号"),
		field.Enum("direction").
			Values("entry", "exit").
			Comment("车道方向: 入口/出口"),
		field.Enum("status").
			Values("active", "inactive", "maintenance").
			Default("active").
			Comment("状态"),
		field.JSON("device_config", map[string]interface{}{}).
			Optional().
			Comment("设备配置JSON"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}
