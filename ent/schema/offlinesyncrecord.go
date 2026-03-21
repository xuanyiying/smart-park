package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// OfflineSyncRecord holds the schema definition for the OfflineSyncRecord entity.
// 离线放行记录表(网络恢复后同步)
type OfflineSyncRecord struct {
	ent.Schema
}

// Fields of the OfflineSyncRecord.
func (OfflineSyncRecord) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.String("offline_id").
			MaxLen(64).
			Unique().
			NotEmpty().
			Comment("设备本地流水号"),
		field.UUID("record_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("停车记录ID"),
		field.UUID("lot_id", uuid.UUID{}).
			Comment("停车场ID"),
		field.String("device_id").
			MaxLen(64).
			NotEmpty().
			Comment("设备ID"),
		field.String("gate_id").
			MaxLen(64).
			NotEmpty().
			Comment("闸机ID"),
		field.Time("open_time").
			Comment("开闸时间"),
		field.Float64("sync_amount").
			Optional().
			Min(0).
			Comment("同步金额"),
		field.Enum("sync_status").
			Values("pending_sync", "synced", "sync_failed").
			Default("pending_sync").
			Comment("同步状态"),
		field.Text("sync_error").
			Optional().
			Comment("同步错误信息"),
		field.Int("retry_count").
			Default(0).
			Min(0).
			Comment("重试次数"),
		field.Time("synced_at").
			Optional().
			Nillable().
			Comment("同步完成时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Indexes of the OfflineSyncRecord.
func (OfflineSyncRecord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("offline_id").Unique().StorageKey("idx_offline_id"),
		index.Fields("lot_id", "sync_status").StorageKey("idx_offline_lot_status"),
		index.Fields("sync_status").StorageKey("idx_offline_status"),
	}
}
