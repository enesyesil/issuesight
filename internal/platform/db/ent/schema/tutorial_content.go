package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// TutorialContent holds the schema definition for the TutorialContent entity.
type TutorialContent struct {
	ent.Schema
}

// Fields of the TutorialContent.
func (TutorialContent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("issue_id", uuid.UUID{}).
			Unique().
			Comment("One tutorial content per issue"),
		field.String("title").
			NotEmpty(),
		field.Text("markdown_body").
			Comment("The AI-generated tutorial content"),
		field.Enum("status").
			Values("pending", "completed", "failed").
			Default("pending"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the TutorialContent.
func (TutorialContent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("issue", GithubIssue.Type).
			Ref("tutorial_content").
			Field("issue_id").
			Required().
			Unique(),
		edge.To("tutorials", Tutorial.Type),
		edge.To("concepts", Concept.Type).
			Through("tutorial_concepts", TutorialConcept.Type),
	}
}
