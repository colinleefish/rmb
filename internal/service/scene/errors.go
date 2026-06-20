package scene

import (
	"context"
	"log"

	"github.com/colinleefish/mem9/internal/llm"
	"github.com/colinleefish/mem9/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (w *Worker) handleProcessError(ctx context.Context, sessionID uuid.UUID, cause error) error {
	if llm.IsTransientError(cause) {
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
