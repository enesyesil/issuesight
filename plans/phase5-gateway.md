# Phase 5: Gateway Service Implementation

## Status: ✅ COMPLETE

This phase implements the API Gateway service that serves the frontend and manages authentication, quotas, and request routing.

## Architecture

```
Infrastructure
├── External
│   └── Frontend (Next.js)
├── Gateway Service
│   ├── Config
│   ├── slog Logger
│   ├── Redis (Cache, Locks)
│   ├── MongoDB (Auth & Quotas)
│   └── PostgreSQL (Tutorial Data)
└── main.go
```

## File Structure

```
backend/gateway/
├── main.go              # Entry point, bootstrap, graceful shutdown
├── server.go            # HTTP server setup
├── handler/
│   ├── issue.go        # Issue submission endpoint
│   ├── tutorial.go     # Tutorial retrieval endpoints
│   ├── health.go       # Health endpoint
│   └── handler_test.go # Handler tests
├── middleware/
│   ├── auth.go         # Authentication middleware
│   ├── quota.go        # Quota checking middleware
│   ├── logging.go      # Request logging
│   └── recovery.go     # Panic recovery
├── auth/
│   ├── oauth.go        # OAuth provider integration
│   ├── session.go      # Session management
│   └── user.go         # User lookup/creation
├── quota/
│   ├── checker.go      # Quota validation
│   └── mongodb.go      # MongoDB quota storage
└── lock/
    └── redis.go        # Distributed lock for duplicate prevention
```

---

## Implementation Tasks

### 1. HTTP Server Setup (backend/gateway/server.go)

Set up HTTP server with routing:

- Gin framework or standard library
- Route registration
- Middleware chain
- Graceful shutdown
- CORS configuration

**Endpoints:**

- `POST /api/issues` - Submit issue URL for processing
- `GET /api/tutorials/:id` - Get tutorial by ID
- `GET /api/tutorials` - List user's tutorials
- `GET /health` - Health check
- `GET /api/auth/callback` - OAuth callback
- `POST /api/auth/logout` - Logout

---

### 2. Authentication (backend/gateway/auth/)

OAuth integration for GitHub and Google:

- OAuth flow initiation
- OAuth callback handling
- Session management (JWT or session cookies)
- User lookup/creation in database
- Multi-provider support (GitHub, Google)

**Edge cases to handle:**

- Invalid OAuth state
- OAuth provider errors
- User already exists with different provider
- Session expiration
- Invalid/expired tokens

---

### 3. Quota Management (backend/gateway/quota/)

Rate limiting per user:

- Check `USERS.last_requested_at` timestamp
- MongoDB integration for quota storage (if needed)
- Quota middleware to block requests
- Quota reset logic

**Quota rules:**

- X requests per day per user
- Track in `USERS.last_requested_at`
- Update on each request

**Edge cases to handle:**

- Quota exceeded
- MongoDB connection failure
- Race conditions (concurrent requests)
- Quota reset timing

---

### 4. Distributed Locking (backend/gateway/lock/)

Prevent duplicate issue submissions:

- Use `internal/platform/lock/` for Redis locks
- Lock key: `lock:issue:{owner}/{repo}/{number}`
- Lock TTL: 5 minutes
- Release on completion or error

**Edge cases to handle:**

- Lock already held (duplicate request)
- Lock timeout
- Lock release failure

---

### 5. Issue Submission Handler (backend/gateway/handler/issue.go)

Handle `POST /api/issues`:

1. Authenticate user (middleware)
2. Check quota (middleware)
3. Parse request body (issue URL)
4. Acquire distributed lock
5. Validate issue URL
6. Check if already processed
7. Publish to Redis Stream (`github-events`)
8. Return response with issue ID

**Request format:**
```json
{
  "url": "https://github.com/owner/repo/issues/123"
}
```

**Response format:**
```json
{
  "id": "uuid",
  "status": "queued",
  "message": "Issue queued for processing"
}
```

**Edge cases to handle:**

- Invalid URL format
- Issue already processed
- Stream publish failure
- User quota exceeded
- Duplicate request (lock held)

---

### 6. Tutorial Retrieval Handler (backend/gateway/handler/tutorial.go)

Handle tutorial retrieval:

- `GET /api/tutorials/:id` - Get specific tutorial
- `GET /api/tutorials` - List user's tutorials
- Check user access (only original requester or unlocked)
- Cache popular tutorials in Redis

