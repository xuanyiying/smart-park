package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Order struct {
	ent.Schema
}

func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("tenant_id", uuid.UUID{}).
			Comment("租户ID"),
		field.UUID("record_id", uuid.UUID{}).
			Comment("停车记录ID"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.UUID("vehicle_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("车辆ID"),
		field.String("plate_number").
			MaxLen(20).
			NotEmpty().
			Comment("车牌号"),
		field.Float("amount").
			Min(0).
			Comment("原始金额"),
		field.Float("discount_amount").
			Default(0).
			Min(0).
			Comment("优惠金额"),
		field.Float("final_amount").
			Min(0).
			Comment("实付金额"),
		field.Enum("status").
			Values("pending", "paid", "refunding", "refunded", "failed").
			Default("pending").
			Comment("订单状态"),
		field.Time("pay_time").
			Optional().
			Nillable().
			Comment("支付时间"),
		field.Enum("pay_method").
			Values("wechat", "alipay", "cash").
			Optional().
			Comment("支付方式"),
		field.String("transaction_id").
			MaxLen(64).
			Optional().
			Comment("支付渠道交易号"),
		field.Float("paid_amount").
			Optional().
			Min(0).
			Comment("实际支付金额(回调写入)"),
		field.Time("refunded_at").
			Optional().
			Nillable().
			Comment("退款时间"),
		field.String("refund_transaction_id").
			MaxLen(64).
			Optional().
			Comment("退款渠道流水号"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("status").StorageKey("idx_orders_status"),
		index.Fields("pay_time").StorageKey("idx_orders_pay_time"),
		index.Fields("transaction_id").StorageKey("idx_orders_transaction"),
		index.Fields("lot_id", "status"),
	}
}
