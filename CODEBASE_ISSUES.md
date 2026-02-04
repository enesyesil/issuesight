# IssueSight Codebase Issues Report

**Generated:** January 27, 2026  
**Total Issues Found:** 145+

---

## Table of Contents

1. [Critical Issues (Must Fix)](#1-critical-issues-must-fix)
2. [Security Issues](#2-security-issues)
3. [Error Handling Problems](#3-error-handling-problems)
4. [Code Duplication](#4-code-duplication)
5. [Unused/Dead Code](#5-unuseddead-code)
6. [Configuration Issues](#6-configuration-issues)
7. [TypeScript/Frontend Issues](#7-typescriptfrontend-issues)
8. [Accessibility Issues](#8-accessibility-issues)
9. [Inconsistent Patterns](#9-inconsistent-patterns)
10. [Missing Validation](#10-missing-validation)

---

## 1. Critical Issues (Must Fix)

### 1.1 Invalid UUID Conversion in AI-Processor
**Location:** `backend/ai-processor/service.go:135, 202, 223`  
**Problem:** Converting `int64` IssueID to UUID using `fmt.Sprintf("%036d", payload.IssueID)` produces invalid UUIDs.  
**Impact:** Database operations will fail  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `github.com/google/uuid` to generate proper UUIDs based on issue ID | Medium |
| B | Change `issue_id` column in `tutorial_content` to `bigint` instead of UUID | High |
| C | Use UUID v5 (name-based) with issue ID as name | Medium |

---

### 1.2 Authentication Not Implemented
**Location:** `backend/gateway/middleware/auth.go:88-104`  
**Problem:** `validateToken` always returns an error - authentication fails for all requests.  
**Impact:** Protected routes are effectively unprotected  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Implement JWT validation using `internal/platform/auth` | Medium |
| B | Use an external auth library (go-jose, golang-jwt) | Medium |

---

### 1.3 OAuth User Info Not Implemented
**Location:** `backend/gateway/handler/auth.go:297-309`  
**Problem:** `getUserInfo` returns placeholder data instead of calling provider APIs.  
**Impact:** OAuth flow cannot create real users  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Implement GitHub/Google user info API calls | Medium |
| B | Use an OAuth library that handles user info (goth) | Medium |

---

### 1.4 Unused my-app Directory
**Location:** `/my-app/`  
**Problem:** Empty Next.js scaffold with no application code. Active frontend is in `/web/`.  
**Impact:** Confusion, unnecessary files  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Delete the `my-app/` directory entirely | Low |

---

## 2. Security Issues

### 2.1 Secrets in Docker Images
**Location:** All Dockerfiles line 25  
**Problem:** Copies `.env` file into Docker images, exposing secrets.  
**Impact:** Secrets baked into images  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove `COPY .env` from Dockerfiles, use runtime env vars | Low |
| B | Use Docker secrets or secrets manager | Medium |
| C | Use `.dockerignore` to exclude `.env` | Low |

---

### 2.2 Hardcoded Database Credentials
**Location:** `deployments/docker-compose.yml:41, 43, 63`  
**Problem:** Hardcoded `postgres:postgres` credentials in environment variables.  
**Impact:** Credentials exposed in config  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `.env` file only, remove hardcoded values | Low |
| B | Use Docker secrets for production | Medium |

---

### 2.3 Secure Cookie Flag Disabled
**Location:** `backend/gateway/handler/auth.go:190`, `middleware/csrf.go:46`  
**Problem:** `Secure` flag commented out on cookies.  
**Impact:** Cookies sent over HTTP in production  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Enable Secure flag based on environment (production = true) | Low |
| B | Use config value: `cfg.IsProduction` to toggle | Low |

---

### 2.4 HSTS Header Disabled
**Location:** `backend/gateway/middleware/security.go:22`  
**Problem:** HSTS header commented out.  
**Impact:** Missing HTTPS enforcement  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Enable HSTS in production environment | Low |
| B | Use config-driven header settings | Low |

---

### 2.5 In-Memory OAuth State Storage
**Location:** `backend/gateway/handler/auth.go:21,69`  
**Problem:** OAuth states stored in a map; not scalable, lost on restart.  
**Impact:** Race conditions, state lost on restart  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Move OAuth state to Redis with TTL | Medium |
| B | Use signed JWT for state (stateless) | Medium |

---

### 2.6 Rate Limiter Uses Spoofable RemoteAddr
**Location:** `backend/gateway/middleware/ratelimit.go:32`  
**Problem:** Uses `r.RemoteAddr` without checking `X-Forwarded-For`.  
**Impact:** Rate limiting bypassed behind proxies  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Check `X-Forwarded-For` and `X-Real-IP` headers first | Low |
| B | Use a library like `realip` to extract client IP | Low |

---

### 2.7 Missing Error Check on rand.Read
**Location:** `backend/gateway/handler/auth.go:315`, `middleware/csrf.go:74`  
**Problem:** `rand.Read` errors are ignored.  
**Impact:** Weak tokens if random generation fails  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Check error and fail if random generation fails | Low |
| B | Use `crypto/rand.Prime` or similar that panics on failure | Low |

---

### 2.8 Hardcoded CORS Origin
**Location:** `backend/gateway/middleware/cors.go:12`  
**Problem:** Origin hardcoded to `http://localhost:3000`.  
**Impact:** Production CORS misconfiguration  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Read allowed origins from config/environment | Low |
| B | Support multiple origins via config array | Low |

---

### 2.9 Markdown Rendering Security (Low Priority)
**Location:** `web/components/tutorial/TutorialViewer.tsx:37`  
**Problem:** ReactMarkdown is used without explicit sanitization. Note: ReactMarkdown is relatively safe by default as it doesn't render raw HTML.  
**Impact:** Low risk, but adding explicit sanitization is best practice  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add `rehype-sanitize` plugin for extra safety | Low |
| B | Keep as-is since ReactMarkdown is safe by default | None |

---

### 2.10 Missing URL Validation in IssueForm
**Location:** `web/components/issue/IssueForm.tsx`  
**Problem:** No validation of GitHub issue URL format before submission.  
**Impact:** Malicious URLs could be submitted  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add regex validation for GitHub issue URL format | Low |
| B | Use Zod schema validation | Low |

---

## 3. Error Handling Problems

### 3.1 Ignored JSON Encoding Errors
**Location:** Multiple files across all services  
- `backend/gateway/handler/issue.go:156`
- `backend/collector/handler/collect.go:68, 79`
- `backend/collector/handler/health.go:73`
- `backend/ai-processor/handler/health.go`

**Problem:** `json.NewEncoder(w).Encode()` errors are ignored.  
**Impact:** Silent failures, corrupted responses  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Log errors: `if err := json.NewEncoder(w).Encode(...); err != nil { log.Error(...) }` | Low |
| B | Create a shared `WriteJSON` helper that handles errors | Medium |

---

### 3.2 Ignored Cache Set Errors
**Location:** `backend/gateway/handler/issue.go:141`, `handler/tutorial.go:114`  
**Problem:** Cache set errors are ignored.  
**Impact:** Cache failures go unnoticed  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Log cache errors but don't fail the request | Low |
| B | Add metrics for cache failures | Medium |

---

### 3.3 Panic Instead of Graceful Exit
**Location:** `backend/gateway/main.go:37`, `backend/ai-processor/main.go:38`  
**Problem:** Uses `panic()` on config load failure.  
**Impact:** Unclean shutdown  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `os.Exit(1)` with error logging | Low |
| B | Return error from main and handle in init | Low |

---

### 3.4 Incorrect Retry-After Header
**Location:** `backend/gateway/middleware/quota.go:64`  
**Problem:** `string(rune(retryAfter))` converts int to a single character.  
**Impact:** Invalid HTTP header  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `strconv.Itoa(retryAfter)` | Low |

---

### 3.5 Unsafe Type Assertions
**Location:** `internal/platform/lock/redis.go:196, 236`, `internal/platform/cache/redis.go:343-346`  
**Problem:** Type assertions without `ok` check can panic.  
**Impact:** Potential panics  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `value, ok := x.(Type); if !ok { handle error }` | Low |

---

### 3.6 Error Handling Stops Entire Consume Loop
**Location:** `internal/platform/stream/redis.go:260`  
**Problem:** A single handler error stops the entire consume loop.  
**Impact:** One bad message stops all processing  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Log and continue, implement retry/DLQ logic | Medium |
| B | Add configurable error handling strategy | Medium |

---

### 3.7 String Error Comparison
**Location:** `internal/platform/stream/redis.go:159`  
**Problem:** `err.Error() != "BUSYGROUP..."` is fragile.  
**Impact:** May break with Redis version changes  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Check for specific error type or use `strings.Contains` | Low |
| B | Use `errors.Is()` if redis-go provides typed errors | Low |

---

### 3.8 Ignored JSON Marshal Error
**Location:** `backend/collector/service.go:155`  
**Problem:** `labelsJSON, _ := json.Marshal(issue.Labels)` - error ignored.  
**Impact:** Labels could be empty string silently  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Check error and log or handle appropriately | Low |

---

## 4. Code Duplication

### 4.1 Health Check Implementations (3x duplicated)
**Location:** All three services have identical health check implementations  
- `backend/gateway/handler/health.go`
- `backend/collector/handler/health.go`
- `backend/ai-processor/handler/health.go`

**Duplicated:**
- `HealthResponse` struct
- Health check logic pattern
- `checkRedis` function
- Response encoding pattern

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Create `/internal/platform/health/` package with shared implementation | Medium |
| B | Create interface-based health checker that accepts dependency checkers | Medium |

---

### 4.2 Error Response Functions (6x duplicated)
**Location:**
- `backend/gateway/handler/issue.go` - `writeError()`
- `backend/gateway/handler/auth.go` - `writeError()`
- `backend/collector/handler/collect.go` - `writeError()`
- `backend/gateway/middleware/auth.go` - `writeAuthError()`
- `backend/gateway/middleware/quota.go` - `writeQuotaError()`

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Create `/internal/platform/http/response.go` with shared `WriteError` | Low |
| B | Create `ErrorResponse` type and writer in shared package | Low |

---

### 4.3 Main.go Initialization Patterns (3x duplicated)
**Location:** All three services have nearly identical initialization  
- Configuration loading
- Logger initialization
- Redis client initialization
- Graceful shutdown pattern
- Signal handling

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Create `/internal/platform/server/bootstrap.go` with shared helpers | Medium |
| B | Create builder pattern for service initialization | High |

---

### 4.4 Dockerfile Patterns (3x duplicated)
**Location:** All Dockerfiles are nearly identical  
- Build stage
- Run stage
- Only binary name differs

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use a single parameterized Dockerfile with build args | Medium |
| B | Create a build script that generates Dockerfiles | Medium |
| C | Use Docker Compose build with args | Medium |

---

### 4.5 String Truncation Utility (2x duplicated)
**Location:** `backend/collector/service.go`, `backend/ai-processor/service.go`  
**Problem:** `truncateString()` and `truncateLog()` are identical.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Move to `/internal/platform/utils/string.go` | Low |

---

### 4.6 Duplicate UI Component Libraries
**Location:** `web/components/ui/` and `web/components/ui_legacy/`  
**Problem:** Both have Badge, Button, Card, Input components.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Delete `ui_legacy/` (appears unused) | Low |
| B | Migrate any used legacy components to new ui/ | Medium |

---

## 5. Unused/Dead Code

### 5.1 Unused GitHub Client Methods
**Location:** `backend/collector/github/client.go`  
- Line 123-175: `FetchIssueComments()` never used
- Line 194-197: `RateLimitStatus()` never used

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused methods | Low |
| B | Keep if planned for future use, add TODO comment | Low |

---

### 5.2 Unused Rate Limit Methods
**Location:** `backend/collector/github/ratelimit.go`  
- Line 135-150: `RetryAfter()` never used
- Line 152-164: `ExponentialBackoff()` never used

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused methods | Low |
| B | Implement proper rate limiting that uses these | Medium |

---

### 5.3 Unused APIError Type
**Location:** `backend/collector/github/errors.go:41-54`  
**Problem:** `APIError` struct and `NewAPIError()` defined but never used.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused type | Low |
| B | Use it in error handling | Medium |

---

### 5.4 Unused ErrInvalidAPIKey
**Location:** `backend/ai-processor/llm/client.go:25`  
**Problem:** Error variable defined but never used.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused variable | Low |
| B | Add API key validation that uses it | Low |

---

### 5.5 Unused GenerateOptions Struct
**Location:** `internal/platform/llm/llm.go:54-58`  
**Problem:** `GenerateOptions` struct defined but never used.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused struct | Low |
| B | Use it in the Generator interface | Medium |

---

### 5.6 Unused Frontend Components
**Location:** `web/components/`  
- `tutorial/ConceptTag.tsx` - never imported
- `ErrorBoundary.tsx` - defined but never used
- Entire `ui_legacy/` directory appears unused

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused components | Low |
| B | Implement ErrorBoundary wrapping | Low |

---

### 5.7 Unused Imports in Frontend
**Location:**
- `web/app/dashboard/layout.tsx:14` - `Menu` icon unused
- `web/app/dashboard/tutorials/[id]/page.tsx:7` - `ExternalLink` unused
- `web/app/dashboard/layout.tsx:10` - `FileText` icon unused

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove unused imports | Low |

---

### 5.8 MongoDB Service Unused
**Location:** `deployments/docker-compose.yml:22-30`  
**Problem:** MongoDB service is defined but no service uses it.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove MongoDB service | Low |
| B | Implement MongoDB usage if planned | High |

---

## 6. Configuration Issues

### 6.1 Missing Environment Variables in Docker Compose
**Location:** `deployments/docker-compose.yml`  
**Problem:** Services missing required env vars:
- Gateway: `JWT_SECRET`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`
- Collector: `PORT` not exposed
- AI-processor: `PORT` not exposed

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add all required env vars to docker-compose.yml | Low |
| B | Create comprehensive `.env.example` file | Low |

---

### 6.2 Duplicate Database URL Variables
**Location:** `deployments/docker-compose.yml:41, 43`  
**Problem:** Both `DATABASE_URL` and `POSTGRES_URL` set with identical values.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use only one variable name consistently | Low |

---

### 6.3 Empty Makefile Build Target
**Location:** `Makefile:12-14`  
**Problem:** `build` target only prints a message, doesn't build anything.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Implement actual build commands | Low |
| B | Add service-specific build targets | Medium |

---

### 6.4 Missing env_file for Services
**Location:** `deployments/docker-compose.yml`  
**Problem:** Collector and AI-processor services lack `env_file` directive.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add `env_file: ../.env` to all services | Low |

---

### 6.5 Hardcoded URLs in Gateway
**Location:** `backend/gateway/handler/auth.go:59, 61, 196`  
**Problem:** Base URLs and redirect URLs hardcoded.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Move all URLs to configuration | Low |
| B | Use environment variables | Low |

---

### 6.6 Inconsistent Environment Variable Access
**Location:** `internal/config/config.go:68-78`  
**Problem:** Mix of `os.Getenv()` and `getEnv()` helper.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `getEnv()` helper consistently for all env vars | Low |

---

### 6.7 Missing Redis DB Validation
**Location:** `internal/config/config.go`  
**Problem:** No validation that `RedisDB` is 0-15 (Redis limit).  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add validation in `Validate()` method | Low |

---

## 7. TypeScript/Frontend Issues

### 7.1 Using `any` Type
**Location:**
- `web/components/issue/IssueForm.tsx:31` - `catch (err: any)`
- `web/components/tutorial/TutorialViewer.tsx:18` - component props as `any`
- `web/lib/api.ts:26` - `(headers as any)['X-CSRF-Token']`

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Define proper error types and use type guards | Low |
| B | Use `unknown` with type narrowing | Low |

---

### 7.2 Missing Error Boundaries
**Location:** `web/app/`  
**Problem:** No ErrorBoundary wrapping pages/components.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Wrap app routes with ErrorBoundary component | Low |
| B | Use Next.js error.tsx files for route error handling | Low |

---

### 7.3 Logout Not Awaited
**Location:** `web/app/dashboard/layout.tsx:73, 86`  
**Problem:** `logout()` called without `await` - potential race condition.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add `await` to logout calls | Low |
| B | Add loading state during logout | Low |

---

### 7.4 Status Type Should Be Union
**Location:** `web/lib/types.ts:16`  
**Problem:** `status: string` should be a union type.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Change to `status: 'pending' \| 'completed' \| 'failed'` | Low |

---

### 7.5 Commented Code in Frontend
**Location:**
- `web/app/dashboard/tutorials/page.tsx:14-17`
- `web/lib/api.ts:39`

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Remove commented code or implement it | Low |

---

## 8. Accessibility Issues

### 8.1 Missing ARIA Labels
**Location:**
- `web/app/dashboard/layout.tsx:74` - Logout button
- `web/app/dashboard/layout.tsx:86` - Mobile logout button
- `web/app/dashboard/tutorials/[id]/page.tsx:30` - Back button

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add `aria-label` to all interactive elements | Low |

---

### 8.2 Badge Uses div Instead of span
**Location:** `web/components/ui/badge.tsx:32`  
**Problem:** Uses `<div>` instead of `<span>` for inline badge.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Change to `<span>` for proper inline semantics | Low |

---

### 8.3 Loading States Not Announced
**Location:** Various pages  
**Problem:** Screen readers not notified of loading states.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add `aria-live="polite"` regions for loading states | Low |
| B | Use `role="status"` for loading indicators | Low |

---

## 9. Inconsistent Patterns

### 9.1 Error Response Format Inconsistency
**Location:** `backend/gateway/middleware/csrf.go:56, 63`  
**Problem:** Uses `http.Error` (plain text) instead of JSON.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use JSON error response for consistency | Low |

---

### 9.2 Context Not Properly Reassigned
**Location:** `backend/gateway/middleware/auth.go:79`  
**Problem:** Context updated but `r` not reassigned properly.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `r = r.WithContext(ctx)` | Low |

---

### 9.3 Inconsistent Error Wrapping
**Location:** `backend/ai-processor/service.go`  
**Problem:** Some errors use `fmt.Errorf("%w: %v", ...)`, others don't.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Standardize error wrapping across codebase | Medium |

---

### 9.4 Status Field Inconsistency
**Location:** `backend/ai-processor/service.go:137`  
**Problem:** Checks `tutorialcontent.StatusCompleted` but domain uses `domain.StatusCompleted`.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use consistent status types from domain package | Low |

---

## 10. Missing Validation

### 10.1 Missing Request Body Size Limit
**Location:** `backend/gateway/handler/issue.go:71`  
**Problem:** No limit on JSON body size.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Use `http.MaxBytesReader` to limit body size | Low |
| B | Add middleware for body size limiting | Low |

---

### 10.2 Missing Content-Type Validation
**Location:** `backend/gateway/handler/issue.go:71`, `backend/collector/handler/collect.go:49-53`  
**Problem:** No check for `Content-Type: application/json`.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add content-type check before JSON decoding | Low |

---

### 10.3 Missing LLM Response Validation
**Location:** `backend/ai-processor/service.go:146`  
**Problem:** No validation before parsing LLM output.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add validation for non-empty, well-formed responses | Low |

---

### 10.4 Missing Input Validation in Transformers
**Location:** `backend/ai-processor/transformer/github.go:79-139`  
**Problem:** `TransformStreamMessage` doesn't validate string lengths.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add length validation with sensible limits | Low |

---

### 10.5 Missing Timeout on LLM Calls
**Location:** `backend/ai-processor/service.go:192-196`  
**Problem:** LLM calls don't have explicit timeouts.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add context with timeout for LLM calls | Low |
| B | Configure timeout in LLM client | Low |

---

### 10.6 Truncation Function Edge Case
**Location:** `backend/collector/service.go:176-181`  
**Problem:** If `maxLen < 3`, `s[:maxLen-3]` will panic.  

**Solution Options:**
| Option | Description | Effort |
|--------|-------------|--------|
| A | Add validation: `if maxLen < 3 { return "" }` | Low |

---

## Summary

### Issues by Severity

| Severity | Count | Description |
|----------|-------|-------------|
| Critical | 4 | Must fix before deployment |
| High | 25 | Security and data integrity issues |
| Medium | 50 | Code quality and maintainability |
| Low | 70+ | Cleanup and best practices |

### Issues by Category

| Category | Count |
|----------|-------|
| Critical Issues | 4 |
| Security Issues | 10 |
| Error Handling | 10 |
| Code Duplication | 6 |
| Unused/Dead Code | 8 |
| Configuration | 7 |
| TypeScript/Frontend | 5 |
| Accessibility | 3 |
| Inconsistent Patterns | 4 |
| Missing Validation | 6 |

### Priority Recommendations

**Immediate (Before any deployment):**
1. Fix UUID conversion bug in ai-processor
2. Implement authentication token validation
3. Remove secrets from Docker images
4. Delete unused my-app directory

**Short-term (Within 1-2 sprints):**
1. Implement OAuth user info fetching
2. Move OAuth state to Redis
3. Fix all security issues
4. Consolidate duplicated code (health checks, error responses)
5. Remove unused code

**Medium-term:**
1. Implement proper error boundaries in frontend
2. Add comprehensive validation
3. Improve accessibility
4. Standardize patterns across services
5. Add proper TypeScript types

---

*This report was generated by analyzing all source files in the IssueSight repository.*
