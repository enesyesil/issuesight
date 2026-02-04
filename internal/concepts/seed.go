package concepts

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/concept"
	"github.com/issuesight/issuesight/internal/platform/db/ent/githubissue"
	"github.com/issuesight/issuesight/internal/platform/db/ent/projectconcept"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialconcept"
)

const (
	defaultProjectIssueLimit = 25
)

// SeedOptions controls seeding behavior.
type SeedOptions struct {
	// ForceBackfill recomputes links even when concepts already exist.
	ForceBackfill bool
}

// AutoSeed performs deterministic seeding and backfill from existing DB state.
// It is safe to run multiple times.
func AutoSeed(ctx context.Context, db *ent.Client, logger *slog.Logger, opts SeedOptions) error {
	if logger == nil {
		logger = slog.Default()
	}

	catalog, err := DefaultCatalog()
	if err != nil {
		return err
	}

	slugToID, err := seedCatalog(ctx, db, catalog, logger)
	if err != nil {
		return err
	}

	if err := backfillProjects(ctx, db, catalog, slugToID, logger, opts); err != nil {
		logger.Warn("concepts: project backfill failed", "error", err)
	}
	if err := backfillTutorials(ctx, db, catalog, slugToID, logger, opts); err != nil {
		logger.Warn("concepts: tutorial backfill failed", "error", err)
	}

	return nil
}

func seedCatalog(ctx context.Context, db *ent.Client, catalog *Catalog, logger *slog.Logger) (map[string]uuid.UUID, error) {
	slugToID := make(map[string]uuid.UUID)

	existing, err := db.Concept.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	existingBySlug := make(map[string]*ent.Concept)
	for _, row := range existing {
		slugToID[row.Slug] = row.ID
		existingBySlug[row.Slug] = row
	}

	createdCount := 0
	for _, entry := range catalog.Entries {
		if row, ok := existingBySlug[entry.Slug]; ok {
			if updateConceptFromEntry(ctx, db, row, entry, logger); row != nil {
				slugToID[row.Slug] = row.ID
			}
			continue
		}
		row, err := db.Concept.Create().
			SetSlug(entry.Slug).
			SetName(entry.Name).
			SetDescription(entry.Description).
			SetCategory(entry.Category).
			SetTutorialMarkdown(BuildTutorialMarkdown(entry)).
			Save(ctx)
		if err != nil {
			if ent.IsConstraintError(err) {
				row, err = db.Concept.Query().Where(concept.SlugEQ(entry.Slug)).Only(ctx)
			}
		}
		if err != nil {
			logger.Warn("concepts: failed to seed concept", "slug", entry.Slug, "error", err)
			continue
		}
		if row != nil {
			slugToID[row.Slug] = row.ID
			createdCount++
		}
	}

	if createdCount > 0 {
		logger.Info("concepts: catalog seeded", "created", createdCount)
	}

	return slugToID, nil
}

func updateConceptFromEntry(ctx context.Context, db *ent.Client, row *ent.Concept, entry Entry, logger *slog.Logger) {
	if row == nil {
		return
	}

	needsUpdate := false
	updater := db.Concept.UpdateOneID(row.ID)

	if row.Name == "" && entry.Name != "" {
		updater.SetName(entry.Name)
		needsUpdate = true
	}
	if row.Description == "" && entry.Description != "" {
		updater.SetDescription(entry.Description)
		needsUpdate = true
	}
	if row.Category == "" && entry.Category != "" {
		updater.SetCategory(entry.Category)
		needsUpdate = true
	}
	if row.TutorialMarkdown == "" {
		updater.SetTutorialMarkdown(BuildTutorialMarkdown(entry))
		needsUpdate = true
	}

	if !needsUpdate {
		return
	}

	if _, err := updater.Save(ctx); err != nil {
		if logger != nil {
			logger.Warn("concepts: failed to update concept metadata", "slug", entry.Slug, "error", err)
		}
	}
}

func backfillProjects(ctx context.Context, db *ent.Client, catalog *Catalog, slugToID map[string]uuid.UUID, logger *slog.Logger, opts SeedOptions) error {
	projects, err := db.Project.Query().All(ctx)
	if err != nil {
		return err
	}

	for _, proj := range projects {
		count, err := db.ProjectConcept.Query().
			Where(projectconcept.ProjectIDEQ(proj.ID)).
			Count(ctx)
		if err != nil {
			logger.Warn("concepts: project concept count failed", "project_id", proj.ID, "error", err)
			continue
		}
		if count > 0 && !opts.ForceBackfill {
			continue
		}

		issues, err := db.GithubIssue.Query().
			Where(githubissue.ProjectIDEQ(proj.ID)).
			Limit(defaultProjectIssueLimit).
			All(ctx)
		if err != nil {
			logger.Warn("concepts: project issue query failed", "project_id", proj.ID, "error", err)
			continue
		}

		signals := buildProjectSignals(proj.Language, issues)
		entries := catalog.MatchSignals(signals)
		if len(entries) == 0 {
			continue
		}

		for _, entry := range entries {
			conceptID := slugToID[entry.Slug]
			if conceptID == uuid.Nil {
				row, err := ensureConceptRow(ctx, db, entry)
				if err != nil {
					logger.Warn("concepts: ensure concept failed", "slug", entry.Slug, "error", err)
					continue
				}
				conceptID = row.ID
				slugToID[entry.Slug] = conceptID
			}
			if err := linkProjectConcept(ctx, db, proj.ID, conceptID); err != nil {
				logger.Warn("concepts: link project concept failed", "project_id", proj.ID, "slug", entry.Slug, "error", err)
			}
		}
	}

	return nil
}

