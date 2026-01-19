package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// ConceptRelationship holds the schema definition for the ConceptRelationship entity.
// This represents parent-child relationships between concepts.
type ConceptRelationship struct {
	ent.Schema
}

// Fields of the ConceptRelationship.
func (ConceptRelationship) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("parent_id", uuid.UUID{}),
		field.UUID("child_id", uuid.UUID{}),
		field.String("rel_type").
			Default("subconcept_of").
			Comment("Relationship type, e.g. subconcept_of"),
	}
}

// Edges of the ConceptRelationship.
func (ConceptRelationship) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("parent", Concept.Type).
			Ref("child_relations").
			Field("parent_id").
			Required().
			Unique(),
		edge.From("child", Concept.Type).
			Ref("parent_relations").
			Field("child_id").
			Required().
			Unique(),
	}
}

// Indexes of the ConceptRelationship.
func (ConceptRelationship) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_id", "child_id").
			Unique(),
	}
}
