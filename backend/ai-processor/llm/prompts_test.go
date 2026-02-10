package llm

import (
	"strings"
	"testing"

	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
)

func TestBuildSystemPrompt_SubstitutesIssueURL(t *testing.T) {
	issueURL := "https://github.com/testowner/testrepo/issues/42"
	prompt := BuildSystemPrompt(issueURL)

	if strings.Contains(prompt, issueURLPlaceholder) {
		t.Fatalf("prompt should not contain placeholder %q", issueURLPlaceholder)
	}
	if !strings.Contains(prompt, "- GitHub Issue URL: "+issueURL) {
		t.Fatalf("prompt should contain injected issue URL")
	}
}

func TestBuildSystemPrompt_MissingIssueURLFallback(t *testing.T) {
	prompt := BuildSystemPrompt("")
	if !strings.Contains(prompt, "- GitHub Issue URL: "+missingIssueURL) {
		t.Fatalf("prompt should contain missing URL fallback")
	}
}

func TestBuildUserPrompt(t *testing.T) {
	payload := &transformer.StreamIssuePayload{
		IssueID:      12345,
		IssueNumber:  42,
		Owner:        "testowner",
		Repo:         "testrepo",
		FullName:     "testowner/testrepo",
		Title:        "Fix memory leak in connection pool",
		Body:         "We've noticed a memory leak when connections are not properly closed.",
		Labels:       []string{"bug", "performance"},
		State:        "open",
		HTMLURL:      "https://github.com/testowner/testrepo/issues/42",
		RepoLanguage: "Go",
		RepoStars:    500,
	}

	prompt := BuildUserPrompt(payload)

	tests := []struct {
		name     string
		contains string
	}{
		{"issue url", "https://github.com/testowner/testrepo/issues/42"},
		{"repository", "testowner/testrepo"},
		{"language", "Primary Language: Go"},
		{"labels", "bug, performance"},
		{"issue title", "Fix memory leak in connection pool"},
		{"issue body", "memory leak when connections"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(prompt, tt.contains) {
				t.Errorf("prompt should contain %q", tt.contains)
			}
		})
	}
}

func TestBuildUserPrompt_EmptyOptionalFields(t *testing.T) {
	payload := &transformer.StreamIssuePayload{
		IssueID: 12345,
		Owner:   "testowner",
		Repo:    "testrepo",
		Title:   "Test Issue",
	}

	prompt := BuildUserPrompt(payload)

	if !strings.Contains(prompt, "GitHub Issue URL: "+missingIssueURL) {
		t.Fatal("prompt should include missing URL fallback")
	}
	if !strings.Contains(prompt, "Issue Title: Test Issue") {
		t.Fatal("prompt should include issue title")
	}
	if !strings.Contains(prompt, "Repository: testowner/testrepo") {
		t.Fatal("prompt should include repository fallback")
	}
}

func TestBuildUserPrompt_NilPayload(t *testing.T) {
	if prompt := BuildUserPrompt(nil); prompt != "" {
		t.Fatalf("BuildUserPrompt(nil) should return empty string, got %q", prompt)
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldTrunc bool
	}{
		{
			name:        "short body",
			input:       "This is a short body.",
			shouldTrunc: false,
		},
		{
			name:        "long body",
			input:       string(make([]byte, maxBodyLength+1000)),
			shouldTrunc: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateBody(tt.input)
			if tt.shouldTrunc {
				if len(result) > maxBodyLength+50 {
					t.Errorf("body not truncated: %d chars", len(result))
				}
				if !strings.Contains(result, "truncated") {
					t.Error("truncated body should contain truncation notice")
				}
			} else if result != tt.input {
				t.Error("short body should not be modified")
			}
		})
	}
}

func TestSystemPromptNotEmpty(t *testing.T) {
	if SystemPrompt == "" {
		t.Error("SystemPrompt should not be empty")
	}
	if len(SystemPrompt) < 100 {
		t.Error("SystemPrompt seems too short")
	}
}

func TestSystemPrompt_RequiredHeadingsOrder(t *testing.T) {
	headings := []string{
		"## 0) Issue Understanding (Ground Truth)",
		"## 1) Before You Start: What You Must Know",
		"### 1.1 Concepts (repo + issue specific)",
		"### 1.2 Tooling you must be able to use",
		"## 2) Safety Rules (Non-Negotiable)",
		"## 3) Repo Recon Plan (Read-Only First)",
		"## 4) Reproduction + Baseline",
		"## 5) Implementation Plan (Codex-Drivable Steps)",
		"## 6) Testing Strategy",
		"## 7) PR Checklist (Maintainer-Friendly)",
		"## 8) Fallbacks + “If You Get Stuck”",
	}

	lastIdx := -1
	for _, heading := range headings {
		idx := strings.Index(SystemPrompt, heading)
		if idx == -1 {
			t.Fatalf("SystemPrompt missing required heading %q", heading)
		}
		if idx <= lastIdx {
			t.Fatalf("SystemPrompt heading %q appears out of order", heading)
		}
		lastIdx = idx
	}
}

func TestPromptVersion(t *testing.T) {
	if PromptVersion == "" {
		t.Error("PromptVersion should be set")
	}
}
