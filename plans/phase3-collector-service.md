# Phase 3: Collector Service Implementation

## Architecture

```
Infrastructure
├── External
│   └── GitHub API
├── Collector Service
│   ├── Config
│   ├── slog Logger
│   └── Redis Stream
└── main.go
```

## File Structure

```
backend/collector/
├── main.go              # Entry point, bootstrap, graceful shutdown
├── service.go           # Core collector logic
├── service_test.go      # Service unit tests
├── github/
│   ├── client.go        # go-github wrapper
│   ├── client_test.go   # GitHub client tests
│   └── ratelimit.go     # Rate limit handling
├── parser/
│   ├── url.go           # GitHub URL parsing
│   └── url_test.go      # Parser tests
└── handler/
    ├── health.go        # Health endpoint
    └── health_test.go   # Health tests
```

---

## Implementation Tasks

### 1. Logging Setup (internal/platform/log/) - COMPLETED

The logging package has been implemented separately. See `internal/platform/log/`:

- `log.go` - Config, New(), level parsing
- `handler.go` - JSON and text handlers with redaction
- `context.go` - Request ID propagation
- `redact.go` - Sensitive data redaction
- `log_test.go` - Comprehensive unit tests

---

### 2. GitHub Client (backend/collector/github/)

Wrap go-github with our abstractions:

- `FetchIssue(ctx, owner, repo, number)` - Get issue details
- `FetchRepository(ctx, owner, repo)` - Get repo metadata
- `FetchIssueComments(ctx, owner, repo, number)` - Get comments

**Edge cases to handle:**

- 404 Not Found (issue/repo doesn't exist)
- 403 Forbidden (private repo, no access)
- 401 Unauthorized (bad/expired token)
- 429 Rate Limited (primary + secondary limits)
- 5xx Server Errors (GitHub down)
- Network timeouts
- Context cancellation
- Empty/nil responses
- Malformed JSON responses

**Rate limiting strategy:**

- Check `X-RateLimit-Remaining` header
- Sleep until `X-RateLimit-Reset` when exhausted
- Implement exponential backoff for 5xx errors
- Log rate limit status periodically

---

### 3. URL Parser (backend/collector/parser/)

Parse GitHub issue URLs into components:

- Input: `https://github.com/owner/repo/issues/123`
- Output: `{Owner: "owner", Repo: "repo", Number: 123}`

**Edge cases to handle:**

- Missing protocol (`github.com/...`)
- Trailing slashes
- Query parameters (`?query=1`)
- Hash fragments (`#comment`)
- Pull request URLs (reject or handle?)
- Non-github.com URLs (enterprise GitHub?)
- Invalid issue numbers (0, negative, non-numeric)
- Unicode in owner/repo names
- Very long URLs
- Empty strings
- Whitespace around URL

---

### 4. Collector Service (backend/collector/service.go)

Core business logic:

1. Receive issue URL from stream/API
2. Parse and validate URL
3. Check if already processed (idempotency)
4. Fetch issue from GitHub
5. Fetch repository metadata
6. Transform to domain types
7. Publish to `github-events` stream

**Edge cases to handle:**

- Duplicate requests (same issue URL)
- Issue was deleted between check and fetch
- Repository made private
- Issue closed/reopened during processing
- Very large issue bodies (>1MB)
- Issues with 1000+ comments
- Binary content in issue body
- Malformed markdown

---

### 5. Main Entry Point (backend/collector/main.go)

Bootstrap and lifecycle:

1. Load config with validation
2. Initialize logger
3. Initialize Redis client with health check
4. Initialize GitHub client
5. Start HTTP server (health endpoint)
6. Start stream consumer (optional, for async mode)
7. Handle graceful shutdown (SIGTERM, SIGINT)

**Edge cases to handle:**

- Config missing required values
- Redis connection failure on startup
- Redis connection lost during runtime
- GitHub token invalid on startup
- Port already in use
- Shutdown timeout exceeded
- Panic recovery

---

### 6. Health Endpoint (backend/collector/handler/)

Implement `/health` endpoint:

- Check Redis connectivity
- Check GitHub API reachability
- Return structured health status

**Response format:**

```json
{
  "status": "healthy",
  "checks": {
    "redis": "ok",
    "github": "ok"
  }
}
```

---

## Testing Strategy

### Unit Tests

Every package should have `*_test.go`:

- Table-driven tests for all edge cases
- Mock external dependencies (GitHub API, Redis)
- Test error paths, not just happy path
- Aim for >80% coverage

**Key test files:**

- `parser/url_test.go` - All URL parsing edge cases
- `github/client_test.go` - Mock HTTP responses
- `service_test.go` - Mock GitHub client + Redis

### Integration Tests

Use `_integration_test.go` suffix:

- Real Redis (via testcontainers or local)
- GitHub API with test token (optional, CI only)
- Build tag: `//go:build integration`

### Test Helpers

Create `internal/testutil/`:

- Mock Redis client
- Mock HTTP server for GitHub
- Test fixtures (sample issues, repos)
- Context with test logger

---

## Error Handling Principles

- Wrap errors with context: `fmt.Errorf("fetch issue %d: %w", num, err)`
- Use sentinel errors: Check with `errors.Is(err, ErrNotFound)`
- Log at boundaries: Log when error crosses package boundary
- Don't log and return: Either log OR return, not both
- Fail fast on startup: Exit if critical deps unavailable
- Retry transient errors: Network issues, rate limits
- Don't retry permanent errors: 404, 401, invalid input

---

## Logging Principles

- Structured fields: `slog.Info("fetched issue", "owner", o, "repo", r, "number", n)`
- Log levels:
  - **ERROR**: Something failed, needs attention
  - **WARN**: Something unexpected but handled
  - **INFO**: Normal operations (startup, shutdown, major events)
  - **DEBUG**: Detailed debugging info
- Include request ID: Propagate via context for tracing
- Redact secrets: Never log tokens, keys, passwords
- Log timing: Duration for external calls (GitHub, Redis)

---

## Dependencies to Add

```go
// go.mod additions
require (
    github.com/google/go-github/v60 v60.0.0  // GitHub API client
    github.com/redis/go-redis/v9 v9.0.0      // Redis client (already added)
)
```
