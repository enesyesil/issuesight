// Package domain contains shared business types used across services.
package domain

import "time"

// Issue represents a GitHub issue in our system.
// This is a cleaned-up version, not the raw API response.
type Issue struct {
	ID           int64
	Number       int
	Title        string
	Body         string
	State        string   // "open" or "closed"
	Labels       []string // e.g., ["good first issue", "bug"]
	HTMLURL      string   // link to GitHub
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepoOwner    string // e.g., "kubernetes"
	RepoName     string // e.g., "kubernetes"
	RepoFullName string // e.g., "kubernetes/kubernetes"
}

// Repository represents a GitHub repository.
type Repository struct {
	ID          int64
	FullName    string // "owner/repo"
	Owner       string
	Name        string
	Language    string
	Stars       int
	Description string
}

// IssueURL holds parsed parts of a GitHub issue URL.
type IssueURL struct {
	Owner  string
	Repo   string
	Number int
}

// FullName returns "owner/repo".
func (u IssueURL) FullName() string {
	return u.Owner + "/" + u.Repo
}
