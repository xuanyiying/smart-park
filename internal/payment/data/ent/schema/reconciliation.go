package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Reconciliation 对账记录
type Reconciliation struct {
	ent.Schema
}

func (Reconciliation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("order_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("订单ID"),
		field.String("payment_method").
			MaxLen(20).
			Optional().
			Comment("支付方式"),
		field.Float("order_amount").
			Min(0).
			Comment("订单金额"),
		field.Float("paid_amount").
			Min(0).
			Comment("实付金额"),
		field.String("transaction_id").
			MaxLen(64).
			Optional().
			Comment("支付渠道交易号"),
		field.Time("reconciliation_time").
			Default(time.Now).
			Comment("对账时间"),
		field.Enum("status").
			Values("pending", "matched", "mismatch", "missing").
			Default("pending").
			Comment("对账状态"),
		field.String("notes").
			MaxLen(500).
			Optional().
			Comment("对账备注"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Reconciliation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id").StorageKey("idx_reconciliation_order"),
		index.Fields("status").StorageKey("idx_reconciliation_status"),
		index.Fields("reconciliation_time").StorageKey("idx_reconciliation_time"),
		index.Fields("transaction_id").StorageKey("idx_reconciliation_transaction"),
	}
}
