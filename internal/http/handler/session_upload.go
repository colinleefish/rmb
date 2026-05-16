package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/service/session"
	"github.com/gin-gonic/gin"
)

type SessionUploadHandler struct {
	service *session.UploadService
}

type sessionUploadRequest struct {
	ScopeKey  string                  `json:"scope_key"`
	Title     string                  `json:"title"`
	StartedAt string                  `json:"started_at"`
	Messages  []sessionMessageRequest `json:"messages"`
}

type sessionMessageRequest struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content"`
}

func NewSessionUploadHandler(svc *session.UploadService) *SessionUploadHandler {
	return &SessionUploadHandler{service: svc}
}

func (h *SessionUploadHandler) Upload(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	var req sessionUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var startedAt *time.Time
	if strings.TrimSpace(req.StartedAt) != "" {
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartedAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "started_at must be RFC3339, e.g. 2026-05-09T17:00:00Z",
			})
			return
		}
		startedAt = &ts
	}

	messages := make([]session.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, session.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	result, err := h.service.Upload(c.Request.Context(), session.UploadInput{
		SessionID: sessionID,
		ScopeKey:  req.ScopeKey,
		Title:     req.Title,
		StartedAt: startedAt,
		Messages:  messages,
	})
	if err != nil {
		if errors.Is(err, session.ErrInvalidUploadInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uri":           result.URI,
		"parent_uri":    result.ParentURI,
		"root_uri":      result.RootURI,
		"category":      result.Category,
		"message_count": result.MessageCnt,
		"archive_index": result.ArchiveIdx,
		"created_at":    result.CreatedAt,
		"updated_at":    result.UpdatedAt,
	})
}
