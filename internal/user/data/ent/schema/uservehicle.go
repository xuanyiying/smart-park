package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// UserVehicle holds the schema definition for the UserVehicle entity.
type UserVehicle struct {
	ent.Schema
}

// Fields of the UserVehicle.
func (UserVehicle) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id"),
		field.UUID("user_id", uuid.UUID{}).
			Comment("用户ID"),
		field.String("plate_number").
			MaxLen(20).
			Comment("车牌号"),
		field.String("owner_name").
			MaxLen(100).
			Optional().
			Comment("车主姓名"),
		field.String("owner_phone").
			MaxLen(20).
			Optional().
			Sensitive().
			Comment("车主电话"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the UserVehicle.
func (UserVehicle) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("plate_number"),
		index.Fields("user_id", "plate_number").Unique(),
	}
}
