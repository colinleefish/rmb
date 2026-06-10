package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/colinleefish/mypast/internal/service/assertion"
	"github.com/colinleefish/mypast/internal/service/memory"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AssertionHandler exposes writing human corrections over HTTP.
type AssertionHandler struct {
	service *assertion.Service
	db      *gorm.DB
}

func NewAssertionHandler(service *assertion.Service, db *gorm.DB) *AssertionHandler {
	return &AssertionHandler{service: service, db: db}
}

type createAssertionRequest struct {
	Kind       string   `json:"kind"`
	TargetURIs []string `json:"target_uris"`
	Statement  string   `json:"statement"`
}

func (h *AssertionHandler) Create(c *gin.Context) {
	var req createAssertionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.service.Create(c.Request.Context(), assertion.CreateInput{
		Kind:       req.Kind,
		TargetURIs: req.TargetURIs,
		Statement:  req.Statement,
	})
	if err != nil {
		if errors.Is(err, assertion.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	wakeT3ForTargets(c, h.db, []string(row.TargetURIs))
	c.JSON(http.StatusCreated, gin.H{
		"uri":         row.URI,
		"kind":        row.Kind,
		"target_uris": row.TargetURIs,
	})
}

// wakeT3ForTargets re-distills the targeted memories so the correction is baked
// into the body. Best-effort: a failure here must not fail the write — the
// assertion is already durable and the read-time overlay still applies.
func wakeT3ForTargets(c *gin.Context, db *gorm.DB, targets []string) {
	if _, err := memory.EnqueueSessionsForMemoryTargets(c.Request.Context(), db, targets); err != nil {
		log.Printf("assertion wake-t3 failed (overlay still applies): %v", err)
	}
}

func (h *AssertionHandler) List(c *gin.Context) {
	rows, err := h.service.List(c.Request.Context(), c.Query("target"))
	if err != nil {
		if errors.Is(err, assertion.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		statement := ""
		if r.Statement != nil {
			statement = *r.Statement
		}
		items = append(items, gin.H{
			"uri":         r.URI,
			"kind":        r.Kind,
			"statement":   statement,
			"target_uris": []string(r.TargetURIs),
			"created_at":  r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *AssertionHandler) Retract(c *gin.Context) {
	target := c.Query("uri")
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uri is required"})
		return
	}
	targets, err := h.service.Retract(c.Request.Context(), target)
	switch {
	case err == nil:
		wakeT3ForTargets(c, h.db, targets)
		c.JSON(http.StatusOK, gin.H{"uri": target, "retracted": true})
	case errors.Is(err, assertion.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, assertion.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
