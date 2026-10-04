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
		field.UUID("record_id", uuid.UUID{}).
			Comment("停车记录ID"),
		// 充电订单以充电会话ID占用该字段，且无停车场归属
		field.UUID("lot_id", uuid.UUID{}).
			Optional().
			Comment("停车场ID"),
		field.UUID("vehicle_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("车辆ID"),
		field.String("plate_number").
			MaxLen(20).
			Optional().
			Comment("车牌号(充电订单无车牌)"),
		field.Enum("order_type").
			Values("parking", "charging").
			Default("parking").
			Comment("订单类型: 停车费/充电费"),
		field.Int64("amount").
			Default(0).
			Min(0).
			Comment("原始金额(分)"),
		field.Int64("discount_amount").
			Default(0).
			Min(0).
			Comment("优惠金额(分)"),
		field.Int64("final_amount").
			Default(0).
			Min(0).
			Comment("实付金额(分)"),
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
		field.Int64("paid_amount").
			Optional().
			Default(0).
			Min(0).
			Comment("实际支付金额(分, 回调写入)"),
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
		index.Fields("status").StorageKey("idx_orders_status"),
		index.Fields("pay_time").StorageKey("idx_orders_pay_time"),
		index.Fields("transaction_id").StorageKey("idx_orders_transaction"),
		index.Fields("lot_id", "status"),
	}
}
