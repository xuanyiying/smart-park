// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FirmwareUpdate holds the schema definition for the FirmwareUpdate entity.
type FirmwareUpdate struct {
	ent.Schema
}

// Fields of the FirmwareUpdate.
func (FirmwareUpdate) Fields() []ent.Field {
	return []ent.Field{
		field.String("update_id").Unique().NotEmpty(),
		field.String("device_id").NotEmpty(),
		field.String("firmware_version").NotEmpty(),
		field.String("status").Default("pending"),
		field.String("progress").Default("0%"),
		field.String("error_message").Optional(),
		field.Time("started_at").Default(time.Now),
		field.Time("completed_at").Optional(),
	}
}

// Indexes of the FirmwareUpdate.
func (FirmwareUpdate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("update_id"),
		index.Fields("device_id"),
		index.Fields("status"),
	}
}
