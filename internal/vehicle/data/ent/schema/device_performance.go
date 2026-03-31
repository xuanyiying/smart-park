package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// DevicePerformance holds the schema definition for the DevicePerformance entity.
type DevicePerformance struct {
	ent.Schema
}

// Fields of the DevicePerformance.
func (DevicePerformance) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("device_id").
			MaxLen(64).
			NotEmpty().
			Comment("设备ID"),
		field.Float("cpu_usage").
			Comment("CPU使用率"),
		field.Float("memory_usage").
			Comment("内存使用率"),
		field.Float("storage_usage").
			Comment("存储使用率"),
		field.Int64("network_in").
			Comment("网络入流量(字节)"),
		field.Int64("network_out").
			Comment("网络出流量(字节)"),
		field.Float("temperature").
			Comment("设备温度"),
		field.Time("timestamp").
			Default(time.Now).
			Comment("性能数据时间戳"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Indexes of the DevicePerformance.
func (DevicePerformance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id").StorageKey("idx_device_id"),
		index.Fields("timestamp").StorageKey("idx_timestamp"),
		index.Fields("device_id", "timestamp").StorageKey("idx_device_timestamp"),
	}
}
