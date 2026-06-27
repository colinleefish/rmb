package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/recall"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// queryEmbedder embeds a single query string (server-side) for vector recall.
type queryEmbedder interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

// RecallHandler exposes search over HTTP.
type RecallHandler struct {
	db    *gorm.DB
	embed queryEmbedder
}

func NewRecallHandler(db *gorm.DB, embed queryEmbedder) *RecallHandler {
	return &RecallHandler{db: db, embed: embed}
}

func (h *RecallHandler) embedQuery(ctx context.Context, query string) (pgarray.Vector, error) {
	vecs, err := h.embed.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, fmt.Errorf("embedding provider returned no vectors")
	}
	return pgarray.Vector(vecs[0]), nil
}

// Search handles GET /api/v1/search?q=<query>[&scope=memory,scene][&k=<n>].
// scope defaults to "memory,scene"; k defaults to 5.
func (h *RecallHandler) Search(c *gin.Context) {
	if h.embed == nil {
		httperr.JSON(c, http.StatusServiceUnavailable, "embeddings not configured")
		return
	}

	query := c.Query("q")
	if query == "" {
		httperr.JSON(c, http.StatusBadRequest, "q is required")
		return
	}

	k := 0
	if v := c.Query("k"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || parsed <= 0 {
			httperr.JSON(c, http.StatusBadRequest, "k must be a positive integer")
			return
		}
		k = parsed
	}

	var scopes []string
	if raw := strings.TrimSpace(c.Query("scope")); raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "memory" && s != "scene" {
				httperr.JSON(c, http.StatusBadRequest, "scope must be memory and/or scene")
				return
			}
			scopes = append(scopes, s)
		}
	}

	matches, err := recall.Search(c.Request.Context(), h.db, h.embedQuery, query, k, scopes)
	if err != nil {
		httperr.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": matches})
}
