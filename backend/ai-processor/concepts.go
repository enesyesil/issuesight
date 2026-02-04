package main

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/backend/ai-processor/transformer"
	"github.com/issuesight/issuesight/internal/concepts"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/concept"
	"github.com/issuesight/issuesight/internal/platform/db/ent/project"
	"github.com/issuesight/issuesight/internal/platform/db/ent/projectconcept"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialcontent"
)

const (
	maxProjectConcepts       = concepts.DefaultMaxResults + 1
	maxProjectDescriptionLen = 240
)

// syncConcepts creates/upserts concepts from deterministic signals and links them to project + tutorial.
// This is best-effort; failures should not block tutorial generation.
func (s *Service) syncConcepts(ctx context.Context, payload *transformer.StreamIssuePayload, issueUUID uuid.UUID, prerequisites []string) error {
	catalog, err := concepts.DefaultCatalog()
	if err != nil {
		return err
	}

	content, err := s.db.TutorialContent.Query().
		Where(tutorialcontent.IssueIDEQ(issueUUID)).
		Only(ctx)
	if err != nil {
		return err
	}

	proj, err := s.db.Project.Query().
		Where(project.FullNameEQ(payload.FullName)).
		Only(ctx)
	if err != nil {
		return err
	}

	signals := concepts.Signals{
		Language: payload.RepoLanguage,
		Title:    payload.Title,
		Body:     payload.Body,
		Labels:   payload.Labels,
	}

	deterministic := catalog.MatchSignals(signals)
	entries := buildConceptEntries(payload, catalog, deterministic, prerequisites)
	if len(entries) == 0 {
		return nil
	}

	shouldLinkProject := true
	projectConceptCount, err := s.db.ProjectConcept.Query().
		Where(projectconcept.ProjectIDEQ(proj.ID)).
		Count(ctx)
	if err == nil && projectConceptCount > 0 {
		shouldLinkProject = false
	}

	var firstErr error
	for _, entry := range entries {
		conceptRow, err := s.getOrCreateConcept(ctx, entry)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if shouldLinkProject {
			if err := s.ensureProjectConcept(ctx, proj.ID, conceptRow.ID); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if err := s.ensureTutorialConcept(ctx, content.ID, conceptRow.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// syncProjectConcepts creates project-level concepts in parallel with tutorial generation.
// It links only project concepts (no tutorial link) and is safe to run multiple times.
func (s *Service) syncProjectConcepts(ctx context.Context, payload *transformer.StreamIssuePayload) error {
	if payload == nil {
		return nil
	}

	catalog, err := concepts.DefaultCatalog()
	if err != nil {
		return err
	}

	proj, err := s.db.Project.Query().
		Where(project.FullNameEQ(payload.FullName)).
		Only(ctx)
	if err != nil {
		return err
	}

	projectConceptCount, err := s.db.ProjectConcept.Query().
		Where(projectconcept.ProjectIDEQ(proj.ID)).
		Count(ctx)
	if err != nil {
		return err
	}
	if projectConceptCount > 0 {
		return nil
	}

	signals := concepts.Signals{
		Language: payload.RepoLanguage,
		Title:    payload.Title,
		Body:     payload.Body,
		Labels:   payload.Labels,
	}

	deterministic := catalog.MatchSignals(signals)
	entries := buildProjectConceptEntries(payload, catalog, deterministic)
	if len(entries) == 0 {
		return nil
	}

	var firstErr error
	for _, entry := range entries {
		conceptRow, err := s.getOrCreateConcept(ctx, entry)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := s.ensureProjectConcept(ctx, proj.ID, conceptRow.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func buildConceptEntries(
	payload *transformer.StreamIssuePayload,
	catalog *concepts.Catalog,
	deterministic []concepts.Entry,
	prerequisites []string,
) []concepts.Entry {
	primary := deterministic
	if projectEntry := projectEntryFromPayload(payload, catalog); projectEntry != nil {
		primary = append([]concepts.Entry{*projectEntry}, primary...)
	}
	secondary := catalog.MatchKeywords(prerequisites)
	return concepts.MergeEntries(primary, secondary, maxProjectConcepts)
}

func buildProjectConceptEntries(
	payload *transformer.StreamIssuePayload,
	catalog *concepts.Catalog,
	deterministic []concepts.Entry,
) []concepts.Entry {
	primary := deterministic
	if projectEntry := projectEntryFromPayload(payload, catalog); projectEntry != nil {
		primary = append([]concepts.Entry{*projectEntry}, primary...)
	}
	return concepts.MergeEntries(primary, nil, maxProjectConcepts)
}

func projectEntryFromPayload(payload *transformer.StreamIssuePayload, catalog *concepts.Catalog) *concepts.Entry {
	if payload == nil {
		return nil
	}
	slug := slugifyConcept(payload.FullName)
	if slug == "" {
		return nil
	}
	if catalogHasSlug(catalog, slug) {
		return nil
	}
	name := strings.TrimSpace(payload.FullName)
	if name == "" {
		name = strings.TrimSpace(payload.Repo)
	}
	if name == "" {
		return nil
	}
	desc := strings.TrimSpace(payload.RepoDescription)
	if desc == "" {
		desc = fmt.Sprintf("Open-source project %s.", payload.FullName)
	}
	desc = truncateConceptDescription(desc)

	aliases := []string{}
	if payload.FullName != "" {
		aliases = append(aliases, payload.FullName)
	}
	if payload.Repo != "" && payload.Repo != payload.FullName {
		aliases = append(aliases, payload.Repo)
	}

	triggers := []string{}
	if payload.FullName != "" {
		triggers = append(triggers, payload.FullName)
	}
	if payload.Repo != "" && payload.Repo != payload.FullName {
		triggers = append(triggers, payload.Repo)
	}

	return &concepts.Entry{
		Slug:        slug,
		Name:        name,
		Description: desc,
		Category:    "project",
		Aliases:     aliases,
		Triggers:    triggers,
	}
}

func catalogHasSlug(catalog *concepts.Catalog, slug string) bool {
	if catalog == nil || slug == "" {
		return false
	}
	for _, entry := range catalog.Entries {
		if entry.Slug == slug {
			return true
		}
	}
	return false
}

func slugifyConcept(input string) string {
	trimmed := strings.TrimSpace(strings.ToLower(input))
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func truncateConceptDescription(input string) string {
	if input == "" {
		return ""
	}
	if len(input) <= maxProjectDescriptionLen {
		return input
	}
	return input[:maxProjectDescriptionLen-3] + "..."
}

func (s *Service) getOrCreateConcept(ctx context.Context, entry concepts.Entry) (*ent.Concept, error) {
	existing, err := s.db.Concept.Query().
		Where(concept.SlugEQ(entry.Slug)).
		Only(ctx)
	if err == nil {
		needsUpdate := false
		updater := s.db.Concept.UpdateOneID(existing.ID)
		if existing.Name == "" && entry.Name != "" {
			updater.SetName(entry.Name)
			needsUpdate = true
		}
		if existing.Description == "" && entry.Description != "" {
			updater.SetDescription(entry.Description)
			needsUpdate = true
		}
		if existing.Category == "" && entry.Category != "" {
			updater.SetCategory(entry.Category)
			needsUpdate = true
		}
		if existing.TutorialMarkdown == "" {
			updater.SetTutorialMarkdown(concepts.BuildTutorialMarkdown(entry))
			needsUpdate = true
		}
		if needsUpdate {
			if _, updateErr := updater.Save(ctx); updateErr == nil {
				return s.db.Concept.Query().
					Where(concept.IDEQ(existing.ID)).
					Only(ctx)
			}
		}
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	created, err := s.db.Concept.Create().
		SetSlug(entry.Slug).
		SetName(entry.Name).
		SetDescription(entry.Description).
		SetCategory(entry.Category).
		SetTutorialMarkdown(concepts.BuildTutorialMarkdown(entry)).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return s.db.Concept.Query().
				Where(concept.SlugEQ(entry.Slug)).
				Only(ctx)
		}
		return nil, err
	}

	return created, nil
}

func (s *Service) ensureProjectConcept(ctx context.Context, projectID, conceptID uuid.UUID) error {
	if projectID == uuid.Nil || conceptID == uuid.Nil {
		return nil
	}
	_, err := s.db.ProjectConcept.Create().
		SetProjectID(projectID).
		SetConceptID(conceptID).
		Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		return nil
	}
	return err
}

func (s *Service) ensureTutorialConcept(ctx context.Context, contentID, conceptID uuid.UUID) error {
	if contentID == uuid.Nil || conceptID == uuid.Nil {
		return nil
	}
	_, err := s.db.TutorialConcept.Create().
		SetContentID(contentID).
		SetConceptID(conceptID).
		Save(ctx)
	if err != nil && ent.IsConstraintError(err) {
		return nil
	}
	return err
}
