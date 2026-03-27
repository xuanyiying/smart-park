package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Tenant holds the schema definition for the Tenant entity.
type Tenant struct {
	ent.Schema
}

// Fields of the Tenant.
func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("name").
			MaxLen(100).
			NotEmpty().
			Comment("租户名称"),
		field.String("code").
			MaxLen(50).
			Unique().
			NotEmpty().
			Comment("租户代码(唯一)"),
		field.Enum("status").
			Values("active", "disabled", "expired").
			Default("active").
			Comment("状态: active-活跃, disabled-禁用, expired-过期"),
		field.Int("max_parking_lots").
			Default(1).
			Comment("最大停车场数量"),
		field.Int("max_devices").
			Default(10).
			Comment("最大设备数量"),
		field.Int("max_users").
			Default(50).
			Comment("最大用户数量"),
		field.Int64("storage_quota").
			Default(104857600). // 100MB
			Comment("存储配额(字节)"),
		field.JSON("features", []string{}).
			Default([]string{"basic"}).
			Comment("功能特性列表"),
		field.String("custom_domain").
			MaxLen(200).
			Optional().
			Comment("自定义域名"),
		field.String("timezone").
			MaxLen(50).
			Default("Asia/Shanghai").
			Comment("时区"),
		field.String("currency").
			MaxLen(10).
			Default("CNY").
			Comment("货币"),
		field.String("language").
			MaxLen(10).
			Default("zh-CN").
			Comment("语言"),
		field.Time("expired_at").
			Optional().
			Nillable().
			Comment("过期时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Tenant.
func (Tenant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("status"),
	}
}
