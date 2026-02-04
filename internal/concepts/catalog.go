package concepts

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
)

//go:embed catalog.json
var catalogJSON []byte

const DefaultMaxResults = 6

// Entry represents a canonical concept definition.
type Entry struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Aliases     []string `json:"aliases"`
	Triggers    []string `json:"triggers"`
}

// Signals represent deterministic inputs for concept matching.
type Signals struct {
	Language string
	Title    string
	Body     string
	Labels   []string
}

// Catalog holds concept entries and precomputed matchers.
type Catalog struct {
	Entries    []Entry
	matchers   map[string][]*regexp.Regexp
	aliasIndex map[string]string
}

var (
	defaultCatalog *Catalog
	onceCatalog    sync.Once
)

// DefaultCatalog returns a cached catalog instance.
func DefaultCatalog() (*Catalog, error) {
	var err error
	onceCatalog.Do(func() {
		defaultCatalog, err = LoadCatalog()
	})
	return defaultCatalog, err
}

// LoadCatalog loads and validates the catalog from the embedded JSON file.
func LoadCatalog() (*Catalog, error) {
	var entries []Entry
	if err := json.Unmarshal(catalogJSON, &entries); err != nil {
		return nil, fmt.Errorf("concepts: parse catalog: %w", err)
	}

	catalog := &Catalog{
		Entries:    entries,
		matchers:   make(map[string][]*regexp.Regexp),
		aliasIndex: make(map[string]string),
	}

	if err := catalog.validate(); err != nil {
		return nil, err
	}
	catalog.buildMatchers()
	return catalog, nil
}

func (c *Catalog) validate() error {
	seen := make(map[string]bool)
	for _, entry := range c.Entries {
		if entry.Slug == "" || entry.Name == "" {
			return fmt.Errorf("concepts: invalid entry: missing slug or name")
		}
		if seen[entry.Slug] {
			return fmt.Errorf("concepts: duplicate slug: %s", entry.Slug)
		}
		seen[entry.Slug] = true
	}
	return nil
}

func (c *Catalog) buildMatchers() {
	for _, entry := range c.Entries {
		var patterns []string
		patterns = append(patterns, entry.Name)
		patterns = append(patterns, entry.Slug)
		patterns = append(patterns, entry.Aliases...)
		patterns = append(patterns, entry.Triggers...)

		for _, p := range patterns {
			key := normalizeKeyword(p)
			if key == "" {
				continue
			}
			if _, ok := c.aliasIndex[key]; !ok {
				c.aliasIndex[key] = entry.Slug
			}
			c.matchers[entry.Slug] = append(c.matchers[entry.Slug], compileWordRegex(p))
		}
	}
}

// MatchSignals matches catalog entries against deterministic signals.
func (c *Catalog) MatchSignals(signals Signals) []Entry {
	scores := make(map[string]int)

	scoreText := func(text string, weight int) {
		normalized := normalizeText(text)
		if normalized == "" {
			return
		}
		for _, entry := range c.Entries {
			if c.matchesEntry(entry.Slug, normalized) {
				scores[entry.Slug] += weight
			}
		}
	}

	scoreText(signals.Language, 4)
	scoreText(signals.Title, 3)
	scoreText(strings.Join(signals.Labels, " "), 2)
	scoreText(truncateText(signals.Body, 8000), 1)

	return c.sortedByScore(scores, DefaultMaxResults)
}

// MatchKeywords maps a list of keywords (e.g. from LLM) to catalog entries.
func (c *Catalog) MatchKeywords(keywords []string) []Entry {
	if len(keywords) == 0 {
		return nil
	}
	scores := make(map[string]int)
	for _, keyword := range keywords {
		key := normalizeKeyword(keyword)
		if key == "" {
			continue
		}
		if slug, ok := c.aliasIndex[key]; ok {
			scores[slug] += 1
		}
	}
	return c.sortedByScore(scores, DefaultMaxResults)
}

func (c *Catalog) matchesEntry(slug string, normalizedText string) bool {
	matchers := c.matchers[slug]
	if len(matchers) == 0 {
		return false
	}
	for _, re := range matchers {
		if re.MatchString(normalizedText) {
			return true
		}
	}
	return false
}

func (c *Catalog) sortedByScore(scores map[string]int, limit int) []Entry {
	if len(scores) == 0 {
		return nil
	}

	type scoredEntry struct {
		entry Entry
		score int
	}

	var list []scoredEntry
	for _, entry := range c.Entries {
		if score, ok := scores[entry.Slug]; ok && score > 0 {
			list = append(list, scoredEntry{entry: entry, score: score})
		}
	}

	sort.SliceStable(list, func(i, j int) bool {
		if list[i].score == list[j].score {
			return list[i].entry.Name < list[j].entry.Name
		}
		return list[i].score > list[j].score
	})

	if limit <= 0 {
		limit = DefaultMaxResults
	}
	if len(list) > limit {
		list = list[:limit]
	}

	entries := make([]Entry, 0, len(list))
	for _, item := range list {
		entries = append(entries, item.entry)
	}
	return entries
}

// MergeEntries merges deterministic and candidate entries, preserving order and uniqueness.
func MergeEntries(primary []Entry, secondary []Entry, limit int) []Entry {
	seen := make(map[string]bool)
	var result []Entry

	add := func(entry Entry) {
		if entry.Slug == "" || seen[entry.Slug] {
			return
		}
		seen[entry.Slug] = true
		result = append(result, entry)
	}

	for _, entry := range primary {
		add(entry)
	}
	for _, entry := range secondary {
		add(entry)
	}

	if limit <= 0 {
		limit = DefaultMaxResults
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result
}

func normalizeText(input string) string {
	return normalizeKeyword(input)
}

func normalizeKeyword(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return ""
	}

	var b strings.Builder
	lastSpace := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteRune(' ')
			lastSpace = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func compileWordRegex(input string) *regexp.Regexp {
	needle := normalizeKeyword(input)
	if needle == "" {
		return regexp.MustCompile("$^")
	}
	pattern := `\b` + regexp.QuoteMeta(needle) + `\b`
	return regexp.MustCompile(pattern)
}

func truncateText(input string, max int) string {
	if max <= 0 || len(input) <= max {
		return input
	}
	return input[:max]
}
