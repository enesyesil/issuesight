package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	ghlib "github.com/google/go-github/v60/github"
	"github.com/issuesight/issuesight/internal/domain"
)

// Client wraps the go-github library with IssueSight-specific
// error handling and rate limiting.
type Client struct {
	client    *ghlib.Client
	rateLimit *RateLimitObserver
	logger    *slog.Logger
	token     string
}

// ClientConfig holds configuration for the GitHub client.
type ClientConfig struct {
	Token   string        // GitHub personal access token
	Timeout time.Duration // HTTP request timeout (default: 30s)
	Logger  *slog.Logger  // Logger for rate limit warnings
}

// NewClient creates a new GitHub API client.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Token == "" {
		return nil, ErrEmptyToken
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Create authenticated client
	client := ghlib.NewClient(nil).WithAuthToken(cfg.Token)

	return &Client{
		client:    client,
		rateLimit: NewRateLimitObserver(),
		logger:    cfg.Logger,
		token:     cfg.Token,
	}, nil
}

// FetchIssue retrieves a GitHub issue by owner, repo, and number.
// Returns the issue converted to our domain type.
func (c *Client) FetchIssue(ctx context.Context, owner, repo string, number int) (*domain.Issue, error) {
	if err := validateParams(owner, repo, number); err != nil {
		return nil, err
	}

	// Check rate limit before making request
	if err := c.rateLimit.WaitIfNeeded(ctx, c.logger); err != nil {
		return nil, err
	}

	start := time.Now()

	issue, resp, err := c.client.Issues.Get(ctx, owner, repo, number)

	// Update rate limit info
	c.rateLimit.UpdateFromResponse(resp)

	if err != nil {
		return nil, c.handleError(err, resp, "fetch issue", owner, repo, number)
	}

	c.logger.Debug("fetched issue",
		"owner", owner,
		"repo", repo,
		"number", number,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return toDomainIssue(issue, owner, repo), nil
}

// FetchRepository retrieves GitHub repository metadata.
func (c *Client) FetchRepository(ctx context.Context, owner, repo string) (*domain.Repository, error) {
	if owner == "" {
		return nil, ErrInvalidOwner
	}
	if repo == "" {
		return nil, ErrInvalidRepo
	}

	// Check rate limit before making request
	if err := c.rateLimit.WaitIfNeeded(ctx, c.logger); err != nil {
		return nil, err
	}

	start := time.Now()

	repository, resp, err := c.client.Repositories.Get(ctx, owner, repo)

	// Update rate limit info
	c.rateLimit.UpdateFromResponse(resp)

	if err != nil {
		return nil, c.handleError(err, resp, "fetch repository", owner, repo, 0)
	}

	c.logger.Debug("fetched repository",
		"owner", owner,
		"repo", repo,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return toDomainRepository(repository), nil
}

// FetchIssueComments retrieves all comments for an issue.
// Returns comment bodies as strings.
func (c *Client) FetchIssueComments(ctx context.Context, owner, repo string, number int) ([]string, error) {
	if err := validateParams(owner, repo, number); err != nil {
		return nil, err
	}

	// Check rate limit before making request
	if err := c.rateLimit.WaitIfNeeded(ctx, c.logger); err != nil {
		return nil, err
	}

	start := time.Now()

	// Fetch all comments with pagination
	opts := &ghlib.IssueListCommentsOptions{
		ListOptions: ghlib.ListOptions{PerPage: 100},
	}

	var allComments []string

	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, owner, repo, number, opts)

		// Update rate limit info
		c.rateLimit.UpdateFromResponse(resp)

		if err != nil {
			return nil, c.handleError(err, resp, "fetch comments", owner, repo, number)
		}

		for _, comment := range comments {
			if comment.Body != nil {
				allComments = append(allComments, *comment.Body)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Debug("fetched comments",
		"owner", owner,
		"repo", repo,
		"number", number,
		"count", len(allComments),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return allComments, nil
}

// ValidateToken checks if the configured token is valid by calling /user.
func (c *Client) ValidateToken(ctx context.Context) error {
	_, resp, err := c.client.Users.Get(ctx, "")
	c.rateLimit.UpdateFromResponse(resp)

	if err != nil {
		return c.handleError(err, resp, "validate token", "", "", 0)
	}

	c.logger.Info("github token validated",
		"rate_limit_remaining", resp.Rate.Remaining,
		"rate_limit_reset", resp.Rate.Reset,
	)

	return nil
}

// RateLimitStatus returns the current rate limit status.
func (c *Client) RateLimitStatus() (remaining, limit int, reset time.Time) {
	return c.rateLimit.Status()
}

// handleError converts GitHub API errors to our error types.
func (c *Client) handleError(err error, resp *ghlib.Response, operation, owner, repo string, number int) error {
	if err == nil {
		return nil
	}

	opContext := fmt.Sprintf("%s %s/%s", operation, owner, repo)
	if number > 0 {
		opContext = fmt.Sprintf("%s#%d", opContext, number)
	}

	// Get status code
	statusCode := 0
	if resp != nil && resp.Response != nil {
		statusCode = resp.Response.StatusCode
	}

	// Check for rate limit error
	var rateLimitErr *ghlib.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return fmt.Errorf("%s: %w", opContext, ErrRateLimited)
	}

	// Check for abuse rate limit error (secondary rate limit)
	var abuseErr *ghlib.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		return fmt.Errorf("%s: %w (retry after %v)", opContext, ErrRateLimited, abuseErr.RetryAfter)
	}

	// Handle by status code
	switch statusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%s: %w", opContext, ErrNotFound)
	case http.StatusUnauthorized:
		return fmt.Errorf("%s: %w", opContext, ErrUnauthorized)
	case http.StatusForbidden:
		return fmt.Errorf("%s: %w", opContext, ErrForbidden)
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return fmt.Errorf("%s: %w", opContext, ErrGitHubUnavailable)
	default:
		return fmt.Errorf("%s: %w", opContext, err)
	}
}

// validateParams validates common parameters.
func validateParams(owner, repo string, number int) error {
	if owner == "" {
		return ErrInvalidOwner
	}
	if repo == "" {
		return ErrInvalidRepo
	}
	if number <= 0 {
		return ErrInvalidIssueNumber
	}
	return nil
}

// toDomainIssue converts a GitHub issue to our domain type.
func toDomainIssue(issue *ghlib.Issue, owner, repo string) *domain.Issue {
	if issue == nil {
		return nil
	}

	labels := make([]string, 0, len(issue.Labels))
	for _, l := range issue.Labels {
		if l.Name != nil {
			labels = append(labels, *l.Name)
		}
	}

	di := &domain.Issue{
		RepoOwner:    owner,
		RepoName:     repo,
		RepoFullName: owner + "/" + repo,
		Labels:       labels,
	}

	if issue.ID != nil {
		di.ID = *issue.ID
	}
	if issue.Number != nil {
		di.Number = *issue.Number
	}
	if issue.Title != nil {
		di.Title = *issue.Title
	}
	if issue.Body != nil {
		di.Body = *issue.Body
	}
	if issue.State != nil {
		di.State = *issue.State
	}
	if issue.HTMLURL != nil {
		di.HTMLURL = *issue.HTMLURL
	}
	if issue.CreatedAt != nil {
		di.CreatedAt = issue.CreatedAt.Time
	}
	if issue.UpdatedAt != nil {
		di.UpdatedAt = issue.UpdatedAt.Time
	}

	return di
}

// toDomainRepository converts a GitHub repository to our domain type.
func toDomainRepository(repo *ghlib.Repository) *domain.Repository {
	if repo == nil {
		return nil
	}

	dr := &domain.Repository{}

	if repo.ID != nil {
		dr.ID = *repo.ID
	}
	if repo.FullName != nil {
		dr.FullName = *repo.FullName
	}
	if repo.Owner != nil && repo.Owner.Login != nil {
		dr.Owner = *repo.Owner.Login
	}
	if repo.Name != nil {
		dr.Name = *repo.Name
	}
	if repo.Language != nil {
		dr.Language = *repo.Language
	}
	if repo.StargazersCount != nil {
		dr.Stars = *repo.StargazersCount
	}
	if repo.Description != nil {
		dr.Description = *repo.Description
	}

	return dr
}
