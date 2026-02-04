package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestIssueIDToUUID(t *testing.T) {
	tests := []struct {
		name    string
		issueID int64
	}{
		{"small issue ID", 1},
		{"medium issue ID", 12345},
		{"large issue ID", 1234567890},
		{"very large issue ID", 9223372036854775807}, // max int64
		{"zero", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := issueIDToUUID(tt.issueID)

			// Verify it's a valid UUID (not nil)
			if result == uuid.Nil {
				t.Errorf("issueIDToUUID(%d) returned nil UUID", tt.issueID)
			}

			// Verify the UUID string is valid format
			uuidStr := result.String()
			if len(uuidStr) != 36 {
				t.Errorf("issueIDToUUID(%d) returned invalid UUID string length: %d", tt.issueID, len(uuidStr))
			}

			// Verify the UUID can be parsed back
			parsed, err := uuid.Parse(uuidStr)
			if err != nil {
				t.Errorf("issueIDToUUID(%d) returned unparseable UUID: %v", tt.issueID, err)
			}
			if parsed != result {
				t.Errorf("issueIDToUUID(%d) UUID parse mismatch", tt.issueID)
			}
		})
	}
}

func TestIssueIDToUUID_Deterministic(t *testing.T) {
	// The same issue ID should always produce the same UUID
	issueID := int64(12345)

	uuid1 := issueIDToUUID(issueID)
	uuid2 := issueIDToUUID(issueID)
	uuid3 := issueIDToUUID(issueID)

	if uuid1 != uuid2 || uuid2 != uuid3 {
		t.Errorf("issueIDToUUID is not deterministic: got %s, %s, %s", uuid1, uuid2, uuid3)
	}
}

func TestIssueIDToUUID_Unique(t *testing.T) {
	// Different issue IDs should produce different UUIDs
	uuid1 := issueIDToUUID(1)
	uuid2 := issueIDToUUID(2)
	uuid3 := issueIDToUUID(12345)

	if uuid1 == uuid2 {
		t.Error("issueIDToUUID(1) == issueIDToUUID(2), expected different UUIDs")
	}
	if uuid2 == uuid3 {
		t.Error("issueIDToUUID(2) == issueIDToUUID(12345), expected different UUIDs")
	}
	if uuid1 == uuid3 {
		t.Error("issueIDToUUID(1) == issueIDToUUID(12345), expected different UUIDs")
	}
}

func TestTruncateLog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 4, "h..."},
		{"empty string", "", 10, ""},
		{"maxLen less than 4", "hello", 3, ""},
		{"maxLen equals 0", "hello", 0, ""},
		{"maxLen negative", "hello", -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateLog(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateLog(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestValidateLLMResponse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr error
	}{
		{
			name:    "valid response with heading",
			content: "# Tutorial Title\n\n" + strings.Repeat("This is content. ", 20),
			wantErr: nil,
		},
		{
			name:    "empty response",
			content: "",
			wantErr: ErrEmptyLLMResponse,
		},
		{
			name:    "whitespace only",
			content: "   \n\t\n   ",
			wantErr: ErrEmptyLLMResponse,
		},
		{
			name:    "too short",
			content: "# Title\nShort",
			wantErr: ErrResponseTooShort,
		},
		{
			name:    "no markdown heading",
			content: strings.Repeat("This is content without any heading markers. ", 10),
			wantErr: ErrResponseNoMarkdown,
		},
		{
			name:    "valid with h2 heading",
			content: "## Tutorial\n\n" + strings.Repeat("This is detailed content. ", 20),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLLMResponse(tt.content)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("validateLLMResponse() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("validateLLMResponse() error = nil, want %v", tt.wantErr)
				return
			}
			if !errors.Is(err, tt.wantErr) && !strings.Contains(err.Error(), tt.wantErr.Error()) {
				t.Errorf("validateLLMResponse() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
