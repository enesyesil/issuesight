package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// ProjectConcept holds the schema definition for the ProjectConcept entity.
// This is an edge schema (join table) linking projects to concepts.
type ProjectConcept struct {
	ent.Schema
}

// Annotations of the ProjectConcept.
func (ProjectConcept) Annotations() []schema.Annotation {
	return []schema.Annotation{
		field.ID("project_id", "concept_id"),
	}
}

// Fields of the ProjectConcept.
func (ProjectConcept) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("project_id", uuid.UUID{}),
		field.UUID("concept_id", uuid.UUID{}),
	}
}

// Edges of the ProjectConcept.
func (ProjectConcept) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("project", Project.Type).
			Unique().
			Required().
			Field("project_id"),
		edge.To("concept", Concept.Type).
			Unique().
			Required().
			Field("concept_id"),
	}
}
