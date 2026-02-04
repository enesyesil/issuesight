package transformer

import (
	"strings"
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
		{
			name:      "json envelope with markdown",
			input:     `{"title":"JSON Title","prerequisites":["Go","Redis Streams"],"markdown":"# JSON Title\n\n## Prerequisites (Keywords)\n- Go\n- Redis Streams\n\n## Context\n- Example"}`,
			wantTitle: "JSON Title",
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

func TestParseLLMOutputWithPrereqs_JSON(t *testing.T) {
	input := `{"title":"JSON Title","prerequisites":["Go","Redis Streams"],"markdown":"# JSON Title\n\n## Prerequisites (Keywords)\n- Go\n- Redis Streams\n\n## Context\n- Example"}`

	tutorial, prereqs, err := ParseLLMOutputWithPrereqs(input)
	if err != nil {
		t.Fatalf("ParseLLMOutputWithPrereqs() error = %v", err)
	}
	if tutorial.Title != "JSON Title" {
		t.Fatalf("Title = %q, want %q", tutorial.Title, "JSON Title")
	}
	if len(prereqs) != 2 {
		t.Fatalf("Prerequisites length = %d, want 2", len(prereqs))
	}
	if prereqs[0] != "Go" || prereqs[1] != "Redis Streams" {
		t.Fatalf("Prerequisites = %v, want [Go Redis Streams]", prereqs)
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

func TestAutoFixTutorialMarkdown(t *testing.T) {
	input := `# Sample Tutorial

## Context
This is the context.

## Setup
Some setup steps.

## Step 1 — Do the thing
**Goal:** Do it.
**Why:** It matters.
**What to do:** Follow the steps.
**Checkpoint:** It works.`

	fixed := AutoFixTutorialMarkdown(input)

	requiredSections := []string{
		"## Context",
		"## Plan",
		"## Assumptions",
		"## Testing",
		"## Pitfalls",
		"## Summary",
	}

	for _, section := range requiredSections {
		if !strings.Contains(fixed, section) {
			t.Fatalf("AutoFixTutorialMarkdown missing section %q", section)
		}
	}

	if !strings.Contains(fixed, "## Step 2") || !strings.Contains(fixed, "## Step 3") {
		t.Fatalf("AutoFixTutorialMarkdown should add missing steps; got:\n%s", fixed)
	}
}

func TestValidateTutorialMarkdown(t *testing.T) {
	valid := `# Title

## Step 1 — First
**Goal:** A
**Why:** B
**What to do:** C
**Checkpoint:** D

## Step 2 — Second
**Goal:** A
**Why:** B
**What to do:** C
**Checkpoint:** D

## Step 3 — Third
**Goal:** A
**Why:** B
**What to do:** C
**Checkpoint:** D`

	if err := ValidateTutorialMarkdown(valid); err != nil {
		t.Fatalf("ValidateTutorialMarkdown unexpected error: %v", err)
	}

	invalid := `# Title

## Step 1 — First
**Goal:** A
**Why:** B
**What to do:** C
**Checkpoint:** D

## Step 2 — Second
**Goal:** A
**Why:** B
**What to do:** C`

	if err := ValidateTutorialMarkdown(invalid); err == nil {
		t.Fatalf("ValidateTutorialMarkdown expected error, got nil")
	}
}
