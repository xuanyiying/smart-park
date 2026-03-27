package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("station_id", uuid.UUID{}).
			Comment("充电桩ID"),
		field.UUID("connector_id", uuid.UUID{}).
			Comment("连接器ID"),
		field.UUID("user_id", uuid.UUID{}).
			Comment("用户ID"),
		field.String("vehicle_plate").
			MaxLen(20).
			NotEmpty().
			Comment("车牌号"),
		field.Time("start_time").
			Default(time.Now).
			Comment("开始时间"),
		field.Time("end_time").
			Optional().
			Nillable().
			Comment("结束时间"),
		field.Float("start_energy").
			Default(0).
			Comment("起始电量(kWh)"),
		field.Float("end_energy").
			Default(0).
			Comment("结束电量(kWh)"),
		field.Float("charged_energy").
			Default(0).
			Comment("充电电量(kWh)"),
		field.Float("cost").
			Default(0).
			Comment("电费"),
		field.Float("service_fee").
			Default(0).
			Comment("服务费"),
		field.Float("total_amount").
			Default(0).
			Comment("总金额"),
		field.Enum("status").
			Values("pending", "charging", "completed", "cancelled", "expired").
			Default("pending").
			Comment("状态"),
		field.Enum("payment_status").
			Values("pending", "paid", "refunded", "failed").
			Default("pending").
			Comment("支付状态"),
		field.Time("pay_time").
			Optional().
			Nillable().
			Comment("支付时间"),
		field.String("payment_method").
			MaxLen(50).
			Optional().
			Comment("支付方式"),
		field.String("transaction_id").
			MaxLen(100).
			Optional().
			Comment("交易ID"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("connector", Connector.Type).
			Ref("sessions").
			Field("connector_id").
			Unique().
			Required(),
	}
}

// Indexes of the Session.
func (Session) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("station_id"),
		index.Fields("connector_id"),
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("payment_status"),
		index.Fields("user_id", "created_at"),
	}
}
