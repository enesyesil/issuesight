package llm

import (
	"testing"

	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
)

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

	// Check that key information is included
	tests := []struct {
		name     string
		contains string
	}{
		{"repository name", "testowner/testrepo"},
		{"language", "Go"},
		{"labels", "bug, performance"},
		{"issue title", "Fix memory leak in connection pool"},
		{"issue body", "memory leak when connections"},
		{"github url", "https://github.com/testowner/testrepo/issues/42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsString(prompt, tt.contains) {
				t.Errorf("Prompt should contain %q", tt.contains)
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
		// All optional fields empty
	}

	prompt := BuildUserPrompt(payload)

	// Should still work with minimal data
	if !containsString(prompt, "Test Issue") {
		t.Error("Prompt should contain the issue title")
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
				if len(result) > maxBodyLength+50 { // Allow some buffer for truncation message
					t.Errorf("Body not truncated: %d chars", len(result))
				}
				if !containsString(result, "truncated") {
					t.Error("Truncated body should contain truncation notice")
				}
			} else {
				if result != tt.input {
					t.Error("Short body should not be modified")
				}
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

func TestPromptVersion(t *testing.T) {
	if PromptVersion == "" {
		t.Error("PromptVersion should be set")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
