package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Tutorial holds the schema definition for the Tutorial entity.
// This represents a user's access to tutorial content (unlocked tutorials).
type Tutorial struct {
	ent.Schema
}

// Fields of the Tutorial.
func (Tutorial) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("content_id", uuid.UUID{}),
		field.Bool("is_original_requester").
			Default(false).
			Comment("True if this user originally requested the tutorial"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Tutorial.
func (Tutorial) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("tutorials").
			Field("user_id").
			Required().
			Unique(),
		edge.From("content", TutorialContent.Type).
			Ref("tutorials").
			Field("content_id").
			Required().
			Unique(),
	}
}

// Indexes of the Tutorial.
func (Tutorial) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "content_id").
			Unique(),
	}
}
