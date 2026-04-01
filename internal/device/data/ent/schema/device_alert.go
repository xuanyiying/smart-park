// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DeviceAlert holds the schema definition for the DeviceAlert entity.
type DeviceAlert struct {
	ent.Schema
}

// Fields of the DeviceAlert.
func (DeviceAlert) Fields() []ent.Field {
	return []ent.Field{
		field.String("alert_id").Unique().NotEmpty(),
		field.String("device_id").NotEmpty(),
		field.String("type").NotEmpty(),
		field.String("severity").NotEmpty(),
		field.String("message").NotEmpty(),
		field.String("status").Default("active"),
		field.Time("created_at").Default(time.Now),
		field.Time("acknowledged_at").Optional(),
		field.String("acknowledged_by").Optional(),
		field.String("notes").Optional(),
		field.JSON("metadata", map[string]string{}).Optional(),
	}
}

// Indexes of the DeviceAlert.
func (DeviceAlert) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("alert_id"),
		index.Fields("device_id"),
		index.Fields("severity"),
		index.Fields("status"),
	}
}
