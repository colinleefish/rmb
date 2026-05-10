package handler

import (
	"net/http"

	"github.com/colinleefish/mypast/internal/service"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	health *service.HealthService
}

func NewHealthHandler(health *service.HealthService) *HealthHandler {
	return &HealthHandler{health: health}
}

func (h *HealthHandler) Get(c *gin.Context) {
	status, err := h.health.Check(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"db": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
