// Package transformer - tutorial output parsing.
package transformer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/issuesight/issuesight/internal/domain"
)

// Errors for tutorial parsing.
var (
	ErrEmptyOutput = errors.New("transformer: LLM output cannot be empty")
	ErrNoTitle     = errors.New("transformer: could not extract title from output")
	ErrEmptyBody   = errors.New("transformer: tutorial body is empty")

	ErrTutorialV2MissingSection = errors.New("transformer: tutorial v2 missing required section")
	ErrTutorialV2SectionOrder   = errors.New("transformer: tutorial v2 sections out of order")
	ErrTutorialV2MissingStep    = errors.New("transformer: tutorial v2 implementation plan must include at least one step")
)

// ParseLLMOutput extracts structured tutorial content from raw LLM response.
// Returns a domain.TutorialContent with Title and MarkdownBody populated.
//
// Title extraction strategy:
//  1. Look for first markdown heading (# Title)
//  2. Fall back to first non-empty line
//  3. Truncate to reasonable length if needed
func ParseLLMOutput(rawOutput string) (*domain.TutorialContent, error) {
	tutorial, _, err := ParseLLMOutputWithPrereqs(rawOutput)
	return tutorial, err
}

// ParseLLMOutputWithPrereqs extracts tutorial content plus prerequisite keywords (if present).
// Supports a JSON envelope format with a separate prerequisites field, and falls back to raw markdown.
func ParseLLMOutputWithPrereqs(rawOutput string) (*domain.TutorialContent, []string, error) {
	if strings.TrimSpace(rawOutput) == "" {
		return nil, nil, ErrEmptyOutput
	}

	// Clean up the output
	output := strings.TrimSpace(rawOutput)

	// Attempt to parse JSON envelope first (new contract)
	if parsed, ok := parseLLMJSON(output); ok {
		markdown := strings.TrimSpace(parsed.Markdown)
		if markdown == "" {
			return nil, nil, ErrEmptyBody
		}

		title := strings.TrimSpace(parsed.Title)
		if title == "" {
			title = extractTitle(markdown)
		}
		if title == "" {
			return nil, nil, ErrNoTitle
		}

		return &domain.TutorialContent{
			Title:        title,
			MarkdownBody: markdown,
			Status:       domain.StatusPending,
		}, normalizePrerequisites(parsed.PrerequisitesList()), nil
	}

	// Fall back to markdown-only parsing (legacy behavior)
	title := extractTitle(output)
	if title == "" {
		return nil, nil, ErrNoTitle
	}

	body := output
	if strings.TrimSpace(body) == "" {
		return nil, nil, ErrEmptyBody
	}

	return &domain.TutorialContent{
		Title:        title,
		MarkdownBody: body,
		Status:       domain.StatusPending, // Will be set to Completed after persistence
	}, normalizePrerequisites(extractPrerequisitesFromMarkdown(body)), nil
}

type llmJSONOutput struct {
	Title         string          `json:"title"`
	Markdown      string          `json:"markdown"`
	Prerequisites json.RawMessage `json:"prerequisites"`
}

// parseLLMJSON tries to parse a JSON envelope from the LLM output.
func parseLLMJSON(raw string) (*llmJSONOutput, bool) {
	candidates := []string{}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, false
	}

	// Prefer fenced JSON if present, otherwise try the raw output.
	if fenced := extractJSONFromFence(trimmed); fenced != "" {
		candidates = append(candidates, fenced)
	}
	candidates = append(candidates, trimmed)

	// As a last resort, try to extract the outermost JSON object.
	if obj := extractJSONObject(trimmed); obj != "" {
		candidates = append(candidates, obj)
	}

	for _, candidate := range candidates {
		var parsed llmJSONOutput
		if err := json.Unmarshal([]byte(candidate), &parsed); err == nil {
			if strings.TrimSpace(parsed.Markdown) != "" || strings.TrimSpace(parsed.Title) != "" {
				return &parsed, true
			}
		}
	}

	return nil, false
}

func extractJSONFromFence(raw string) string {
	start := strings.Index(raw, "```")
	if start == -1 {
		return ""
	}
	fence := raw[start+3:]
	newline := strings.Index(fence, "\n")
	if newline == -1 {
		return ""
	}
	// Optional language tag (e.g. "json")
	fence = fence[newline+1:]
	end := strings.Index(fence, "```")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(fence[:end])
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(raw[start : end+1])
}

func parsePrerequisites(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	// Standard case: JSON array of strings
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}

	// Fallback: single comma-separated string
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		parts := strings.Split(single, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts
	}

	return nil
}

