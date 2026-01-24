package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/issuesight/issuesight/internal/domain"
	"github.com/issuesight/issuesight/internal/platform/cache"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorial"
	"github.com/issuesight/issuesight/internal/platform/db/ent/tutorialcontent"
)

// TutorialResponse represents a tutorial in API responses.
type TutorialResponse struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	MarkdownBody string    `json:"markdown_body"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tutorials/{id} [get]
func (h *TutorialHandler) Get() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

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

		// Try cache first (cache-aside pattern)
		cacheKey := domain.CacheKeyTutorial + tutorialID
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

		// Build response
		response := TutorialResponse{
			ID:           content.ID.String(),
			Title:        content.Title,
			MarkdownBody: content.MarkdownBody,
			Status:       string(content.Status),
			CreatedAt:    content.CreatedAt,
			UpdatedAt:    content.UpdatedAt,
		}

		// Marshal and cache the response
		jsonData, err := json.Marshal(response)
		if err == nil {
			// Cache for 5 minutes
			_ = h.cache.Set(ctx, cacheKey, jsonData, 5*time.Minute)
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
// @Failure      500  {object}  ErrorResponse
// @Router       /tutorials [get]
func (h *TutorialHandler) List() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// TODO: Get user ID from auth context
		// For now, we'll list all tutorials (limited)
		// userID := ctx.Value("user_id").(uuid.UUID)

		// Query tutorials with their content
		tutorials, err := h.db.Tutorial.Query().
			WithContent().
			Order(tutorial.ByCreatedAt()).
			Limit(50).
			All(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "Failed to fetch tutorials")
			return
		}

		// Build response
		var tutorialResponses []TutorialResponse
		for _, t := range tutorials {
			if t.Edges.Content != nil {
				tutorialResponses = append(tutorialResponses, TutorialResponse{
					ID:           t.Edges.Content.ID.String(),
					Title:        t.Edges.Content.Title,
					MarkdownBody: t.Edges.Content.MarkdownBody,
					Status:       string(t.Edges.Content.Status),
					CreatedAt:    t.Edges.Content.CreatedAt,
					UpdatedAt:    t.Edges.Content.UpdatedAt,
				})
			}
		}

		response := TutorialListResponse{
			Tutorials: tutorialResponses,
			Count:     len(tutorialResponses),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