**Cache strategy:**

- Cache key: `cache:tutorial:{id}`
- TTL: 5 minutes
- Cache-aside pattern

**Edge cases to handle:**

- Tutorial not found
- User doesn't have access
- Cache miss (fetch from DB)
- Cache corruption

---

### 7. Middleware Chain (backend/gateway/middleware/)

Request processing pipeline:

1. **Recovery** - Panic recovery
2. **Logging** - Request/response logging with request ID
3. **Auth** - Authentication check
4. **Quota** - Quota validation (for write endpoints)
5. **Handler** - Business logic

---

### 8. Main Entry Point (backend/gateway/main.go)

Bootstrap and lifecycle:

1. Load config with validation
2. Initialize logger
3. Initialize Redis client (cache, locks)
4. Initialize PostgreSQL client (Ent)
5. Initialize MongoDB client (if used for quotas)
6. Initialize OAuth providers
7. Start HTTP server
8. Handle graceful shutdown

**Edge cases to handle:**

- Config missing required values
- Redis connection failure
- PostgreSQL connection failure
- MongoDB connection failure (if used)
- OAuth provider config invalid
- Port already in use

---

### 9. Health Endpoint (backend/gateway/handler/health.go)

Implement `/health` endpoint:

- Check Redis connectivity
- Check PostgreSQL connectivity
- Check MongoDB connectivity (if used)
- Return structured health status

**Response format:**

```json
{
  "status": "healthy",
  "checks": {
    "redis": "ok",
    "postgres": "ok",
    "mongodb": "ok"
  }
}
```

---

## Testing Strategy

### Unit Tests

- Mock Redis (cache, locks)
- Mock database operations
- Mock OAuth providers
- Test middleware chain
- Test handler logic

**Key test files:**

- `handler/issue_test.go` - Issue submission tests
- `handler/tutorial_test.go` - Tutorial retrieval tests
- `middleware/auth_test.go` - Auth middleware tests
- `middleware/quota_test.go` - Quota middleware tests
- `auth/oauth_test.go` - OAuth flow tests

### Integration Tests

- Real Redis (testcontainers)
- Real PostgreSQL (testcontainers)
- Mock OAuth providers (HTTP test server)
- Build tag: `//go:build integration`

---

## Error Handling

### HTTP Error Responses

- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Not authenticated
- `403 Forbidden` - Quota exceeded or no access
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

### Error Response Format

```json
{
  "error": "quota_exceeded",
  "message": "Daily quota exceeded. Try again tomorrow.",
  "retry_after": 3600
}
```

---

## Logging

- Log all HTTP requests (INFO)
- Log authentication events (INFO)
- Log quota violations (WARN)
- Log errors with full context (ERROR)
- Include request ID in all logs

**Example:**
```go
log.Info("issue submitted",
    "user_id", userID,
    "issue_url", url,
    "request_id", requestID,
)
```

---

## Dependencies

```go
// go.mod additions
require (
    github.com/gin-gonic/gin v1.9.0  // HTTP framework (optional)
    go.mongodb.org/mongo-driver v1.0.0  // MongoDB driver (if using)
)
```

**Existing dependencies:**
- `internal/platform/cache/` - Cache operations
- `internal/platform/lock/` - Distributed locking
- `internal/platform/db/` - Database client
- `internal/platform/log/` - Structured logging

---

## Security Considerations

- **CORS:** Configure allowed origins
- **CSRF:** Protect state-changing endpoints
- **Rate Limiting:** Per-user and per-IP limits
- **Input Validation:** Validate all user inputs
- **SQL Injection:** Use parameterized queries (Ent handles this)
- **XSS:** Sanitize user inputs
- **Secrets:** Never log tokens, API keys, passwords

---

## Performance Considerations

- **Caching:** Cache popular tutorials in Redis
- **Connection Pooling:** Use connection pools for DB and Redis
- **Async Processing:** Issue submission is async (stream-based)
- **Load Balancing:** Stateless design for horizontal scaling

---

## Next Steps

After Phase 5 completion:
1. Gateway serves frontend requests
2. Authentication and quotas are enforced
3. Issues are queued for processing
4. Tutorials are served to users
5. Ready for Phase 6: Frontend implementation

**See:** [Phase 6: Frontend Service](./phase6-frontend.md)
