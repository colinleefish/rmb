package handler

import (
	"net/http"
	"strings"

	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/skill"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SkillHandler manages Agent Skills bundles at rmb://skills/<slug>.
type SkillHandler struct {
	db       *gorm.DB
	maxBytes int64
}

func NewSkillHandler(db *gorm.DB, maxBytes int64) *SkillHandler {
	return &SkillHandler{db: db, maxBytes: maxBytes}
}

// Put replaces a skill bundle (versioned).
func (h *SkillHandler) Put(c *gin.Context) {
	slug := strings.TrimSpace(c.Param("slug"))
	if slug == "" {
		httperr.JSON(c, http.StatusBadRequest, "slug is required")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes)

	var req struct {
		Files []struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		} `json:"files"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httperr.JSON(c, http.StatusBadRequest, "files array is required")
		return
	}
	files := make([]skill.FileInput, 0, len(req.Files))
	for _, f := range req.Files {
		files = append(files, skill.FileInput{Path: f.Path, Content: f.Content})
	}
	result, err := skill.ReplaceBundle(c.Request.Context(), h.db, slug, skill.BundleInput{Files: files})
	if err != nil {
		httperr.Write(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
