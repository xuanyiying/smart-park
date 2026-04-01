// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FirmwareVersion holds the schema definition for the FirmwareVersion entity.
type FirmwareVersion struct {
	ent.Schema
}

// Fields of the FirmwareVersion.
func (FirmwareVersion) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty(),
		field.String("device_type").NotEmpty(),
		field.String("version").NotEmpty(),
		field.String("url").NotEmpty(),
		field.String("checksum").NotEmpty(),
		field.String("description").Optional(),
		field.Bool("is_active").Default(false),
		field.Time("created_at").Default(time.Now),
	}
}

// Indexes of the FirmwareVersion.
func (FirmwareVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
		index.Fields("device_type", "version"),
		index.Fields("is_active"),
	}
}
