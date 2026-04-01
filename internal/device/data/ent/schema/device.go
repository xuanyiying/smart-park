// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Device holds the schema definition for the Device entity.
type Device struct {
	ent.Schema
}

// Fields of the Device.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.String("device_id").Unique().NotEmpty(),
		field.String("device_type").NotEmpty(),
		field.String("status").Default("active"),
		field.Bool("online").Default(false),
		field.Time("last_heartbeat").Optional(),
		field.String("lot_id").Optional(),
		field.String("lane_id").Optional(),
		field.String("protocol").Default("http"),
		field.JSON("config", map[string]string{}).Optional(),
		field.String("firmware_version").Default("1.0.0"),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Indexes of the Device.
func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id"),
		index.Fields("device_type"),
		index.Fields("status"),
		index.Fields("online"),
	}
}
