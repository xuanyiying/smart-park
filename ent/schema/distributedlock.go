package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DistributedLock holds the schema definition for the DistributedLock entity.
// 分布式锁表(用于出场计费等关键操作)
type DistributedLock struct {
	ent.Schema
}

// Fields of the DistributedLock.
func (DistributedLock) Fields() []ent.Field {
	return []ent.Field{
		field.String("lock_key").
			MaxLen(128).
			NotEmpty().
			Comment("锁键名"),
		field.String("owner").
			MaxLen(64).
			NotEmpty().
			Comment("锁持有者标识"),
		field.Time("expire_at").
			Comment("过期时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Indexes of the DistributedLock.
func (DistributedLock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lock_key").Unique().StorageKey("idx_lock_key"),
		index.Fields("expire_at").StorageKey("idx_locks_expire"),
	}
}