func extractPrerequisitesFromMarkdown(markdown string) []string {
	lines := strings.Split(markdown, "\n")
	inSection := false
	sectionLevel := 0
	var items []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if level, title, ok := parseMarkdownHeading(trimmed); ok {
			titleLower := strings.ToLower(strings.TrimSpace(title))
			if isConceptSectionHeading(titleLower) {
				inSection = true
				sectionLevel = level
				continue
			}
			if inSection && level <= sectionLevel {
				break
			}
		}
		if !inSection || trimmed == "" {
			continue
		}

		if concept, ok := extractConceptField(trimmed); ok {
			items = append(items, concept)
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "- "):
			items = append(items, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		case strings.HasPrefix(trimmed, "* "):
			items = append(items, strings.TrimSpace(strings.TrimPrefix(trimmed, "* ")))
		default:
			// Simple numeric list like "1. Item"
			if idx := strings.Index(trimmed, ". "); idx > 0 && idx < 4 {
				items = append(items, strings.TrimSpace(trimmed[idx+2:]))
			}
		}
	}

	return items
}

func parseMarkdownHeading(line string) (int, string, bool) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false
	}

	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}

	title := strings.TrimSpace(line[level+1:])
	if title == "" {
		return 0, "", false
	}

	return level, title, true
}

func isConceptSectionHeading(titleLower string) bool {
	return strings.Contains(titleLower, "1.1 concepts") ||
		strings.Contains(titleLower, "prereq") ||
		strings.Contains(titleLower, "concepts") ||
		strings.Contains(titleLower, "key concepts")
}

func extractConceptField(line string) (string, bool) {
	item, ok := trimListPrefix(line)
	if !ok {
		return "", false
	}

	raw := strings.TrimSpace(item)
	lower := strings.ToLower(raw)
	prefixes := []string{
		"**concept:**",
		"**concept**:",
		"concept:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			value := strings.TrimSpace(raw[len(prefix):])
			if value != "" {
				return value, true
			}
		}
	}

	return "", false
}

func trimListPrefix(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "- "):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")), true
	case strings.HasPrefix(trimmed, "* "):
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "* ")), true
	default:
		if idx := strings.Index(trimmed, ". "); idx > 0 && idx < 4 {
			return strings.TrimSpace(trimmed[idx+2:]), true
		}
	}
	return "", false
}

func normalizePrerequisites(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}

	const maxItems = 20
	const maxLen = 80

	seen := make(map[string]bool)
	var out []string

	for _, item := range raw {
		clean := strings.TrimSpace(item)
		clean = strings.TrimPrefix(clean, "- ")
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		if len(clean) > maxLen {
			clean = clean[:maxLen]
		}
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
		if len(out) >= maxItems {
			break
		}
	}

	return out
}

func (o *llmJSONOutput) PrerequisitesList() []string {
	return parsePrerequisites(o.Prerequisites)
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

// AutoFixTutorialMarkdown enforces a consistent tutorial structure for issue tutorials.
// It ensures required sections exist, steps are labeled, and each step includes Goal/Why/What/Checkpoint.
func AutoFixTutorialMarkdown(markdown string) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return markdown
	}

	titleLine, body := extractTitleLine(trimmed)
	intro, blocks := splitMarkdownBlocks(body)

	requiredSections := []string{"Context", "Plan", "Assumptions", "Testing", "Pitfalls", "Summary"}

	sections := make(map[string]mdBlock)
	var prereqBlocks []mdBlock
	var stepBlocks []mdBlock
	var stepCandidates []mdBlock

	for _, block := range blocks {
		titleLower := strings.ToLower(strings.TrimSpace(block.title))
		if isPrereqHeading(titleLower) {
			prereqBlocks = append(prereqBlocks, block)
			continue
		}
		if canonical := matchRequiredSection(titleLower); canonical != "" {
			sections[canonical] = block
			continue
		}
		if strings.HasPrefix(titleLower, "step") {
			stepBlocks = append(stepBlocks, block)
			continue
		}
		stepCandidates = append(stepCandidates, block)
	}

	if len(stepBlocks) == 0 && len(stepCandidates) > 0 {
		stepBlocks = append(stepBlocks, stepCandidates...)
		stepCandidates = nil
	} else if len(stepCandidates) > 0 {
		stepBlocks = append(stepBlocks, stepCandidates...)
		stepCandidates = nil
	}

	for len(stepBlocks) < 3 {
		stepBlocks = append(stepBlocks, mdBlock{
			title:   "Step",
			content: "",
		})
	}

	builder := strings.Builder{}
	if titleLine != "" {
		builder.WriteString(titleLine)
		builder.WriteString("\n\n")
	}
	if intro != "" {
		builder.WriteString(intro)
		builder.WriteString("\n\n")
	}

	for _, block := range prereqBlocks {
		builder.WriteString("## ")
		builder.WriteString(block.title)
		builder.WriteString("\n")
		if block.content != "" {
			builder.WriteString(block.content)
		} else {
			builder.WriteString("- Add prerequisites here\n")
		}
		builder.WriteString("\n\n")
	}

	for _, section := range requiredSections {
		if block, ok := sections[section]; ok {
			builder.WriteString("## ")
			builder.WriteString(section)
			builder.WriteString("\n")
			if block.content != "" {
				builder.WriteString(block.content)
			} else {
				builder.WriteString(sectionPlaceholder(section))
			}
			builder.WriteString("\n\n")
			continue
		}
		builder.WriteString("## ")
		builder.WriteString(section)
		builder.WriteString("\n")
		builder.WriteString(sectionPlaceholder(section))
		builder.WriteString("\n\n")
	}

	for i, block := range stepBlocks {
		stepTitle := strings.TrimSpace(block.title)
		stepTitle = cleanStepTitle(stepTitle)
		if stepTitle == "" {
			stepTitle = "Complete the task"
		}
		builder.WriteString("## Step ")
		builder.WriteString(intToString(i + 1))
		builder.WriteString(" — ")
		builder.WriteString(stepTitle)
		builder.WriteString("\n")
		builder.WriteString(ensureStepFields(strings.TrimSpace(block.content)))
		builder.WriteString("\n\n")
	}

	return strings.TrimSpace(builder.String())
}

