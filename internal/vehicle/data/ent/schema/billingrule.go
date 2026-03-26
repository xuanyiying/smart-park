package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type BillingRule struct {
	ent.Schema
}

func (BillingRule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.String("rule_name").
			MaxLen(100).
			NotEmpty().
			Comment("规则名称"),
		field.Enum("rule_type").
			Values("time", "period", "monthly", "coupon", "vip").
			Comment("规则类型"),
		field.Text("conditions_json").
			Optional().
			Comment("条件配置JSON"),
		field.Text("actions_json").
			Optional().
			Comment("动作配置JSON"),
		field.JSON("rule_config", map[string]interface{}{}).
			Optional().
			Comment("规则配置JSON(兼容)"),
		field.Int("priority").
			Default(0).
			Min(0).
			Comment("优先级(数字越大优先级越高)"),
		field.Bool("is_active").
			Default(true).
			Comment("是否启用"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (BillingRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lot_id", "priority").StorageKey("idx_billing_rules_lot_priority"),
		index.Fields("lot_id", "is_active"),
	}
}
