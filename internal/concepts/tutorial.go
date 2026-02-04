package concepts

import (
	"fmt"
	"strings"
)

// BuildTutorialMarkdown generates a beginner-friendly tutorial for a concept entry.
func BuildTutorialMarkdown(entry Entry) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = strings.TrimSpace(entry.Slug)
	}
	if name == "" {
		name = "This Concept"
	}

	desc := strings.TrimSpace(entry.Description)
	if desc == "" {
		desc = fmt.Sprintf("%s is a core software concept that helps engineers build reliable systems.", name)
	}

	category := strings.ToLower(strings.TrimSpace(entry.Category))
	step2Action := "Find a simple example in the docs or codebase."
	step2Checkpoint := "You can point to where it is used."
	step3Action := "Use it to guide a small change or decision in the repo."
	step3Checkpoint := "You can explain how it influenced your choice."

	switch category {
	case "project":
		step2Action = "Find a recent issue or PR that depends on this project."
		step2Checkpoint = "You can describe what the project solves."
		step3Action = "Identify a small contribution you could make."
		step3Checkpoint = "You can explain how to start contributing."
	case "language":
		step2Action = "Find a file that uses this language in the repo."
		step2Checkpoint = "You can recognize basic syntax patterns."
		step3Action = "Write a tiny change in this language (docs or code)."
		step3Checkpoint = "You can run or reason about the change."
	case "framework":
		step2Action = "Find a module or feature built with this framework."
		step2Checkpoint = "You can name a file that depends on it."
		step3Action = "Adjust a config or small component using the framework."
		step3Checkpoint = "You can explain the framework-specific behavior."
	case "database", "data":
		step2Action = "Find where data is modeled or queried."
		step2Checkpoint = "You can point to a schema or query."
		step3Action = "Identify a tiny data-related improvement."
		step3Checkpoint = "You can explain the data impact."
	case "cloud", "infra":
		step2Action = "Find a deployment or infrastructure config."
		step2Checkpoint = "You can describe the resource it defines."
		step3Action = "Propose a small infra improvement or cleanup."
		step3Checkpoint = "You can describe the expected impact."
	}

	return fmt.Sprintf(`# %s — Step-by-step

## Step 1 — Define it clearly
**Goal:** Explain %s in one sentence.
**Why:** Clear definitions prevent confusion while learning.
**What to do:**
- Use this description: %s
- Rewrite it in your own words.
**Checkpoint:** You can explain %s without notes.

## Step 2 — Find a real example
**Goal:** See it in action.
**Why:** Examples make abstract ideas concrete.
**What to do:**
- %s
**Checkpoint:** %s

## Step 3 — Apply it once
**Goal:** Use it in a small task.
**Why:** Practice makes the concept stick.
**What to do:**
- %s
**Checkpoint:** %s
`, name, name, desc, name, step2Action, step2Checkpoint, step3Action, step3Checkpoint)
}
