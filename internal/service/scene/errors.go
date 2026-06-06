package scene

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/colinleefish/mypast/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func isTransientLLMError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "http 429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "速率限制") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "client.timeout") ||
		errors.Is(err, context.DeadlineExceeded)
}

func (w *Worker) handleProcessError(ctx context.Context, sessionID uuid.UUID, cause error) error {
	if isTransientLLMError(cause) {
		log.Printf("t2 worker transient error session=%s: %v", sessionID, cause)
		_ = w.markSessionPending(ctx, sessionID)
		return nil
	}
	return w.markSessionFailed(ctx, sessionID, cause)
}

func (w *Worker) markSessionPending(ctx context.Context, sessionID uuid.UUID) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.PipelineState{}).
			Where("session_id = ?", sessionID).
			Update("t2_status", model.PipelineStatusPending).Error
	})
}
