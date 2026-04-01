package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("open_id").
			MaxLen(100).
			Unique().
			Comment("微信/支付宝 OpenID"),
		field.String("nickname").
			MaxLen(100).
			Optional().
			Comment("用户昵称"),
		field.String("avatar").
			MaxLen(500).
			Optional().
			Comment("用户头像URL"),
		field.String("phone").
			MaxLen(20).
			Optional().
			Sensitive().
			Comment("手机号(加密存储)"),
		field.Int("credit_score").
			Default(100).
			Comment("信用评分"),
		field.String("credit_level").
			Default("A").
			Comment("信用等级"),
		field.String("default_pay_method").
			Optional().
			Comment("默认支付方式"),
		field.String("payment_token").
			Optional().
			Sensitive().
			Comment("支付令牌"),
		field.Bool("auto_pay_enabled").
			Default(false).
			Comment("是否启用自动扣费"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("open_id").Unique(),
	}
}
