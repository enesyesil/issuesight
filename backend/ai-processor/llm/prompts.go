// Package llm - prompt templates for tutorial generation.
package llm

import (
	"fmt"
	"strings"

	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
)

// Prompt version for tracking changes
const PromptVersion = "v1.0.0"

// SystemPrompt provides context and instructions for the LLM.
const SystemPrompt = `You are an expert software engineer and technical writer. Your task is to analyze GitHub issues and create educational tutorials that teach software engineering concepts.

When analyzing an issue:
1. Identify the core problem being solved
2. Extract the key technical concepts involved
3. Explain the solution approach in a teaching-friendly way
4. Include relevant code examples when appropriate

Your tutorials should:
- Start with a clear, descriptive title using a markdown heading (# Title)
- Be well-structured with logical sections
- Use markdown formatting for readability
- Include code blocks with language specification when showing code
- Explain concepts from first principles when needed
- Be practical and actionable

Format your response as a complete markdown tutorial that could stand alone as an educational resource.`

// BuildUserPrompt creates the user prompt from stream payload.
// Uses StreamIssuePayload which contains both issue and repository data.
func BuildUserPrompt(payload *transformer.StreamIssuePayload) string {
	var sb strings.Builder

	sb.WriteString("Create a tutorial based on the following GitHub issue:\n\n")

	// Issue metadata
	sb.WriteString(fmt.Sprintf("**Repository:** %s\n", payload.FullName))
	if payload.RepoLanguage != "" {
		sb.WriteString(fmt.Sprintf("**Primary Language:** %s\n", payload.RepoLanguage))
	}
	if len(payload.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels:** %s\n", strings.Join(payload.Labels, ", ")))
	}
	sb.WriteString("\n")

	// Issue title and body
	sb.WriteString(fmt.Sprintf("## Issue Title\n%s\n\n", payload.Title))

	if payload.Body != "" {
		sb.WriteString("## Issue Description\n")
		sb.WriteString(truncateBody(payload.Body))
		sb.WriteString("\n\n")
	}

	// Reference
	if payload.HTMLURL != "" {
		sb.WriteString(fmt.Sprintf("**Source:** %s\n", payload.HTMLURL))
	}

	return sb.String()
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
