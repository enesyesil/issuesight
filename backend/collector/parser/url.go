// Package parser provides utilities for parsing GitHub URLs.
package parser

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/issuesight/issuesight/internal/domain"
)

// Sentinel errors for URL parsing failures.
var (
	ErrEmptyURL           = errors.New("parser: url cannot be empty")
	ErrInvalidURL         = errors.New("parser: invalid url format")
	ErrInvalidHost        = errors.New("parser: not a github.com url")
	ErrInvalidPath        = errors.New("parser: invalid issue path")
	ErrPullRequestURL     = errors.New("parser: pull request urls not supported")
	ErrInvalidIssueNumber = errors.New("parser: invalid issue number")
)

// githubHostRegex matches github.com and www.github.com
var githubHostRegex = regexp.MustCompile(`^(www\.)?github\.com$`)

// Parse extracts owner, repo, and issue number from a GitHub issue URL.
//
// Supported formats:
//   - https://github.com/owner/repo/issues/123
//   - http://github.com/owner/repo/issues/123
//   - github.com/owner/repo/issues/123
//   - https://www.github.com/owner/repo/issues/123
//
// The function handles:
//   - Missing protocol (auto-prepends https://)
//   - Trailing slashes
//   - Query parameters and hash fragments
//
// Returns ErrPullRequestURL for pull request URLs.
func Parse(rawURL string) (*domain.IssueURL, error) {
	// Trim whitespace
	rawURL = strings.TrimSpace(rawURL)

	// Check for empty input
	if rawURL == "" {
		return nil, ErrEmptyURL
	}

	// Add protocol if missing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	// Parse the URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidURL
	}

	// Validate host is github.com
	if !githubHostRegex.MatchString(parsed.Host) {
		return nil, ErrInvalidHost
	}

	// Parse path: /owner/repo/issues/123 or /owner/repo/pull/123
	// Remove leading and trailing slashes, then split
	path := strings.Trim(parsed.Path, "/")
	parts := strings.Split(path, "/")

	// We need exactly 4 parts: owner, repo, issues|pull, number
	if len(parts) != 4 {
		return nil, ErrInvalidPath
	}

	owner := parts[0]
	repo := parts[1]
	issueType := parts[2]
	numberStr := parts[3]

	// Validate owner and repo are not empty
	if owner == "" || repo == "" {
		return nil, ErrInvalidPath
	}

	// Check if it's a pull request URL
	if issueType == "pull" {
		return nil, ErrPullRequestURL
	}

	// Validate it's an issues URL
	if issueType != "issues" {
		return nil, ErrInvalidPath
	}

	// Parse issue number
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return nil, ErrInvalidIssueNumber
	}

	// Validate issue number is positive
	if number <= 0 {
		return nil, ErrInvalidIssueNumber
	}

	return &domain.IssueURL{
		Owner:  owner,
		Repo:   repo,
		Number: number,
	}, nil
}
