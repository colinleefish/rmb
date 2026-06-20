package handler

import (
	"net/http"

	"github.com/colinleefish/rmb/internal/service/health"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	health *health.Service
}

func NewHealthHandler(svc *health.Service) *HealthHandler {
	return &HealthHandler{health: svc}
}

func (h *HealthHandler) Get(c *gin.Context) {
	status, err := h.health.Check(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"db": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
