# Phase 4: AI Processor Service Implementation

## Status: 📋 PLANNING

This phase implements the AI Processor service that consumes GitHub issue events from Redis Streams and generates tutorial content using LLMs.

## Architecture

```
Infrastructure
├── External
│   └── LLM Provider (OpenAI)
├── AI Processor Service
│   ├── Config
│   ├── slog Logger
│   ├── Redis Stream (Consumer)
│   └── PostgreSQL (Persistence)
└── main.go
```

## File Structure

```
backend/ai-processor/
├── main.go              # Entry point, bootstrap, graceful shutdown
├── service.go           # Core AI processing logic
├── service_test.go      # Service unit tests
├── llm/
│   ├── client.go       # LLM client wrapper
│   ├── client_test.go  # LLM client tests
│   └── prompts.go      # Prompt templates
├── handler/
│   ├── health.go       # Health endpoint
│   └── health_test.go  # Health tests
└── transformer/
    ├── github.go       # Transform GitHub issue to LLM input
    └── tutorial.go     # Transform LLM output to domain types
```

---

## Implementation Tasks

### 1. Stream Consumer (backend/ai-processor/service.go)

Consume messages from Redis Stream (`github-events`):

- Create consumer group: `ai-workers`
- Consume messages with acknowledgment
- Handle message processing errors
- Implement retry logic for transient failures

**Edge cases to handle:**

- Stream empty (no messages)
- Message processing timeout
- LLM API failures
- Database write failures
- Malformed message payload
- Duplicate message processing (idempotency)

---

### 2. LLM Client Integration (backend/ai-processor/llm/)

Use `internal/platform/llm/` to call LLM provider:

- Generate tutorial content from issue data
- Handle rate limits
- Handle API errors
- Parse LLM responses
- Validate output format

**Edge cases to handle:**

- LLM API rate limits
- LLM API timeouts
- Invalid API key
- Malformed LLM responses
- Empty responses
- Response too large
- Context window exceeded

---

### 3. Prompt Engineering (backend/ai-processor/llm/prompts.go)

Design prompts for tutorial generation:

- Issue analysis prompt
- Prerequisites extraction prompt
- Architecture summary prompt
- Implementation guide prompt
- Concept tagging prompt

**Considerations:**

- Token limits
- Output format (markdown)
- Context window size
- Few-shot examples
- Prompt versioning

---

### 4. Data Transformation (backend/ai-processor/transformer/)

Transform between formats:

- **GitHub → LLM Input:** Convert GitHub issue JSON to prompt context
- **LLM Output → Domain:** Parse LLM markdown to `TUTORIAL_CONTENTS`

**Edge cases to handle:**

- Missing issue fields
- Very large issue bodies
- Binary content in issue body
- Malformed markdown from LLM
- Missing required fields in LLM output

---

### 5. Database Persistence (backend/ai-processor/service.go)

Persist tutorial content to PostgreSQL:

- Create `TUTORIAL_CONTENTS` record
- Link to `GITHUB_ISSUES` via `issue_id`
- Update status (PENDING → COMPLETED/FAILED)
- Handle database errors

**Edge cases to handle:**

- Issue already has tutorial (idempotency)
- Issue deleted between stream and DB write
- Database connection lost
- Transaction failures
- Constraint violations

---

### 6. Main Entry Point (backend/ai-processor/main.go)

Bootstrap and lifecycle:

1. Load config with validation
2. Initialize logger
3. Initialize Redis client (stream consumer)
4. Initialize PostgreSQL client (Ent)
5. Initialize LLM client
6. Start HTTP server (health endpoint)
7. Start stream consumer
8. Handle graceful shutdown

**Edge cases to handle:**

- Config missing required values
- Redis connection failure
- PostgreSQL connection failure
- LLM API key invalid
- Port already in use
- Shutdown timeout exceeded

---

### 7. Health Endpoint (backend/ai-processor/handler/)

Implement `/health` endpoint:

- Check Redis connectivity
- Check PostgreSQL connectivity
- Check LLM API reachability
- Return structured health status

**Response format:**

```json
{
  "status": "healthy",
  "checks": {
    "redis": "ok",
    "postgres": "ok",
    "llm": "ok"
  }
}
```

---

## Testing Strategy

### Unit Tests

- Mock LLM client responses
- Mock Redis stream messages
- Mock database operations
- Test prompt generation
- Test data transformation logic

**Key test files:**

- `llm/client_test.go` - Mock LLM API responses
- `transformer/github_test.go` - GitHub data transformation
- `transformer/tutorial_test.go` - Tutorial parsing
- `service_test.go` - End-to-end service logic with mocks

### Integration Tests

- Real Redis Stream (testcontainers)
- Real PostgreSQL (testcontainers)
- Mock LLM API (HTTP test server)
- Build tag: `//go:build integration`

---

## Error Handling

### Retry Strategy

- **Transient errors:** Retry with exponential backoff
  - LLM API rate limits
  - Network timeouts
  - Database connection lost
- **Permanent errors:** Fail fast, log, and move to next message
  - Invalid issue data
  - LLM API key invalid
  - Database constraint violations

### Error States

- `TUTORIAL_CONTENTS.status = "FAILED"` - Processing failed
- Log error with context (issue ID, error type, retry count)
- Optionally publish to dead-letter queue

---

## Logging

- Log each message consumed (DEBUG)
- Log LLM API calls (INFO with duration)
- Log tutorial creation (INFO)
- Log errors with full context (ERROR)
- Include request ID for tracing

**Example:**
```go
log.Info("tutorial generated",
    "issue_id", issueID,
    "duration_ms", duration.Milliseconds(),
    "tokens_used", tokens,
)
```

---

## Dependencies

```go
// go.mod additions
require (
    github.com/openai/openai-go v1.0.0  // OpenAI SDK (if using)
    // OR use internal/platform/llm/ wrapper
)
```

**Existing dependencies:**
- `internal/platform/stream/` - Stream consumer
- `internal/platform/db/` - Database client
- `internal/platform/llm/` - LLM client wrapper
- `internal/platform/log/` - Structured logging

---

## Performance Considerations

- **Concurrency:** Process multiple messages in parallel (goroutines)
- **Batch processing:** Group related issues for batch LLM calls
- **Caching:** Cache LLM responses for similar issues
- **Rate limiting:** Respect LLM API rate limits
- **Stream lag:** Monitor consumer lag, scale workers if needed

---

## Next Steps

After Phase 4 completion:
1. AI Processor can generate tutorials from GitHub issues
2. Tutorials are persisted to PostgreSQL
3. Ready for Phase 5: Gateway service to serve tutorials to frontend

**See:** [Phase 5: Gateway Service](./phase5-gateway.md)
