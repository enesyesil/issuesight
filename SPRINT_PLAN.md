# IssueSight Sprint Breakdown

This plan breaks the codebase review findings into atomic, committable tasks.
Each sprint yields a demoable build that can run, test, and serve as a base for the next sprint.

## Assumptions
- Backend services are run via `deployments/docker-compose.yml`.
- Authentication uses a JWT stored in a cookie set by gateway auth handlers.
- Validation steps use either tests (`go test`, `npm test`) or manual API/UI verification.

---

## Implementation Status

| Sprint | Status | Summary |
|--------|--------|---------|
| 1 | DONE | Cookie auth + route protection |
| 2 | DONE | User-scoped tutorials |
| 3 | DONE | Quota middleware + API |
| 4 | DONE | Issue dedupe with SetNX |
| 5 | DONE | Stream consumer resilience |
| 6 | DONE | Consistent error responses |
| 7 | DONE | Frontend auth guard + ErrorBoundary |
| 8 | DONE | Status enum alignment |

---

## Sprint 1: Auth Compatibility and Route Protection [DONE]

Goal: authenticated users can access protected APIs via cookie sessions; unauthenticated users are blocked.
Demo: login in the web app, submit an issue and list tutorials successfully; unauthenticated requests fail.

Tasks:
- T1.1 Add cookie-based auth support in gateway auth middleware.
  - Files: `backend/gateway/middleware/auth.go`, `backend/gateway/handler/auth.go` (cookie name source)
  - Validation: add tests in `backend/gateway/middleware/auth_test.go` to verify both `Authorization: Bearer` and cookie token paths; `go test ./backend/gateway/middleware/...`
- T1.2 Apply `RequireAuth` to `/api/issues`, `/api/tutorials`, `/api/tutorials/{id}`.
  - Files: `backend/gateway/server.go`
  - Validation: handler tests for 401 when no auth token and 200 with token; include CSRF header+cookie in POST tests; `go test ./backend/gateway/handler/...`
- T1.3 Add shared test helper for CSRF token setup to avoid false 403 failures.
  - Files: `backend/gateway/middleware/csrf_test.go` (helper) and update handler tests
  - Validation: tests that previously failed with CSRF now pass.

---

## Sprint 2: User-Scoped Tutorials (Plumbing and Access Control)

Goal: tutorial ownership is recorded and enforced end-to-end.
Demo: User A submits an issue and sees the tutorial; User B does not.

Tasks:
- T2.1 Carry `user_id` through the pipeline: gateway -> collector -> stream -> AI processor.
  - Files: `backend/gateway/handler/issue.go`, `backend/collector/handler/collect.go`,
    `backend/collector/service.go`, `backend/ai-processor/transformer/github.go`,
    `backend/ai-processor/service.go`
  - Validation: unit tests for stream payload parsing including `user_id`; `go test ./backend/ai-processor/transformer/...`
- T2.2 Create Tutorial access rows when tutorial content is created (original requester).
  - Files: `backend/ai-processor/service.go`, `internal/platform/db/ent/schema/tutorial.go`
  - Validation: unit/integration test that a Tutorial row exists for the `user_id` tied to the message.
- T2.3 Enforce user scoping in tutorial list and get endpoints.
  - Files: `backend/gateway/handler/tutorial.go`
  - Validation: handler tests to ensure only owner can list/get tutorials; cache tests ensure auth is checked before cache hit.
- T2.4 Make tutorial cache keys user-scoped or move cache read after ownership verification.
  - Files: `backend/gateway/handler/tutorial.go`
  - Validation: tests for cache-hit path still enforcing ownership.

---

## Sprint 3: Quota Model and Quota API

Goal: quota matches configured limits and is observable by clients.
Demo: user can view remaining quota and is blocked after exceeding daily limit.

Tasks:
- T3.1 Define and implement a per-day quota counter model aligned to `DefaultQuotaLimit`.
  - Files: `backend/gateway/middleware/quota.go`, `internal/domain/constants.go`
  - Validation: unit tests for counter increment, limit, and reset at day boundary (use miniredis).
- T3.2 Wire quota middleware for `POST /api/issues`.
  - Files: `backend/gateway/server.go`
  - Validation: handler test verifies 429 and `Retry-After` on exceed.
