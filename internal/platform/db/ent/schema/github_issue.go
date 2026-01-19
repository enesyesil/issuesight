package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// GithubIssue holds the schema definition for the GithubIssue entity.
type GithubIssue struct {
	ent.Schema
}

// Fields of the GithubIssue.
func (GithubIssue) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("project_id", uuid.UUID{}),
		field.Int("issue_number").
			Positive(),
		field.Int64("gh_issue_id").
			Unique().
			Comment("GitHub issue ID"),
		field.JSON("raw_data", map[string]interface{}{}).
			Optional().
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Comment("Cached GitHub JSON response"),
		field.Time("last_synced_at").
			Default(time.Now),
	}
}

// Edges of the GithubIssue.
func (GithubIssue) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("issues").
			Field("project_id").
			Required().
			Unique(),
		edge.To("tutorial_content", TutorialContent.Type).
			Unique(),
	}
}
