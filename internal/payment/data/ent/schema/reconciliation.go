package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Reconciliation struct {
	ent.Schema
}

func (Reconciliation) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("date").
			Comment("对账日期，格式：YYYY-MM-DD"),
		field.Enum("pay_method").
			Values("wechat", "alipay", "all").
			Default("all").
			Comment("支付方式"),
		field.Enum("status").
			Values("pending", "success", "failed", "partial").
			Default("pending").
			Comment("对账状态"),
		field.Int("total_orders").
			Default(0).
			Comment("总订单数"),
		field.Int("matched_orders").
			Default(0).
			Comment("匹配订单数"),
		field.Int("exception_orders").
			Default(0).
			Comment("异常订单数"),
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
		index.Fields("date").StorageKey("idx_reconciliation_date"),
		index.Fields("pay_method").StorageKey("idx_reconciliation_pay_method"),
		index.Fields("status").StorageKey("idx_reconciliation_status"),
	}
}

type ReconciliationException struct {
	ent.Schema
}

func (ReconciliationException) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("reconciliation_id", uuid.UUID{}).
			Comment("对账记录ID"),
		field.UUID("order_id", uuid.UUID{}).
			Comment("系统订单ID"),
		field.String("platform_order_id").
			MaxLen(64).
			Optional().
			Comment("支付平台订单ID"),
		field.Float("system_amount").
			Min(0).
			Comment("系统记录金额"),
		field.Float("platform_amount").
			Min(0).
			Comment("支付平台记录金额"),
		field.Enum("status").
			Values("unhandled", "handled", "ignored").
			Default("unhandled").
			Comment("异常处理状态"),
		field.String("reason").
			MaxLen(255).
			Comment("异常原因"),
		field.String("action").
			MaxLen(32).
			Optional().
			Comment("处理动作"),
		field.String("remark").
			MaxLen(255).
			Optional().
			Comment("处理备注"),
		field.Time("handled_at").
			Optional().
			Nillable().
			Comment("处理时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (ReconciliationException) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("reconciliation_id").StorageKey("idx_reconciliation_exception_reconciliation"),
		index.Fields("order_id").StorageKey("idx_reconciliation_exception_order"),
		index.Fields("status").StorageKey("idx_reconciliation_exception_status"),
	}
}