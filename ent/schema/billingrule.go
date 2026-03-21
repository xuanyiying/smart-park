package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// BillingRule holds the schema definition for the BillingRule entity.
// 计费规则表
type BillingRule struct {
	ent.Schema
}

// Fields of the BillingRule.
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
		// 规则配置(JSON格式存储条件和动作)
		field.Text("conditions_json").
			Optional().
			Comment("条件配置JSON"),
		field.Text("actions_json").
			Optional().
			Comment("动作配置JSON"),
		// 兼容旧字段
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

// Edges of the BillingRule.
func (BillingRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parking_lot", ParkingLot.Type).
			Ref("billing_rules").
			Field("lot_id").
			Unique().
			Required(),
	}
}

// Indexes of the BillingRule.
func (BillingRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lot_id", "priority").StorageKey("idx_billing_rules_lot_priority"),
		index.Fields("lot_id", "is_active"),
	}
}
