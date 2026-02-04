package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/issuesight/issuesight/internal/concepts"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/concept"
)

// ConceptResponse represents a concept in API responses.
type ConceptResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Tutorial    string `json:"tutorial_markdown,omitempty"`
}

// ConceptListResponse is the response for listing concepts.
type ConceptListResponse struct {
	Concepts []ConceptResponse `json:"concepts"`
	Count    int               `json:"count"`
}

// ConceptHandler handles concept-related endpoints.
type ConceptHandler struct {
	db *ent.Client
}

// NewConceptHandler creates a new concept handler.
func NewConceptHandler(db *ent.Client) *ConceptHandler {
	return &ConceptHandler{db: db}
}

// List handles GET /api/concepts - list all concepts.
func (h *ConceptHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		concepts, err := h.db.Concept.Query().
			Order(concept.ByName()).
			Limit(300).
			All(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch concepts")
			return
		}
		if len(concepts) == 0 {
			if seeded, seedErr := h.seedCatalog(ctx); seedErr == nil && seeded {
				concepts, err = h.db.Concept.Query().
					Order(concept.ByName()).
					Limit(300).
					All(ctx)
				if err != nil {
					writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch concepts")
					return
				}
			}
		}

		responses := make([]ConceptResponse, 0, len(concepts))
		for _, c := range concepts {
			responses = append(responses, ConceptResponse{
				ID:          c.ID.String(),
				Slug:        c.Slug,
				Name:        c.Name,
				Description: c.Description,
				Category:    c.Category,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ConceptListResponse{
			Concepts: responses,
			Count:    len(responses),
		})
	}
}

// Get handles GET /api/concepts/{slug} - fetch a concept by slug.
func (h *ConceptHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		slug := r.PathValue("slug")
		if slug == "" {
			writeError(w, http.StatusBadRequest, "missing_slug", "Concept slug is required")
			return
		}

		c, err := h.db.Concept.Query().
			Where(concept.SlugEQ(slug)).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "not_found", "Concept not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch concept")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ConceptResponse{
			ID:          c.ID.String(),
			Slug:        c.Slug,
			Name:        c.Name,
			Description: c.Description,
			Category:    c.Category,
			Tutorial:    c.TutorialMarkdown,
		})
	}
}

func (h *ConceptHandler) seedCatalog(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := concepts.AutoSeed(ctx, h.db, slog.Default(), concepts.SeedOptions{}); err != nil {
		return false, err
	}
	return true, nil
}
