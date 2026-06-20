package scene

import (
	"time"

	"github.com/colinleefish/rmb/internal/model"
)

// shouldRunT2 decides whether to run T2 for a session on this worker tick.
func shouldRunT2(
	now time.Time,
	t1Status string,
	t2Status string,
	t1AdvancedAt *time.Time,
	delayAfterT1 time.Duration,
) bool {
	if t1Status == model.PipelineStatusRunning {
		return false
	}
	if t2Status != model.PipelineStatusPending && t2Status != model.PipelineStatusFailed {
		return false
	}
	if t1AdvancedAt == nil {
		return false
	}
	if delayAfterT1 > 0 && now.Sub(t1AdvancedAt.UTC()) < delayAfterT1 {
		return false
	}
	return true
}
