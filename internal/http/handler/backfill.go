package handler

import (
	"net/http"

	"github.com/colinleefish/mypast/internal/service/extract"
	"github.com/colinleefish/mypast/internal/service/memory"
	"github.com/colinleefish/mypast/internal/service/scene"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BackfillHandler exposes manual pipeline re-queuing over HTTP so that the CLI
// can trigger backfills against a remote server without a direct DB connection.
type BackfillHandler struct {
	db *gorm.DB
}

func NewBackfillHandler(db *gorm.DB) *BackfillHandler {
	return &BackfillHandler{db: db}
}

// T1 enqueues T1 atom extraction. With ?session=<key> it targets a single
// session; without it, all sessions that have unprocessed turns are enqueued.
func (h *BackfillHandler) T1(c *gin.Context) {
	ctx := c.Request.Context()
	sessionKey := c.Query("session")

	if sessionKey != "" {
		if err := extract.EnqueueSessionByKey(ctx, h.db, sessionKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enqueued": 1, "session": sessionKey})
		return
	}

	type row struct {
		SessionID uuid.UUID
	}
	var rows []row
	if err := h.db.WithContext(ctx).Raw(`
		SELECT DISTINCT session_id
		FROM session_turns
		WHERE t1_extracted_at IS NULL
	`).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, r := range rows {
		if err := extract.EnqueueSession(ctx, h.db, r.SessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"enqueued": len(rows)})
}

// T2 enqueues T2 scene building. With ?session=<key> it targets a single
// session; without it, all sessions that have atoms are enqueued.
func (h *BackfillHandler) T2(c *gin.Context) {
	ctx := c.Request.Context()
	sessionKey := c.Query("session")

	if sessionKey != "" {
		if err := scene.EnqueueSessionByKey(ctx, h.db, sessionKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enqueued": 1, "session": sessionKey})
		return
	}

	count, err := scene.EnqueueAllSessionsWithAtoms(ctx, h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enqueued": count})
}

// T3 enqueues T3 memory rollup. With ?session=<key> it targets a single
// session; without it, all sessions that have scenes are enqueued.
func (h *BackfillHandler) T3(c *gin.Context) {
	ctx := c.Request.Context()
	sessionKey := c.Query("session")

	if sessionKey != "" {
		if err := memory.EnqueueSessionByKey(ctx, h.db, sessionKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"enqueued": 1, "session": sessionKey})
		return
	}

	count, err := memory.EnqueueAllSessionsWithScenes(ctx, h.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enqueued": count})
}
