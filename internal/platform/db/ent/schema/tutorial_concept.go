package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// TutorialConcept holds the schema definition for the TutorialConcept entity.
// This is an edge schema (join table) linking tutorial contents to concepts.
type TutorialConcept struct {
	ent.Schema
}

// Annotations of the TutorialConcept.
func (TutorialConcept) Annotations() []schema.Annotation {
	return []schema.Annotation{
		field.ID("content_id", "concept_id"),
	}
}

// Fields of the TutorialConcept.
func (TutorialConcept) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("content_id", uuid.UUID{}),
		field.UUID("concept_id", uuid.UUID{}),
	}
}

// Edges of the TutorialConcept.
func (TutorialConcept) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("content", TutorialContent.Type).
			Unique().
			Required().
			Field("content_id"),
		edge.To("concept", Concept.Type).
			Unique().
			Required().
			Field("concept_id"),
	}
}