// ValidateTutorialMarkdown checks if the markdown has the expected structure.
func ValidateTutorialMarkdown(markdown string) error {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ErrEmptyBody
	}

	_, body := extractTitleLine(trimmed)
	_, blocks := splitMarkdownBlocks(body)
	stepCount := 0

	for _, block := range blocks {
		titleLower := strings.ToLower(strings.TrimSpace(block.title))
		if strings.HasPrefix(titleLower, "step") {
			stepCount++
			if !stepHasRequiredFields(block.content) {
				return errors.New("transformer: tutorial step missing required fields")
			}
		}
	}

	if stepCount < 3 {
		return errors.New("transformer: tutorial requires at least 3 steps")
	}

	return nil
}

var tutorialV2SectionSequence = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}

// ValidateTutorialMarkdownV2 validates the numbered tutorial format from prompt template v2.
func ValidateTutorialMarkdownV2(markdown string) error {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ErrEmptyBody
	}

	lines := strings.Split(trimmed, "\n")
	expectedIdx := 0
	inSection5 := false
	hasStepInSection5 := false

	for _, line := range lines {
		level, title, ok := parseMarkdownHeading(strings.TrimSpace(line))
		if !ok {
			continue
		}

		if level == 3 && inSection5 {
			titleLower := strings.ToLower(strings.TrimSpace(title))
			if strings.HasPrefix(titleLower, "step ") {
				hasStepInSection5 = true
			}
			continue
		}

		if level != 2 {
			continue
		}

		if inSection5 && !hasStepInSection5 {
			return ErrTutorialV2MissingStep
		}
		inSection5 = false

		if expectedIdx >= len(tutorialV2SectionSequence) {
			return fmt.Errorf("%w: unexpected extra section %q", ErrTutorialV2SectionOrder, title)
		}

		number, ok := parseTutorialSectionNumber(title)
		if !ok {
			return fmt.Errorf("%w: section heading %q", ErrTutorialV2SectionOrder, title)
		}

		expected := tutorialV2SectionSequence[expectedIdx]
		if number != expected {
			return fmt.Errorf("%w: expected %d) got %d)", ErrTutorialV2SectionOrder, expected, number)
		}

		if number == 5 {
			inSection5 = true
			hasStepInSection5 = false
		}

		expectedIdx++
	}

	if expectedIdx < len(tutorialV2SectionSequence) {
		return fmt.Errorf("%w: expected section %d)", ErrTutorialV2MissingSection, tutorialV2SectionSequence[expectedIdx])
	}

	if inSection5 && !hasStepInSection5 {
		return ErrTutorialV2MissingStep
	}

	return nil
}

func parseTutorialSectionNumber(title string) (int, bool) {
	closingIdx := strings.Index(title, ")")
	if closingIdx <= 0 {
		return 0, false
	}

	numberPart := strings.TrimSpace(title[:closingIdx])
	if numberPart == "" {
		return 0, false
	}

	n, err := strconv.Atoi(numberPart)
	if err != nil {
		return 0, false
	}
	return n, true
}

