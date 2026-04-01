// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DeviceStatusHistory holds the schema definition for the DeviceStatusHistory entity.
type DeviceStatusHistory struct {
	ent.Schema
}

// Fields of the DeviceStatusHistory.
func (DeviceStatusHistory) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty(),
		field.String("device_id").NotEmpty(),
		field.String("status").NotEmpty(),
		field.Bool("online").Default(false),
		field.String("firmware_version").Optional(),
		field.Time("timestamp").Default(time.Now),
		field.JSON("metadata", map[string]string{}).Optional(),
	}
}

// Indexes of the DeviceStatusHistory.
func (DeviceStatusHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
		index.Fields("device_id"),
		index.Fields("timestamp"),
	}
}
