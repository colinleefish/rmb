package extract

import (
	"time"

	"github.com/colinleefish/rmb/internal/model"
)

// shouldRunT1 decides whether to run T1 for a session on this worker tick.
func shouldRunT1(
	now time.Time,
	t1Status string,
	unprocessedTurns int,
	turnsSinceAdvanced int,
	warmupThreshold int,
	everyN int,
	warmupEnabled bool,
	idleSeconds time.Duration,
	lastTurnAt time.Time,
) bool {
	if unprocessedTurns <= 0 {
		return false
	}
	// Retry after a failed extraction once turns are still unprocessed.
	if t1Status == model.PipelineStatusPending || t1Status == model.PipelineStatusFailed {
		return true
	}

	threshold := everyN
	if everyN <= 0 {
		threshold = 8
	}
	if warmupEnabled && warmupThreshold > 0 && warmupThreshold < threshold {
		threshold = warmupThreshold
	}

	if turnsSinceAdvanced >= threshold {
		return true
	}

	if idleSeconds > 0 && !lastTurnAt.IsZero() {
		if now.Sub(lastTurnAt) >= idleSeconds {
			return true
		}
	}
	return false
}

// nextWarmupThreshold doubles the ramp cap until everyN.
func nextWarmupThreshold(current, everyN int, warmupEnabled bool) int {
	if !warmupEnabled || everyN <= 0 {
		return everyN
	}
	if current <= 0 {
		return 2
	}
	next := current * 2
	if next > everyN {
		return everyN
	}
	return next
}