type mdBlock struct {
	title   string
	content string
}

func extractTitleLine(markdown string) (string, string) {
	lines := strings.Split(markdown, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			rest := strings.Join(lines[i+1:], "\n")
			return trimmed, strings.TrimLeft(rest, "\n")
		}
		break
	}
	return "", markdown
}

func splitMarkdownBlocks(markdown string) (string, []mdBlock) {
	lines := strings.Split(markdown, "\n")
	var blocks []mdBlock
	var current mdBlock
	var introLines []string
	var currentLines []string
	inBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inBlock {
				current.content = strings.TrimSpace(strings.Join(currentLines, "\n"))
				blocks = append(blocks, current)
				currentLines = nil
			}
			inBlock = true
			current = mdBlock{title: strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))}
			continue
		}
		if inBlock {
			currentLines = append(currentLines, line)
		} else {
			introLines = append(introLines, line)
		}
	}

	if inBlock {
		current.content = strings.TrimSpace(strings.Join(currentLines, "\n"))
		blocks = append(blocks, current)
	}

	intro := strings.TrimSpace(strings.Join(introLines, "\n"))
	return intro, blocks
}

func isPrereqHeading(titleLower string) bool {
	return strings.Contains(titleLower, "prereq") || strings.Contains(titleLower, "concept")
}

func matchRequiredSection(titleLower string) string {
	switch {
	case strings.HasPrefix(titleLower, "context"):
		return "Context"
	case strings.HasPrefix(titleLower, "plan"):
		return "Plan"
	case strings.HasPrefix(titleLower, "assumptions"):
		return "Assumptions"
	case strings.HasPrefix(titleLower, "testing"):
		return "Testing"
	case strings.HasPrefix(titleLower, "pitfalls"):
		return "Pitfalls"
	case strings.HasPrefix(titleLower, "summary"):
		return "Summary"
	}
	return ""
}

func cleanStepTitle(title string) string {
	if title == "" {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(title))
	if strings.HasPrefix(lower, "step") {
		parts := strings.SplitN(title, "—", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
		parts = strings.SplitN(title, "-", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(title)
}

func stepHasRequiredFields(content string) bool {
	lower := strings.ToLower(content)
	hasGoal := strings.Contains(lower, "goal:")
	hasWhy := strings.Contains(lower, "why:")
	hasCheckpoint := strings.Contains(lower, "checkpoint:")
	hasWhat := strings.Contains(lower, "what to do:") || strings.Contains(lower, "what to change:")
	return hasGoal && hasWhy && hasCheckpoint && hasWhat
}

func ensureStepFields(content string) string {
	text := strings.TrimSpace(content)
	missing := []string{}
	lower := strings.ToLower(text)

	if !strings.Contains(lower, "goal:") {
		missing = append(missing, "Goal")
	}
	if !strings.Contains(lower, "why:") {
		missing = append(missing, "Why")
	}
	if !strings.Contains(lower, "what to do:") && !strings.Contains(lower, "what to change:") {
		missing = append(missing, "What to do")
	}
	if !strings.Contains(lower, "checkpoint:") {
		missing = append(missing, "Checkpoint")
	}

	if text == "" {
		text = "**Goal:** Clarify the outcome for this step.\n**Why:** Explain why this step matters.\n**What to do:** List the concrete actions you will take.\n**Checkpoint:** State how you will verify completion."
		return text
	}

	if len(missing) == 0 {
		return text
	}

	builder := strings.Builder{}
	builder.WriteString(text)
	if !strings.HasSuffix(text, "\n") {
		builder.WriteString("\n")
	}
	for _, label := range missing {
		builder.WriteString("**")
		builder.WriteString(label)
		builder.WriteString(":** ")
		switch label {
		case "Goal":
			builder.WriteString("Clarify the outcome for this step.")
		case "Why":
			builder.WriteString("Explain why this step matters.")
		case "What to do":
			builder.WriteString("List the concrete actions you will take.")
		case "Checkpoint":
			builder.WriteString("State how you will verify completion.")
		}
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func sectionPlaceholder(section string) string {
	switch section {
	case "Context":
		return "Provide a brief overview of the issue and why it matters."
	case "Plan":
		return "- Outline the approach you will take.\n- Call out key files or components."
	case "Assumptions":
		return "- Note any assumptions or constraints you are relying on."
	case "Testing":
		return "- List the checks or tests you will run."
	case "Pitfalls":
		return "- Highlight common mistakes or edge cases to avoid."
	case "Summary":
		return "- Recap the key steps and expected outcome."
	default:
		return "Add notes here."
	}
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
