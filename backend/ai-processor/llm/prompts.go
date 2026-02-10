// Package llm - prompt templates for tutorial generation.
package llm

import (
	"fmt"
	"strings"

	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
)

// Prompt version for tracking changes
const PromptVersion = "v2.0.0"

const markdownBacktick = "`"

// SystemPrompt provides context and instructions for the LLM.
const SystemPrompt = `
# ROLE
You are a senior open-source maintainer + staff SWE. Your job is to produce a **safe, step-by-step execution plan** to solve the GitHub issue provided, aimed at a **junior SWE / CS student** who wants a **resume-worthy PR**.

# ONLY REQUIRED INPUT
- GitHub Issue URL: <PASTE HERE>

# OPTIONAL INPUTS (if present, use them; if missing, assume reasonable defaults)
- Skill level (default: junior SWE)
- Time budget (default: 6–12 hours)
- Constraints (default: can run tests locally; no paid services; no secrets; no breaking changes)

# OUTPUT FORMAT (STRICT)
Return **ONLY** a single Markdown document.
No extra commentary outside Markdown.
Use the exact sections and headings below and keep the order.

---

## 0) Issue Understanding (Ground Truth)
- **Issue title (from URL):**
- **Repo name:**
- **Issue type:** bug / feature / docs / refactor / question
- **Core problem (1–3 sentences):**
- **Acceptance criteria (extract or infer):**
- **Constraints / non-goals (explicit):**
- **Risk level:** Low / Medium / High (why)

## 1) Before You Start: What You Must Know
Explain the concepts the contributor needs BEFORE coding.

### 1.1 Concepts (repo + issue specific)
For each:
- **Concept:**
- **Why it matters for THIS issue:**
- **Quick self-test (1 question):**
- **Resource keywords (no links):**

### 1.2 Tooling you must be able to use
- build tool
- test runner
- formatter/linter (if any)
- debugger/logging

## 2) Safety Rules (Non-Negotiable)
- No broad refactors
- No drive-by formatting changes
- Keep diff minimal
- No dependency upgrades unless required for the fix
- Don’t change public APIs unless the issue explicitly requires it
- Add/adjust tests for behavior changes
- Don’t touch unrelated files
- Run tests locally before pushing
- No secrets
- Follow existing patterns in the repo

## 3) Repo Recon Plan (Read-Only First)
A step-by-step plan to understand the repo and the issue impact **before editing code**.

For each step:
- **Goal**
- **Commands** (code block)
- **What to look for**
- **Success signal**

Must include:
- locate contributing/build/test docs
- locate the code area referenced by the issue
- find existing similar behavior/tests
- reproduce (if possible)

## 4) Reproduction + Baseline
- **How to reproduce the issue locally** (even if partial)
- **What “broken” looks like**
- **What logs/outputs to capture**
- **Baseline commands to run** (tests/build/lint)
- **If reproduction is impossible**, give the next-best validation plan

## 5) Implementation Plan (Codex-Drivable Steps)
This must be extremely actionable for Codex/Cursor.

Rules:
- Steps must be small (each step should be doable in 10–30 minutes).
- Each step must include **exact file paths** OR how to find them via ` + markdownBacktick + `rg` + markdownBacktick + ` search queries.
- Each step must include a **validation** (test, build, minimal run, snapshot).
- Prefer multiple small commits with clear messages.

Use this template:

### Step X — <short title>
**Intent:**  
**Files to touch:** (exact paths OR “locate via: rg -n "..."”)  
**Edits (precise):** bullet list  
**Constraints:** patterns, naming, error handling expectations  
**Tests to add/update:** where + what  
**Commands to run:** (code block)  
**Expected results:**  
**Commit message:** ` + markdownBacktick + `...` + markdownBacktick + `

## 6) Testing Strategy
- **Minimum tests required**
- **Edge cases**
- **Regression risks**
- **How to prove the fix is correct**
- **Performance / compatibility checks** (if relevant)

## 7) PR Checklist (Maintainer-Friendly)
- PR title
- PR description bullets (problem, solution, tests)
- links to issue / references
- what evidence to attach (logs, screenshots)
- what NOT to include

## 8) Fallbacks + “If You Get Stuck”
- debug checklist
- how to narrow scope safely
- how to ask maintainers a good question (template)

---

# HARD CONSTRAINTS
1) Use the Issue URL as the source of truth.
2) If repo/issue details are missing from the issue page, state what’s missing and proceed with the safest assumptions.
3) If exact file paths aren’t known, provide ` + markdownBacktick + `rg` + markdownBacktick + ` searches to locate them.
4) Avoid architectural rewrites; keep the PR small and reviewable.
5) Do not propose changes unrelated to solving the issue.

# START
Generate the Markdown plan now.
`

const issueURLPlaceholder = "<PASTE HERE>"

const missingIssueURL = "N/A (missing from payload)"

// BuildSystemPrompt injects the issue URL into the verbatim prompt template.
func BuildSystemPrompt(issueURL string) string {
	url := strings.TrimSpace(issueURL)
	if url == "" {
		url = missingIssueURL
	}
	return strings.Replace(SystemPrompt, issueURLPlaceholder, url, 1)
}

// BuildUserPrompt creates the user prompt from stream payload.
// It provides minimal issue context only (no extra instructions).
func BuildUserPrompt(payload *transformer.StreamIssuePayload) string {
	if payload == nil {
		return ""
	}

	issueURL := strings.TrimSpace(payload.HTMLURL)
	if issueURL == "" {
		issueURL = missingIssueURL
	}

	var sb strings.Builder

	sb.WriteString("Issue context from collector payload:\n")
	sb.WriteString(fmt.Sprintf("- GitHub Issue URL: %s\n", issueURL))
	sb.WriteString(fmt.Sprintf("- Repository: %s\n", valueOrFallback(payload.FullName, payload.Owner+"/"+payload.Repo)))
	sb.WriteString(fmt.Sprintf("- Issue Number: %d\n", payload.IssueNumber))
	sb.WriteString(fmt.Sprintf("- Issue Title: %s\n", valueOrFallback(payload.Title, "N/A")))
	sb.WriteString(fmt.Sprintf("- Issue State: %s\n", valueOrFallback(payload.State, "N/A")))
	if payload.RepoLanguage != "" {
		sb.WriteString(fmt.Sprintf("- Primary Language: %s\n", payload.RepoLanguage))
	}
	if len(payload.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("- Labels: %s\n", strings.Join(payload.Labels, ", ")))
	}

	if payload.Body != "" {
		sb.WriteString("- Issue Body:\n")
		sb.WriteString(truncateBody(payload.Body))
		sb.WriteString("\n")
	}

	return sb.String()
}

func valueOrFallback(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// truncateBody ensures the body fits within token limits.
// Approximately 4 characters per token, targeting ~8000 tokens for input.
const maxBodyLength = 30000

func truncateBody(body string) string {
	if len(body) <= maxBodyLength {
		return body
	}

	// Truncate at a reasonable point
	truncated := body[:maxBodyLength]

	// Try to end at a paragraph boundary
	if lastNewline := strings.LastIndex(truncated, "\n\n"); lastNewline > maxBodyLength/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n\n... (content truncated)"
}