- T3.3 Add `GET /api/quota` endpoint returning remaining count and reset time.
  - Files: `backend/gateway/handler/quota.go` (new), `backend/gateway/server.go`
  - Validation: handler tests for success response shape and values.

---

## Sprint 4: Issue Submission Dedupe and Cache Key Fixes

Goal: duplicate submissions are rejected deterministically; cache keys are consistent.
Demo: two submissions of the same issue return a single processing pipeline.

Tasks:
- T4.1 Remove duplicate cache prefixing (fix key composition).
  - Files: `internal/domain/constants.go` or `internal/platform/cache/redis.go`
  - Validation: unit tests for cache key composition or handler tests that assert expected keys.
- T4.2 Use `SetNX` to mark an issue as processing before calling the collector.
  - Files: `backend/gateway/handler/issue.go`
  - Validation: handler tests with a fake cache: second request returns 409 (or 202 if desired behavior changes).

---

## Sprint 5: Stream Reliability and Recovery

Goal: stream consumer continues after errors and can recover pending messages.
Demo: malformed message doesn’t stop consumer; pending messages are reprocessed.

Tasks:
- T5.1 Modify stream consumer loop to continue on handler errors and log context.
  - Files: `internal/platform/stream/redis.go`
  - Validation: integration test using miniredis verifying loop continues after a handler error.
- T5.2 Add pending-claim processing on startup (XAUTOCLAIM or XREADGROUP `0`).
  - Files: `internal/platform/stream/redis.go`, `internal/platform/stream/stream.go`
  - Validation: test that pending message is reclaimed and processed.

---

## Sprint 6: Error Response Consistency

Goal: all API errors follow one JSON shape.
Demo: error responses across middleware and handlers are consistent.

Tasks:
- T6.1 Standardize handler errors to `internal/platform/http.WriteError`.
  - Files: `backend/gateway/handler/*.go`, `internal/platform/http/response.go`
  - Validation: handler tests assert `{"error": "...", "message": "..."}` across endpoints.
- T6.2 Standardize middleware errors to the same response shape.
  - Files: `backend/gateway/middleware/{auth,csrf,ratelimit,quota,recovery}.go`
  - Validation: middleware tests assert consistent error body and status.

---

## Sprint 7: Frontend Resilience and UX

Goal: UI handles auth expiry and runtime errors gracefully; quota visible to users.
Demo: expired session redirects to login; errors show the fallback UI; quota displays.

Tasks:
- T7.1 Enable 401 redirect in `web/lib/api.ts` (client-only).
  - Files: `web/lib/api.ts`
  - Validation: manual test: expire cookie and navigate to dashboard.
- T7.2 Wrap app or dashboard with `ErrorBoundary`.
  - Files: `web/app/layout.tsx` or `web/app/dashboard/layout.tsx`, `web/components/ErrorBoundary.tsx`
  - Validation: manual test: throw an error in a component and verify fallback UI.
- T7.3 Add dashboard auth guard in layout using `useAuth`.
  - Files: `web/app/dashboard/layout.tsx`, `web/hooks/useAuth.ts`
  - Validation: manual test: unauthenticated user is redirected to `/login`.
- T7.4 Display quota in UI (issue form or dashboard sidebar).
  - Files: `web/components/issue/IssueForm.tsx` or `web/app/dashboard/layout.tsx`
  - Validation: manual UI verification with `/api/quota` responses.

---

## Sprint 8: Status and Mapping Consistency

Goal: tutorial status values are consistent across domain, DB, and API DTOs.
Demo: status displayed in UI matches DB enum and API response without ad-hoc conversions.

Tasks:
- T8.1 Align domain `TutorialStatus` with Ent enum values (lowercase) or add mapping helpers.
  - Files: `internal/domain/tutorial.go`, `backend/ai-processor/service.go`, `backend/gateway/handler/tutorial.go`
  - Validation: unit tests for status conversions.
- T8.2 Introduce DTO mapping helpers for tutorial responses to avoid duplication.
  - Files: `backend/gateway/handler/tutorial.go` (or a new `backend/gateway/handler/dto.go`)
  - Validation: unit tests for DTO mapping.

---

## Review Prompt Used (Subagent)

Please review this sprint/task breakdown for completeness, atomicity, and testability. Identify missing tasks, ordering issues, and validation gaps, then suggest improvements.
