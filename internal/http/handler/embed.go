package handler

import (
	"net/http"

	"github.com/colinleefish/mypast/internal/service/embed"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// EmbedHandler exposes embedding-worker stats over HTTP.
type EmbedHandler struct {
	db *gorm.DB
}

func NewEmbedHandler(db *gorm.DB) *EmbedHandler {
	return &EmbedHandler{db: db}
}

// Status returns embedding coverage across atoms, scenes, and memories.
func (h *EmbedHandler) Status(c *gin.Context) {
	rows, err := embed.Status(c.Request.Context(), h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		items = append(items, gin.H{
			"tier":     r.Tier,
			"total":    r.Total,
			"embedded": r.Total - r.Pending,
			"pending":  r.Pending,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
