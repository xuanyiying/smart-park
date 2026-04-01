// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// DeviceMetric holds the schema definition for the DeviceMetric entity.
type DeviceMetric struct {
	ent.Schema
}

// Fields of the DeviceMetric.
func (DeviceMetric) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().NotEmpty(),
		field.String("device_id").NotEmpty(),
		field.String("metric").NotEmpty(),
		field.String("value").NotEmpty(),
		field.String("unit").Optional(),
		field.Time("timestamp").Default(time.Now),
	}
}

// Indexes of the DeviceMetric.
func (DeviceMetric) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("id"),
		index.Fields("device_id"),
		index.Fields("metric"),
		index.Fields("timestamp"),
	}
}
