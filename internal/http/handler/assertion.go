package handler

import (
	"errors"
	"net/http"

	"github.com/colinleefish/mypast/internal/service/assertion"
	"github.com/gin-gonic/gin"
)

// AssertionHandler exposes writing human corrections over HTTP.
type AssertionHandler struct {
	service *assertion.Service
}

func NewAssertionHandler(service *assertion.Service) *AssertionHandler {
	return &AssertionHandler{service: service}
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
	c.JSON(http.StatusCreated, gin.H{
		"uri":         row.URI,
		"kind":        row.Kind,
		"target_uris": row.TargetURIs,
	})
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
	err := h.service.Retract(c.Request.Context(), target)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"uri": target, "retracted": true})
	case errors.Is(err, assertion.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, assertion.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
