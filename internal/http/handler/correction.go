package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/correction"
	"github.com/colinleefish/rmb/internal/service/memory"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CorrectionHandler exposes writing human corrections over HTTP.
type CorrectionHandler struct {
	service *correction.Service
	db      *gorm.DB
}

func NewCorrectionHandler(service *correction.Service, db *gorm.DB) *CorrectionHandler {
	return &CorrectionHandler{service: service, db: db}
}

type createCorrectionRequest struct {
	TargetURIs []string `json:"target_uris"`
	Statement  string   `json:"statement"`
}

func (h *CorrectionHandler) Create(c *gin.Context) {
	var req createCorrectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.JSON(c, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.service.Create(c.Request.Context(), correction.CreateInput{
		TargetURIs: req.TargetURIs,
		Statement:  req.Statement,
	})
	if err != nil {
		httperr.Write(c, err)
		return
	}
	wakeT3ForTargets(c, h.db, []string(row.TargetURIs))
	c.JSON(http.StatusCreated, gin.H{
		"uri":         row.URI,
		"target_uris": row.TargetURIs,
	})
}

// wakeT3ForTargets re-distills the targeted memories so the correction is baked
// into the body. Best-effort: a failure here must not fail the write — the
// correction is already durable and the read-time overlay still applies.
func wakeT3ForTargets(c *gin.Context, db *gorm.DB, targets []string) {
	if _, err := memory.EnqueueSessionsForMemoryTargets(c.Request.Context(), db, targets); err != nil {
		log.Printf("correction wake-t3 failed (overlay still applies): %v", err)
	}
}

func (h *CorrectionHandler) List(c *gin.Context) {
	p := parseListParams(c)
	rows, total, err := h.service.List(c.Request.Context(), c.Query("target"), p.Limit, p.Offset)
	if err != nil {
		httperr.Write(c, err)
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
			"statement":   statement,
			"target_uris": []string(r.TargetURIs),
			"created_at":  r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "limit": p.Limit, "offset": p.Offset})
}

func (h *CorrectionHandler) Retract(c *gin.Context) {
	target := c.Query("uri")
	if target == "" {
		httperr.JSON(c, http.StatusBadRequest, "uri is required")
		return
	}
	targets, err := h.service.Retract(c.Request.Context(), target)
	switch {
	case err == nil:
		wakeT3ForTargets(c, h.db, targets)
		c.JSON(http.StatusOK, gin.H{"uri": target, "retracted": true})
	case errors.Is(err, correction.ErrInvalidInput):
		httperr.Write(c, err)
	case errors.Is(err, correction.ErrNotFound):
		httperr.Write(c, err)
	default:
		httperr.Write(c, err)
	}
}
