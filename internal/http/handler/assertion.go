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
