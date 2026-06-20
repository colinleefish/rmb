package handler

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/colinleefish/mypast/internal/service/alias"
	"github.com/colinleefish/mypast/internal/service/memory"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AliasHandler exposes writing entity aliases over HTTP.
type AliasHandler struct {
	service *alias.Service
	db      *gorm.DB
}

func NewAliasHandler(service *alias.Service, db *gorm.DB) *AliasHandler {
	return &AliasHandler{service: service, db: db}
}

type createAliasRequest struct {
	AliasURI     string `json:"alias_uri"`
	CanonicalURI string `json:"canonical_uri"`
	Note         string `json:"note"`
}

func (h *AliasHandler) Create(c *gin.Context) {
	var req createAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.service.Create(c.Request.Context(), alias.CreateInput{
		AliasURI:     req.AliasURI,
		CanonicalURI: req.CanonicalURI,
		Note:         req.Note,
	})
	switch {
	case err == nil:
		// Wake T3 for the sessions feeding both slugs BEFORE superseding the alias
		// row (so its source scenes still resolve), then retire the alias slug's
		// standalone memory — its facts move into the canonical at the next rollup.
		h.wakeT3(c, []string{row.AliasURI, row.CanonicalURI})
		if err := memory.SupersedeActiveMemory(c.Request.Context(), h.db, row.AliasURI, time.Now().UTC()); err != nil {
			log.Printf("alias supersede %s failed (read-time fold still applies): %v", row.AliasURI, err)
		}
		c.JSON(http.StatusCreated, gin.H{
			"uri":           row.URI,
			"alias_uri":     row.AliasURI,
			"canonical_uri": row.CanonicalURI,
		})
	case errors.Is(err, alias.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, alias.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// wakeT3 re-distills the memories whose identity changed so the canonical body
// reflects the fold. Best-effort: a failure must not fail the write — the alias
// is durable and the read-time fold still applies until the next rollup.
func (h *AliasHandler) wakeT3(c *gin.Context, targets []string) {
	if _, err := memory.EnqueueSessionsForMemoryTargets(c.Request.Context(), h.db, targets); err != nil {
		log.Printf("alias wake-t3 failed (read-time fold still applies): %v", err)
	}
}

func (h *AliasHandler) List(c *gin.Context) {
	rows, err := h.service.List(c.Request.Context(), c.Query("uri"))
	if err != nil {
		if errors.Is(err, alias.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		note := ""
		if r.Note != nil {
			note = *r.Note
		}
		items = append(items, gin.H{
			"uri":           r.URI,
			"alias_uri":     r.AliasURI,
			"canonical_uri": r.CanonicalURI,
			"note":          note,
			"created_at":    r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ListCandidates returns alias candidates proposed by the suggest worker,
// filtered by status (default pending).
func (h *AliasHandler) ListCandidates(c *gin.Context) {
	items, err := h.service.ListCandidates(c.Request.Context(), c.Query("status"))
	if err != nil {
		if errors.Is(err, alias.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type candidateActionRequest struct {
	ID string `json:"id"`
}

// ConfirmCandidate promotes a pending candidate into a live alias and runs the
// same post-write side-effects as a manual alias set (wake T3, supersede the
// alias slug's standalone memory).
func (h *AliasHandler) ConfirmCandidate(c *gin.Context) {
	var req candidateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := h.service.ConfirmCandidate(c.Request.Context(), req.ID)
	switch {
	case err == nil:
		h.wakeT3(c, []string{row.AliasURI, row.CanonicalURI})
		if err := memory.SupersedeActiveMemory(c.Request.Context(), h.db, row.AliasURI, time.Now().UTC()); err != nil {
			log.Printf("alias confirm supersede %s failed (read-time fold still applies): %v", row.AliasURI, err)
		}
		c.JSON(http.StatusCreated, gin.H{
			"uri":           row.URI,
			"alias_uri":     row.AliasURI,
			"canonical_uri": row.CanonicalURI,
		})
	case errors.Is(err, alias.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, alias.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, alias.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// RejectCandidate marks a pending candidate rejected so it is never re-proposed.
func (h *AliasHandler) RejectCandidate(c *gin.Context) {
	var req candidateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.service.RejectCandidate(c.Request.Context(), req.ID)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"id": req.ID, "rejected": true})
	case errors.Is(err, alias.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, alias.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func (h *AliasHandler) Retract(c *gin.Context) {
	target := c.Query("uri")
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uri is required"})
		return
	}
	canonicalURI, err := h.service.Retract(c.Request.Context(), target)
	switch {
	case err == nil:
		// Re-roll the canonical so it re-distills without the unfolded atoms; the
		// retired alias slug's own bucket is rebuilt on the same global cycle.
		h.wakeT3(c, []string{canonicalURI})
		c.JSON(http.StatusOK, gin.H{"uri": target, "retracted": true})
	case errors.Is(err, alias.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, alias.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
