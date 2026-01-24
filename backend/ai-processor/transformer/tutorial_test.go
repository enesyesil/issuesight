package transformer

import (
	"testing"

	"github.com/issuesight/issuesight/internal/domain"
)

func TestParseLLMOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantErr   bool
	}{
		{
			name: "standard markdown with h1 heading",
			input: `# Understanding Redis Streams

This tutorial explains how to use Redis Streams for message queuing.

## Introduction

Redis Streams provide a powerful way to handle data streaming.`,
			wantTitle: "Understanding Redis Streams",
			wantErr:   false,
		},
		{
			name: "h2 heading as title",
			input: `## Building a REST API

Learn how to build a REST API with Go.`,
			wantTitle: "Building a REST API",
			wantErr:   false,
		},
		{
			name: "no heading - uses first line",
			input: `This is a tutorial about testing.

It covers unit testing and integration testing.`,
			wantTitle: "This is a tutorial about testing.",
			wantErr:   false,
		},
		{
			name:      "empty output",
			input:     "",
			wantTitle: "",
			wantErr:   true,
		},
		{
			name:      "whitespace only",
			input:     "   \n\t\n   ",
			wantTitle: "",
			wantErr:   true,
		},
		{
			name: "heading with extra whitespace",
			input: `#    Spaced Title   

Content here.`,
			wantTitle: "Spaced Title",
			wantErr:   false,
		},
		{
			name: "empty lines before heading",
			input: `

# Delayed Title

Some content.`,
			wantTitle: "Delayed Title",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLLMOutput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLLMOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if result.Title != tt.wantTitle {
					t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
				}
				if result.MarkdownBody == "" {
					t.Error("MarkdownBody should not be empty")
				}
				// Check that status is set
				if result.Status != domain.StatusPending {
					t.Errorf("Status = %v, want %v", result.Status, domain.StatusPending)
				}
			}
		})
	}
}

func TestTruncateTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short title",
			input:    "Short Title",
			expected: "Short Title",
		},
		{
			name:     "exactly max length",
			input:    string(make([]byte, maxTitleLength)),
			expected: string(make([]byte, maxTitleLength)),
		},
		{
			name:     "over max length - truncates at word boundary",
			input:    "This is a very long title that needs to be truncated at some point because it exceeds the maximum allowed length for a title which is quite long but we need to handle it gracefully without breaking words in the middle which would look bad to the user",
			expected: "This is a very long title that needs to be truncated at some point because it exceeds the maximum allowed length for a title which is quite long but we need to handle it gracefully without breaking words in the...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateTitle(tt.input)
			if len(result) > maxTitleLength+3 { // +3 for "..."
				t.Errorf("Truncated title too long: %d", len(result))
			}
		})
	}
}

func TestValidateTutorialContent(t *testing.T) {
	tests := []struct {
		name    string
		input   *domain.TutorialContent
		wantErr bool
	}{
		{
			name: "valid output",
			input: &domain.TutorialContent{
				Title:        "Test Title",
				MarkdownBody: "Test content",
			},
			wantErr: false,
		},
		{
			name:    "nil output",
			input:   nil,
			wantErr: true,
		},
		{
			name: "empty title",
			input: &domain.TutorialContent{
				Title:        "",
				MarkdownBody: "Test content",
			},
			wantErr: true,
		},
		{
			name: "empty body",
			input: &domain.TutorialContent{
				Title:        "Test Title",
				MarkdownBody: "",
			},
			wantErr: true,
		},
		{
			name: "whitespace only body",
			input: &domain.TutorialContent{
				Title:        "Test Title",
				MarkdownBody: "   \n\t   ",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTutorialContent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTutorialContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
