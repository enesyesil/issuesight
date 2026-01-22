package domain

import (
	"context"

	"github.com/google/uuid"
)

// TutorialRepository handles tutorial persistence.
type TutorialRepository interface {
	Create(ctx context.Context, tutorial *TutorialContent) error
	GetByID(ctx context.Context, id uuid.UUID) (*TutorialContent, error)
	GetByIssueID(ctx context.Context, issueID uuid.UUID) (*TutorialContent, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TutorialListItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status TutorialStatus) error
}

// IssueRepository handles GitHub issue persistence.
type IssueRepository interface {
	Create(ctx context.Context, issue *GithubIssue) error
	GetByID(ctx context.Context, id uuid.UUID) (*GithubIssue, error)
	GetByGitHubID(ctx context.Context, ghIssueID int64) (*GithubIssue, error)
	Upsert(ctx context.Context, issue *GithubIssue) error
}

// UserRepository handles user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	UpdateLastRequestedAt(ctx context.Context, id uuid.UUID) error
}

// UserIdentityRepository handles OAuth identity persistence.
type UserIdentityRepository interface {
	Create(ctx context.Context, identity *UserIdentity) error
	GetByProvider(ctx context.Context, provider, providerID string) (*UserIdentity, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*UserIdentity, error)
}

// ProjectRepository handles project persistence.
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	GetByGitHubID(ctx context.Context, ghRepoID int64) (*Project, error)
	GetByFullName(ctx context.Context, fullName string) (*Project, error)
	Upsert(ctx context.Context, project *Project) error
}

// ConceptRepository handles concept persistence.
type ConceptRepository interface {
	Create(ctx context.Context, concept *Concept) error
	GetBySlug(ctx context.Context, slug string) (*Concept, error)
	List(ctx context.Context, limit, offset int) ([]*Concept, error)
}

// Database entity types (internal, not for API responses)

// TutorialContent is the DB entity for tutorial content.
type TutorialContent struct {
	ID           uuid.UUID
	IssueID      uuid.UUID
	Title        string
	MarkdownBody string
	Status       TutorialStatus
}

// GithubIssue is the DB entity for GitHub issues.
type GithubIssue struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	IssueNumber int
	GhIssueID   int64
	RawData     map[string]interface{} // JSONB
}

// Project is the DB entity for GitHub repositories.
type Project struct {
	ID          uuid.UUID
	GhRepoID    int64
	OwnerHandle string
	RepoName    string
	FullName    string
	Language    string
}
