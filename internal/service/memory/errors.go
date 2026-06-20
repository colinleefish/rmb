package memory

import (
	"context"
	"time"

	"github.com/colinleefish/rmb/internal/llm"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/google/uuid"
)

// isTransient reports whether a rollup error is a temporary LLM failure (rate
// limit / timeout). Transient failures leave sessions pending for retry rather
// than draining them, so no memory is silently skipped.
func isTransient(err error) bool {
	return llm.IsTransientError(err)
}

// markSessionsIdle clears t3_status for sessions whose rollup completed.
func (w *Worker) markSessionsIdle(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return w.db.WithContext(ctx).Model(&model.PipelineState{}).
		Where("session_id IN ? AND t3_status = ?", ids, model.PipelineStatusPending).
		Updates(map[string]any{
			"t3_status":      model.PipelineStatusIdle,
			"t3_advanced_at": now,
		}).Error
}
