package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/http/httperr"
	"github.com/colinleefish/rmb/internal/service/session"
	"github.com/gin-gonic/gin"
)

type SessionUploadHandler struct {
	service        *session.UploadService
	maxUploadBytes int64
}

type sessionUploadRequest struct {
	StartedAt string                  `json:"started_at"`
	Messages  []sessionMessageRequest `json:"messages"`
}

type sessionMessageRequest struct {
	Role    string `json:"role" binding:"required"`
	Content string `json:"content"`
}

func NewSessionUploadHandler(svc *session.UploadService, maxUploadBytes int64) *SessionUploadHandler {
	return &SessionUploadHandler{service: svc, maxUploadBytes: maxUploadBytes}
}

func (h *SessionUploadHandler) Upload(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		httperr.JSON(c, http.StatusBadRequest, "session_id is required")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes)

	var req sessionUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httperr.JSON(c, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		httperr.JSON(c, http.StatusBadRequest, err.Error())
		return
	}

	var startedAt *time.Time
	if s := strings.TrimSpace(req.StartedAt); s != "" {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			httperr.JSON(c, http.StatusBadRequest, "started_at must be RFC3339, e.g. 2026-05-09T17:00:00Z")
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
		StartedAt: startedAt,
		Messages:  messages,
	})
	if err != nil {
		httperr.Write(c, err)
		return
	}

	body := gin.H{
		"uri":           result.URI,
		"parent_uri":    result.ParentURI,
		"root_uri":      result.RootURI,
		"category":      result.Category,
		"message_count": result.MessageCnt,
		"archive_index": result.ArchiveIdx,
		"created_at":    result.CreatedAt,
		"updated_at":    result.UpdatedAt,
	}
	if result.TaskID != nil {
		body["task_id"] = result.TaskID.String()
		c.JSON(http.StatusAccepted, body)
		return
	}
	c.JSON(http.StatusOK, body)
}
