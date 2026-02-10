# Security Best Practices Report

## Executive Summary
I ran a deeper security audit focused on auth, headers, CSRF, and SSRF. I found **one high‑severity server hardening issue** (missing HTTP header timeouts/limits on all three Go services), **one medium issue** (outbound OAuth calls without explicit HTTP client timeouts), and **two low‑severity hardening gaps** (CSP allows `unsafe-inline`; CSRF token is not bound to session). I did **not** find SSRF exposure paths: user‑supplied GitHub URLs are validated to `github.com` before any outbound calls.

---

## High Severity Findings

### [H-1] Missing `ReadHeaderTimeout` and `MaxHeaderBytes` on public HTTP servers (DoS risk)
- **Rule ID:** GO-HTTP-001
- **Severity:** High
- **Locations:**
  - `backend/gateway/server.go:112-118`
  - `backend/collector/main.go:148-154`
  - `backend/ai-processor/main.go:165-171`
- **Evidence:**
  - `backend/gateway/server.go:112-118` shows `http.Server{ ReadTimeout, WriteTimeout, IdleTimeout }` with no `ReadHeaderTimeout` or `MaxHeaderBytes`.
  - `backend/collector/main.go:148-154` shows the same pattern.
  - `backend/ai-processor/main.go:165-171` shows the same pattern.
- **Impact:** Attackers can use slowloris‑style header reads or oversized headers to exhaust server resources and degrade availability.
- **Fix:** Add `ReadHeaderTimeout` and `MaxHeaderBytes` to each server. Example (calibrate for your traffic):
  - `ReadHeaderTimeout: 5 * time.Second`
  - `MaxHeaderBytes: 1 << 20` (1 MB)
- **Mitigation:** If a reverse proxy enforces header timeouts/limits, it reduces exposure, but app‑level limits are still recommended.
- **False positive notes:** None. The fields are missing in the server structs.

---

## Medium Severity Findings

### [M-1] Outbound OAuth user‑info calls use `http.DefaultClient` without timeouts
- **Rule ID:** GO-HTTP-OUTBOUND-001
- **Severity:** Medium
- **Location:** `backend/gateway/handler/auth.go:343-456`
- **Evidence:**
  - `backend/gateway/handler/auth.go:352` uses `http.DefaultClient.Do(req)`
  - `backend/gateway/handler/auth.go:406` uses `http.DefaultClient.Do(req)`
  - `backend/gateway/handler/auth.go:456` uses `http.DefaultClient.Do(req)`
- **Impact:** A slow or hanging upstream (GitHub/Google) can tie up goroutines and connections indefinitely, causing resource exhaustion or partial outage.
- **Fix:** Use a dedicated `http.Client` with a reasonable timeout (e.g., 10s) and reuse it in the handler (or add context timeouts around these requests).
- **Mitigation:** Add per‑request context timeouts if you can’t share a client immediately.
- **False positive notes:** None. `http.DefaultClient` has no timeout by default.

---

## Low Severity Findings

### [L-1] CSP allows `unsafe-inline` scripts and styles
- **Rule ID:** GO-HEADERS-001
- **Severity:** Low
- **Location:** `backend/gateway/middleware/security.go:40-42`
- **Evidence:**
  - `Content-Security-Policy` includes `script-src 'self' 'unsafe-inline'` and `style-src 'self' 'unsafe-inline'`.
- **Impact:** `unsafe-inline` weakens CSP’s protection against XSS if HTML responses are ever served from the gateway.
- **Fix:** Remove `unsafe-inline` and use nonces/hashes for any inline scripts or styles. If the gateway only serves JSON, consider moving CSP to the frontend layer where it applies to HTML.
- **Mitigation:** Keep other XSS protections (escaping, input validation) and avoid HTML responses on the API server.
- **False positive notes:** If the gateway never serves HTML, this is defense‑in‑depth only.

### [L-2] CSRF token is not bound to session (double‑submit only)
- **Rule ID:** GO-CSRF-001
- **Severity:** Low
- **Location:** `backend/gateway/middleware/csrf.go:45-78`
- **Evidence:**
  - The CSRF middleware compares header token to cookie value only (double‑submit), with a note: `// For higher security, use HMAC binding to session`.
- **Impact:** If an attacker can set the CSRF cookie (e.g., via subdomain injection), they can forge a matching header token. This is a low‑likelihood but real weakness compared to session‑bound tokens.
- **Fix:** Bind the CSRF token to the user/session (e.g., HMAC or server‑side store) and verify server‑side, or adopt a standard CSRF library that supports session‑bound tokens.
- **Mitigation:** Keep `SameSite=Lax` (already present) and avoid exposing subdomains that can set cookies for the parent domain.
- **False positive notes:** If all auth uses bearer tokens and no cookies, CSRF is not applicable.

---

## Informational Notes
- **SSRF:** The only user‑supplied URL (`/api/issues`) is parsed and restricted to `github.com` (`backend/collector/parser/url.go:25-90`), and outbound calls use fixed provider URLs. No direct SSRF paths found in this scan.
- **Secrets:** No tracked secrets detected in the git index during this scan. Ensure local `.env` remains untracked and rotate any secrets if they were ever exposed.

