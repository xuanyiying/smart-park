package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ParkingRecord holds the schema definition for the ParkingRecord entity.
// 停车记录表
type ParkingRecord struct {
	ent.Schema
}

// Fields of the ParkingRecord.
func (ParkingRecord) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		// 停车场关联
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		// 入场信息
		field.UUID("entry_lane_id", uuid.UUID{}).
			Comment("入场车道ID"),
		field.UUID("vehicle_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("车辆ID(无牌车为空)"),
		field.String("plate_number").
			MaxLen(20).
			Optional().
			Nillable().
			Comment("车牌号(无牌车为空)"),
		field.Enum("plate_number_source").
			Values("camera", "manual", "offline").
			Optional().
			Comment("车牌来源"),
		field.Time("entry_time").
			Comment("入场时间"),
		field.String("entry_image_url").
			MaxLen(255).
			Optional().
			Comment("入场图片URL"),
		// 记录状态
		field.Enum("record_status").
			Values("entry", "exiting", "exited", "paid").
			Default("entry").
			Comment("记录状态: 入场中/出场中/已出场/已支付"),
		// 出场信息
		field.Time("exit_time").
			Optional().
			Nillable().
			Comment("出场时间"),
		field.String("exit_image_url").
			MaxLen(255).
			Optional().
			Comment("出场图片URL"),
		field.UUID("exit_lane_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("出场车道ID"),
		field.String("exit_device_id").
			MaxLen(64).
			Optional().
			Comment("出场设备ID(用于支付后自动开闸)"),
		// 停车时长
		field.Int("parking_duration").
			Optional().
			Default(0).
			Comment("停车时长(秒)"),
		// 支付状态
		field.Enum("exit_status").
			Values("unpaid", "paid", "refunded", "waived").
			Default("unpaid").
			Comment("出场支付状态"),
		// 并发控制
		field.Int("payment_lock").
			Default(0).
			Comment("乐观锁版本号"),
		// 扩展元数据
		field.JSON("record_metadata", map[string]interface{}{}).
			Optional().
			Comment("扩展元数据(如月卡过期标记等)"),
		// 时间戳
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the ParkingRecord.
func (ParkingRecord) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parking_lot", ParkingLot.Type).
			Ref("records").
			Field("lot_id").
			Unique().
			Required(),
		edge.To("order", Order.Type).Unique(),
	}
}

// Indexes of the ParkingRecord.
func (ParkingRecord) Indexes() []ent.Index {
	return []ent.Index{
		// 按车牌查询入场记录(出场时使用)
		index.Fields("plate_number", "entry_time").StorageKey("idx_parking_records_plate_entry"),
		// 按停车场和状态查询(查找未出场记录)
		index.Fields("lot_id", "record_status").StorageKey("idx_parking_records_lot_status"),
		// 按出场时间查询(报表统计)
		index.Fields("exit_time").StorageKey("idx_parking_records_exit"),
	}
}
