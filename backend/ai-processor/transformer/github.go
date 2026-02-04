// Package transformer provides data transformation utilities for the AI Processor.
//
// It handles converting between different data formats:
//   - Redis stream messages → StreamIssuePayload (wire format)
//   - LLM responses → domain.TutorialContent
package transformer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/issuesight/issuesight/internal/domain"
)

// Errors for message transformation.
var (
	ErrEmptyPayload   = errors.New("transformer: payload cannot be empty")
	ErrMissingField   = errors.New("transformer: required field missing")
	ErrFieldTooLong   = errors.New("transformer: field exceeds maximum length")
	ErrInvalidPayload = errors.New("transformer: payload validation failed")
)

// Field length limits to prevent memory issues and ensure reasonable data sizes.
const (
	MaxOwnerLength           = 100    // GitHub username max is 39, but allow some buffer
	MaxRepoLength            = 256    // GitHub repo name max is 100, but allow some buffer
	MaxTitleLength           = 1024   // Reasonable title length
	MaxBodyLength            = 500000 // ~500KB body limit (GitHub allows ~65K but may include images)
	MaxURLLength             = 2048   // Standard URL length limit
	MaxLanguageLength        = 100    // Programming language name
	MaxRepoDescriptionLength = 2000   // Repository description length limit
)

