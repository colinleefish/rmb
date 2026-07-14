package handler

import (
	"net/http"

	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/agentmemory"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AgentMemoryHandler updates the curated rmb://agent document.
type AgentMemoryHandler struct {
	db *gorm.DB
}

func NewAgentMemoryHandler(db *gorm.DB) *AgentMemoryHandler {
	return &AgentMemoryHandler{db: db}
}

// Put replaces the body of rmb://agent (versioned like profile).
func (h *AgentMemoryHandler) Put(c *gin.Context) {
	var req struct {
		Body string `json:"body"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.JSON(c, http.StatusBadRequest, "body is required")
		return
	}
	if err := agentmemory.ReplaceBody(c.Request.Context(), h.db, req.Body); err != nil {
		httperr.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"uri": "rmb://agent"})
}
