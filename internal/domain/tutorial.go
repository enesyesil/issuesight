package domain

import (
	"time"

	"github.com/google/uuid"
)

// TutorialStatus represents the generation state of a tutorial.
type TutorialStatus string

const (
	StatusPending   TutorialStatus = "PENDING"
	StatusCompleted TutorialStatus = "COMPLETED"
	StatusFailed    TutorialStatus = "FAILED"
)

// TutorialRequest is what a user submits to generate a tutorial.
type TutorialRequest struct {
	IssueURL string    // GitHub issue URL
	UserID   uuid.UUID // who requested it
}

// TutorialResponse is what the API returns.
type TutorialResponse struct {
	ID           uuid.UUID      `json:"id"`
	Title        string         `json:"title"`
	MarkdownBody string         `json:"markdown_body"`
	Status       TutorialStatus `json:"status"`
	Issue        *Issue         `json:"issue,omitempty"`
	Concepts     []Concept      `json:"concepts,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// Concept represents a programming concept used for tagging.
type Concept struct {
	Slug        string `json:"slug"`        // e.g., "message-queues"
	Name        string `json:"name"`        // e.g., "Message Queues"
	Description string `json:"description"` // brief explanation
}

// TutorialListItem is a lightweight version for list endpoints.
type TutorialListItem struct {
	ID        uuid.UUID      `json:"id"`
	Title     string         `json:"title"`
	Status    TutorialStatus `json:"status"`
	RepoName  string         `json:"repo_name"`
	CreatedAt time.Time      `json:"created_at"`
}