// StreamIssuePayload represents the wire format from Redis stream.
// This combines data from Issue + Repository + collection metadata.
// This is NOT a domain type - it's a DTO for the stream message format.
type StreamIssuePayload struct {
	IssueID         int64     `json:"issue_id"`
	IssueNumber     int       `json:"issue_number"`
	Owner           string    `json:"owner"`
	Repo            string    `json:"repo"`
	FullName        string    `json:"full_name"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	Labels          []string  `json:"labels"`
	State           string    `json:"state"`
	HTMLURL         string    `json:"html_url"`
	RepoID          int64     `json:"repo_id"`
	RepoLanguage    string    `json:"repo_language"`
	RepoStars       int       `json:"repo_stars"`
	RepoDescription string    `json:"repo_description"`
	CollectedAt     time.Time `json:"collected_at"`
	UserID          string    `json:"user_id"` // Requesting user for access control
}

// ToIssue converts the stream payload to a domain.Issue.
func (p *StreamIssuePayload) ToIssue() *domain.Issue {
	return &domain.Issue{
		ID:           p.IssueID,
		Number:       p.IssueNumber,
		Title:        p.Title,
		Body:         p.Body,
		State:        p.State,
		Labels:       p.Labels,
		HTMLURL:      p.HTMLURL,
		RepoOwner:    p.Owner,
		RepoName:     p.Repo,
		RepoFullName: p.FullName,
	}
}

// ToRepository converts the stream payload to a domain.Repository.
func (p *StreamIssuePayload) ToRepository() *domain.Repository {
	return &domain.Repository{
		ID:       p.RepoID,
		FullName: p.FullName,
		Owner:    p.Owner,
		Name:     p.Repo,
		Language: p.RepoLanguage,
		Stars:    p.RepoStars,
	}
}

// TransformStreamMessage converts a Redis stream message payload into StreamIssuePayload.
//
// The payload comes from the collector service and contains:
//   - issue_id, issue_number, owner, repo, full_name
//   - title, body, labels (JSON string), state, html_url
//   - repo_id, repo_language, repo_stars
//   - collected_at (RFC3339 timestamp)
func TransformStreamMessage(payload map[string]interface{}) (*StreamIssuePayload, error) {
	if payload == nil || len(payload) == 0 {
		return nil, ErrEmptyPayload
	}

	p := &StreamIssuePayload{}

	// Required string fields with length validation
	var ok bool
	if p.Owner, ok = getStringField(payload, "owner"); !ok {
		return nil, fmt.Errorf("%w: owner", ErrMissingField)
	}
	if len(p.Owner) > MaxOwnerLength {
		return nil, fmt.Errorf("%w: owner exceeds %d chars", ErrFieldTooLong, MaxOwnerLength)
	}

	if p.Repo, ok = getStringField(payload, "repo"); !ok {
		return nil, fmt.Errorf("%w: repo", ErrMissingField)
	}
	if len(p.Repo) > MaxRepoLength {
		return nil, fmt.Errorf("%w: repo exceeds %d chars", ErrFieldTooLong, MaxRepoLength)
	}

	if p.Title, ok = getStringField(payload, "title"); !ok {
		return nil, fmt.Errorf("%w: title", ErrMissingField)
	}
	if len(p.Title) > MaxTitleLength {
		p.Title = p.Title[:MaxTitleLength] // Truncate title instead of failing
	}

	// Optional string fields (with defaults and length limits)
	p.FullName, _ = getStringField(payload, "full_name")
	// Be resilient: some producers may omit full_name; derive it.
	if p.FullName == "" {
		p.FullName = fmt.Sprintf("%s/%s", p.Owner, p.Repo)
	}
	p.Body, _ = getStringField(payload, "body")
	if len(p.Body) > MaxBodyLength {
		p.Body = p.Body[:MaxBodyLength] // Truncate body instead of failing
	}
	p.State, _ = getStringField(payload, "state")
	p.HTMLURL, _ = getStringField(payload, "html_url")
	if len(p.HTMLURL) > MaxURLLength {
		p.HTMLURL = p.HTMLURL[:MaxURLLength]
	}
	p.RepoLanguage, _ = getStringField(payload, "repo_language")
	if len(p.RepoLanguage) > MaxLanguageLength {
		p.RepoLanguage = p.RepoLanguage[:MaxLanguageLength]
	}
	p.RepoDescription, _ = getStringField(payload, "repo_description")
	if len(p.RepoDescription) > MaxRepoDescriptionLength {
		p.RepoDescription = p.RepoDescription[:MaxRepoDescriptionLength]
	}

	// Required numeric fields
	var err error
	if p.IssueID, err = getInt64Field(payload, "issue_id"); err != nil {
		return nil, fmt.Errorf("%w: issue_id: %v", ErrMissingField, err)
	}

	// Optional numeric fields
	p.IssueNumber, _ = getIntField(payload, "issue_number")
	p.RepoID, _ = getInt64Field(payload, "repo_id")
	p.RepoStars, _ = getIntField(payload, "repo_stars")

	// Parse labels from JSON string
	if labelsStr, ok := getStringField(payload, "labels"); ok && labelsStr != "" {
		var labels []string
		if err := json.Unmarshal([]byte(labelsStr), &labels); err != nil {
			// Labels parsing failed, use empty slice
			p.Labels = []string{}
		} else {
			p.Labels = labels
		}
	}

	// Parse collected_at timestamp
	if collectedAtStr, ok := getStringField(payload, "collected_at"); ok {
		if t, err := time.Parse(time.RFC3339, collectedAtStr); err == nil {
			p.CollectedAt = t
		} else {
			p.CollectedAt = time.Now().UTC()
		}
	} else {
		p.CollectedAt = time.Now().UTC()
	}

	// Parse user_id (optional - for access control)
	p.UserID, _ = getStringField(payload, "user_id")

	return p, nil
}

// getStringField extracts a string field from the payload.
func getStringField(payload map[string]interface{}, key string) (string, bool) {
	val, exists := payload[key]
	if !exists {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	case []byte:
		return string(v), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// getInt64Field extracts an int64 field from the payload.
func getInt64Field(payload map[string]interface{}, key string) (int64, error) {
	val, exists := payload[key]
	if !exists {
		return 0, fmt.Errorf("field %s not found", key)
	}

	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", val)
	}
}

// getIntField extracts an int field from the payload.
func getIntField(payload map[string]interface{}, key string) (int, error) {
	val, err := getInt64Field(payload, key)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}
