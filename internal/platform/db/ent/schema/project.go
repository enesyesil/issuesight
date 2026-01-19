package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Project holds the schema definition for the Project entity.
type Project struct {
	ent.Schema
}

// Fields of the Project.
func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.Int64("gh_repo_id").
			Unique().
			Comment("GitHub repository ID"),
		field.String("owner_handle").
			NotEmpty(),
		field.String("repo_name").
			NotEmpty(),
		field.String("full_name").
			Unique().
			NotEmpty().
			Comment("owner/repo format"),
		field.String("language").
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the Project.
func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("issues", GithubIssue.Type),
		edge.To("concepts", Concept.Type).
			Through("project_concepts", ProjectConcept.Type),
	}
}
