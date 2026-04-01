// Package schema defines the schema for the device service
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Command holds the schema definition for the Command entity.
type Command struct {
	ent.Schema
}

// Fields of the Command.
func (Command) Fields() []ent.Field {
	return []ent.Field{
		field.String("command_id").Unique().NotEmpty(),
		field.String("device_id").NotEmpty(),
		field.String("command").NotEmpty(),
		field.JSON("params", map[string]string{}).Optional(),
		field.String("status").Default("pending"),
		field.Time("created_at").Default(time.Now),
		field.Time("executed_at").Optional(),
		field.String("result").Optional(),
	}
}

// Indexes of the Command.
func (Command) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("command_id"),
		index.Fields("device_id"),
		index.Fields("status"),
	}
}