func backfillTutorials(ctx context.Context, db *ent.Client, catalog *Catalog, slugToID map[string]uuid.UUID, logger *slog.Logger, opts SeedOptions) error {
	contents, err := db.TutorialContent.Query().
		WithIssue(func(q *ent.GithubIssueQuery) {
			q.WithProject()
		}).
		All(ctx)
	if err != nil {
		return err
	}

	for _, content := range contents {
		if !opts.ForceBackfill {
			count, err := db.TutorialConcept.Query().
				Where(tutorialconcept.ContentIDEQ(content.ID)).
				Count(ctx)
			if err != nil {
				logger.Warn("concepts: tutorial concept count failed", "content_id", content.ID, "error", err)
				continue
			}
			if count > 0 {
				continue
			}
		}

		issue := content.Edges.Issue
		if issue == nil || issue.Edges.Project == nil {
			continue
		}

		signals := buildIssueSignals(issue.Edges.Project.Language, issue)
		entries := catalog.MatchSignals(signals)
		if len(entries) == 0 {
			continue
		}

		for _, entry := range entries {
			conceptID := slugToID[entry.Slug]
			if conceptID == uuid.Nil {
				row, err := ensureConceptRow(ctx, db, entry)
				if err != nil {
					logger.Warn("concepts: ensure concept failed", "slug", entry.Slug, "error", err)
					continue
				}
				conceptID = row.ID
				slugToID[entry.Slug] = conceptID
			}
			if err := linkTutorialConcept(ctx, db, content.ID, conceptID); err != nil {
				logger.Warn("concepts: link tutorial concept failed", "content_id", content.ID, "slug", entry.Slug, "error", err)
			}
		}
	}

	return nil
}

func ensureConceptRow(ctx context.Context, db *ent.Client, entry Entry) (*ent.Concept, error) {
	row, err := db.Concept.Query().Where(concept.SlugEQ(entry.Slug)).Only(ctx)
	if err == nil {
		updateConceptFromEntry(ctx, db, row, entry, slog.Default())
		return row, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}
	row, err = db.Concept.Create().
		SetSlug(entry.Slug).
		SetName(entry.Name).
		SetDescription(entry.Description).
		SetCategory(entry.Category).
		SetTutorialMarkdown(BuildTutorialMarkdown(entry)).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return db.Concept.Query().Where(concept.SlugEQ(entry.Slug)).Only(ctx)
		}
		return nil, err
	}
	return row, nil
}

func linkProjectConcept(ctx context.Context, db *ent.Client, projectID, conceptID uuid.UUID) error {
	if projectID == uuid.Nil || conceptID == uuid.Nil {
		return nil
	}
	_, err := db.ProjectConcept.Create().
		SetProjectID(projectID).
		SetConceptID(conceptID).
		Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		return nil
	}
	return err
}

func linkTutorialConcept(ctx context.Context, db *ent.Client, contentID, conceptID uuid.UUID) error {
	if contentID == uuid.Nil || conceptID == uuid.Nil {
		return nil
	}
	_, err := db.TutorialConcept.Create().
		SetContentID(contentID).
		SetConceptID(conceptID).
		Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		return nil
	}
	return err
}

func buildProjectSignals(language string, issues []*ent.GithubIssue) Signals {
	var titles []string
	var bodies []string
	var labels []string

	for _, issue := range issues {
		title, body, lbls := extractIssueFields(issue.RawData)
		if title != "" {
			titles = append(titles, title)
		}
		if body != "" {
			bodies = append(bodies, body)
		}
		if len(lbls) > 0 {
			labels = append(labels, lbls...)
		}
	}

	return Signals{
		Language: language,
		Title:    strings.Join(titles, " "),
		Body:     strings.Join(bodies, " "),
		Labels:   labels,
	}
}

func buildIssueSignals(language string, issue *ent.GithubIssue) Signals {
	title, body, labels := extractIssueFields(issue.RawData)
	return Signals{
		Language: language,
		Title:    title,
		Body:     body,
		Labels:   labels,
	}
}

func extractIssueFields(raw map[string]interface{}) (string, string, []string) {
	if raw == nil {
		return "", "", nil
	}

	title, _ := raw["title"].(string)
	body, _ := raw["body"].(string)

	var labels []string
	switch v := raw["labels"].(type) {
	case []string:
		labels = append(labels, v...)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				labels = append(labels, s)
			}
		}
	}

	return title, body, labels
}
