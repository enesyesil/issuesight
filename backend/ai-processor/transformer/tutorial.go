// Package transformer - tutorial output parsing.
package transformer

import (
	"errors"
	"strings"

	"github.com/issuesight/issuesight/internal/domain"
)

// Errors for tutorial parsing.
var (
	ErrEmptyOutput = errors.New("transformer: LLM output cannot be empty")
	ErrNoTitle     = errors.New("transformer: could not extract title from output")
	ErrEmptyBody   = errors.New("transformer: tutorial body is empty")
)

// ParseLLMOutput extracts structured tutorial content from raw LLM response.
// Returns a domain.TutorialContent with Title and MarkdownBody populated.
//
// Title extraction strategy:
//  1. Look for first markdown heading (# Title)
//  2. Fall back to first non-empty line
//  3. Truncate to reasonable length if needed
func ParseLLMOutput(rawOutput string) (*domain.TutorialContent, error) {
	if strings.TrimSpace(rawOutput) == "" {
		return nil, ErrEmptyOutput
	}

	// Clean up the output
	output := strings.TrimSpace(rawOutput)

	// Extract title
	title := extractTitle(output)
	if title == "" {
		return nil, ErrNoTitle
	}

	// The body is the full content (including the title heading)
	body := output

	if strings.TrimSpace(body) == "" {
		return nil, ErrEmptyBody
	}

	return &domain.TutorialContent{
		Title:        title,
		MarkdownBody: body,
		Status:       domain.StatusPending, // Will be set to Completed after persistence
	}, nil
}

// extractTitle attempts to extract a title from the markdown content.
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for markdown heading (# Title)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			return truncateTitle(strings.TrimSpace(title))
		}

		// Check for ## heading as fallback
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimPrefix(line, "## ")
			return truncateTitle(strings.TrimSpace(title))
		}

		// If first line is not a heading, use it as title
		return truncateTitle(line)
	}

	return ""
}

// truncateTitle ensures title is within reasonable length.
const maxTitleLength = 200

func truncateTitle(title string) string {
	if len(title) <= maxTitleLength {
		return title
	}

	// Try to truncate at a word boundary
	truncated := title[:maxTitleLength]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxTitleLength/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}

// ValidateTutorialContent performs validation on the parsed content.
func ValidateTutorialContent(t *domain.TutorialContent) error {
	if t == nil {
		return ErrEmptyOutput
	}
	if t.Title == "" {
		return ErrNoTitle
	}
	if strings.TrimSpace(t.MarkdownBody) == "" {
		return ErrEmptyBody
	}
	return nil
}
