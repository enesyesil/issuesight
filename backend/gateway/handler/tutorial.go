package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/backend/gateway/middleware"
	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorial"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialcontent"
	"github.com/issuesight/issuesight/internal/platform/log"
)

// TutorialResponse represents a tutorial in API responses.
type TutorialResponse struct {
	ID           string                    `json:"id"`
	Title        string                    `json:"title"`
	MarkdownBody string                    `json:"markdown_body"`
	Status       string                    `json:"status"`
	Concepts     []TutorialConceptResponse `json:"concepts,omitempty"`
	CreatedAt    time.Time                 `json:"created_at"`
	UpdatedAt    time.Time                 `json:"updated_at"`
}

// TutorialConceptResponse represents a concept tied to a tutorial.
type TutorialConceptResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// TutorialListResponse is the response for listing tutorials.
type TutorialListResponse struct {
	Tutorials []TutorialResponse `json:"tutorials"`
	Count     int                `json:"count"`
}

// TutorialHandler handles tutorial retrieval requests.
type TutorialHandler struct {
	cache cache.Cache
	db    *ent.Client
}

// NewTutorialHandler creates a new tutorial handler.
func NewTutorialHandler(cache cache.Cache, db *ent.Client) *TutorialHandler {
	return &TutorialHandler{
		cache: cache,
		db:    db,
	}
}

// Get handles GET /api/tutorials/{id} - get a specific tutorial.
// @Summary      Get Tutorial
// @Description  Retrieves a specific tutorial by ID.
// @Tags         tutorials
// @Produce      json
// @Param        id   path      string  true  "Tutorial ID"
// @Success      200  {object}  TutorialResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tutorials/{id} [get]
func (h *TutorialHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get authenticated user
		user := middleware.GetUser(ctx)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		// Get tutorial ID from path
		tutorialID := r.PathValue("id")
		if tutorialID == "" {
			writeError(w, http.StatusBadRequest, "missing_id", "Tutorial ID is required")
			return
		}

		// Parse UUID
		id, err := uuid.Parse(tutorialID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "Invalid tutorial ID format")
			return
		}

		// Check user has access to this tutorial (via Tutorial join table)
		hasAccess, err := h.db.Tutorial.Query().
			Where(
				tutorial.UserIDEQ(user.ID),
				tutorial.ContentIDEQ(id),
			).
			Exist(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to check access")
			return
		}
		if !hasAccess {
			writeError(w, http.StatusNotFound, "not_found", "Tutorial not found")
			return
		}

		// Try cache first (cache-aside pattern) - use user-scoped cache key
		cacheKey := domain.CacheKeyTutorial + user.ID.String() + ":" + tutorialID
		cachedData, err := h.cache.Get(ctx, cacheKey)
		if err == nil {
			// Cache hit - return cached response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write(cachedData)
			return
		}

		// Cache miss - fetch from database
		content, err := h.db.TutorialContent.Query().
			Where(tutorialcontent.ID(id)).
			First(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				writeError(w, http.StatusNotFound, "not_found", "Tutorial not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch tutorial")
			return
		}

		// Fetch linked concepts for this tutorial
		conceptRows, err := h.db.TutorialContent.Query().
			Where(tutorialcontent.ID(id)).
			QueryConcepts().
			All(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch tutorial concepts")
			return
		}

		concepts := make([]TutorialConceptResponse, 0, len(conceptRows))
		for _, c := range conceptRows {
			concepts = append(concepts, TutorialConceptResponse{
				ID:          c.ID.String(),
				Slug:        c.Slug,
				Name:        c.Name,
				Description: c.Description,
			})
		}

		// Build response
		response := TutorialResponse{
			ID:           content.ID.String(),
			Title:        content.Title,
			MarkdownBody: content.MarkdownBody,
			Status:       string(content.Status),
			Concepts:     concepts,
			CreatedAt:    content.CreatedAt,
			UpdatedAt:    content.UpdatedAt,
		}

		// Marshal and cache the response
		jsonData, err := json.Marshal(response)
		if err == nil {
			// Cache for 5 minutes
			if cacheErr := h.cache.Set(ctx, cacheKey, jsonData, 5*time.Minute); cacheErr != nil {
				log.Warn("failed to cache tutorial response",
					"tutorial_id", tutorialID,
					"error", cacheErr,
				)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "MISS")
		w.Write(jsonData)
	}
}

// List handles GET /api/tutorials - list user's tutorials.
// @Summary      List Tutorials
// @Description  Lists the tutorials for the authenticated user.
// @Tags         tutorials
// @Produce      json
// @Success      200  {object}  TutorialListResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tutorials [get]
func (h *TutorialHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Get authenticated user
		user := middleware.GetUser(ctx)
		if user == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		// Query tutorials the user has access to via the Tutorial join table
		tutorials, err := h.db.Tutorial.Query().
			Where(tutorial.UserIDEQ(user.ID)).
			QueryContent().
			Order(tutorialcontent.ByCreatedAt()).
			Limit(50).
			All(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch tutorials")
			return
		}

		// Build response
		var tutorialResponses []TutorialResponse
		for _, c := range tutorials {
			tutorialResponses = append(tutorialResponses, TutorialResponse{
				ID:           c.ID.String(),
				Title:        c.Title,
				MarkdownBody: c.MarkdownBody,
				Status:       string(c.Status),
				CreatedAt:    c.CreatedAt,
				UpdatedAt:    c.UpdatedAt,
			})
		}

		response := TutorialListResponse{
			Tutorials: tutorialResponses,
			Count:     len(tutorialResponses),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
