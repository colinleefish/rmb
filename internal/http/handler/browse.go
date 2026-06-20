package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/colinleefish/rmb/internal/service/browse"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultPageLimit = 25
	maxPageLimit     = 200
)

// parseListParams reads pagination/search/sort from the query string and clamps
// limit/offset to safe bounds. Sort/order are validated downstream against a
// per-entity allowlist, so they are passed through verbatim here.
func parseListParams(c *gin.Context) browse.ListParams {
	limit := defaultPageLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	offset := 0
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v > 0 {
		offset = v
	}
	return browse.ListParams{
		Limit:  limit,
		Offset: offset,
		Query:  c.Query("q"),
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	}
}

type BrowseHandler struct {
	svc *browse.Service
}

func NewBrowseHandler(svc *browse.Service) *BrowseHandler {
	return &BrowseHandler{svc: svc}
}

func (h *BrowseHandler) Overview(c *gin.Context) {
	out, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *BrowseHandler) ListSessions(c *gin.Context) {
	rows, err := h.svc.ListSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *BrowseHandler) GetSession(c *gin.Context) {
	key := c.Param("session_key")
	detail, err := h.svc.GetSession(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *BrowseHandler) ListAtoms(c *gin.Context) {
	p := parseListParams(c)
	rows, total, err := h.svc.ListAtoms(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total, "limit": p.Limit, "offset": p.Offset})
}

func (h *BrowseHandler) ListScenes(c *gin.Context) {
	p := parseListParams(c)
	rows, total, err := h.svc.ListScenes(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total, "limit": p.Limit, "offset": p.Offset})
}

func (h *BrowseHandler) ListMemories(c *gin.Context) {
	p := parseListParams(c)
	rows, total, err := h.svc.ListMemories(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total, "limit": p.Limit, "offset": p.Offset})
}

func (h *BrowseHandler) ListPipelineStates(c *gin.Context) {
	rows, err := h.svc.ListPipelineStates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

func (h *BrowseHandler) ListTasks(c *gin.Context) {
	p := parseListParams(c)
	rows, total, err := h.svc.ListTasks(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total, "limit": p.Limit, "offset": p.Offset})
}
