package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Concept holds the schema definition for the Concept entity.
type Concept struct {
	ent.Schema
}

// Fields of the Concept.
func (Concept) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("slug").
			Unique().
			NotEmpty().
			Comment("URL-friendly identifier, e.g. message-queues"),
		field.String("name").
			NotEmpty(),
		field.Text("description").
			Optional(),
		field.String("category").
			Optional().
			Comment("Concept category, e.g. project, language, framework"),
		field.Text("tutorial_markdown").
			Optional().
			Comment("Beginner-friendly concept tutorial in markdown"),
	}
}

// Edges of the Concept.
func (Concept) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("projects", Project.Type).
			Ref("concepts").
			Through("project_concepts", ProjectConcept.Type),
		edge.From("tutorial_contents", TutorialContent.Type).
			Ref("concepts").
			Through("tutorial_concepts", TutorialConcept.Type),
		// Relationships where this concept is the parent
		edge.To("child_relations", ConceptRelationship.Type),
		// Relationships where this concept is the child
		edge.To("parent_relations", ConceptRelationship.Type),
	}
}
