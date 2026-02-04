// Package github provides a wrapper around the go-github library
// with error handling, rate limiting, and IssueSight-specific abstractions.
package github

import (
	"errors"
)

// Sentinel errors for GitHub API operations.
var (
	// ErrNotFound is returned when the requested resource doesn't exist (404).
	ErrNotFound = errors.New("github: resource not found")

	// ErrUnauthorized is returned when authentication fails (401).
	ErrUnauthorized = errors.New("github: unauthorized - check token")

	// ErrForbidden is returned when access is denied (403).
	// This can happen for private repos or when rate limited.
	ErrForbidden = errors.New("github: forbidden - no access or rate limited")

	// ErrRateLimited is returned specifically for rate limit errors.
	ErrRateLimited = errors.New("github: rate limit exceeded")

	// ErrGitHubUnavailable is returned for server errors (5xx).
	ErrGitHubUnavailable = errors.New("github: service unavailable")

	// ErrEmptyToken is returned when no GitHub token is provided.
	ErrEmptyToken = errors.New("github: token cannot be empty")

	// ErrInvalidOwner is returned when owner is empty.
	ErrInvalidOwner = errors.New("github: owner cannot be empty")

	// ErrInvalidRepo is returned when repo is empty.
	ErrInvalidRepo = errors.New("github: repo cannot be empty")

	// ErrInvalidIssueNumber is returned when issue number is <= 0.
	ErrInvalidIssueNumber = errors.New("github: issue number must be positive")
)
