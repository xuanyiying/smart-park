package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RefundApproval holds the schema definition for the RefundApproval entity.
// 退款审批记录表(管理端异常退款用)
type RefundApproval struct {
	ent.Schema
}

// Fields of the RefundApproval.
func (RefundApproval) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("order_id", uuid.UUID{}).
			Comment("订单ID"),
		field.String("applicant").
			MaxLen(64).
			NotEmpty().
			Comment("申请人"),
		field.String("approver").
			MaxLen(64).
			Optional().
			Comment("审批人"),
		field.Float64("amount").
			Min(0).
			Comment("退款金额"),
		field.Text("reason").
			NotEmpty().
			Comment("退款原因"),
		field.Enum("refund_method").
			Values("original", "manual").
			Default("original").
			Comment("退款方式: 原路返回/人工"),
		field.Enum("status").
			Values("pending", "approved", "rejected").
			Default("pending").
			Comment("审批状态"),
		field.Time("approved_at").
			Optional().
			Nillable().
			Comment("审批时间"),
		field.Text("reject_reason").
			Optional().
			Comment("拒绝原因"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the RefundApproval.
func (RefundApproval) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("order", Order.Type).
			Ref("refund_approvals").
			Field("order_id").
			Unique().
			Required(),
	}
}

// Indexes of the RefundApproval.
func (RefundApproval) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id").StorageKey("idx_refund_order"),
		index.Fields("status").StorageKey("idx_refund_status"),
		index.Fields("applicant"),
	}
}
