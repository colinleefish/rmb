package memory

import (
	"context"
	"fmt"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EnqueueSessionsForMemoryTargets marks T3 pending for the sessions that own the
// memories targeted by a human correction, so the next rollup re-distills those
// buckets and bakes the correction into the body (write-time injection). Targets
// that are not memories (or have no materialized memory yet) resolve to no
// sessions. Returns the number of sessions enqueued.
func EnqueueSessionsForMemoryTargets(ctx context.Context, gdb *gorm.DB, targetURIs []string) (int, error) {
	if len(targetURIs) == 0 {
		return 0, nil
	}
	type row struct{ SessionID uuid.UUID }
	var rows []row
	if err := gdb.WithContext(ctx).Raw(`
		SELECT DISTINCT ps.session_id
		FROM memories m
		JOIN scenes sc ON ('rmb://scenes/' || lower(sc.id::text)) = ANY(m.source_scene_uris)
		JOIN pipeline_state ps ON ps.session_id = sc.session_id
		WHERE m.superseded_at IS NULL AND m.uri = ANY(?)
	`, pgarray.TextArray(targetURIs)).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("resolve correction target sessions: %w", err)
	}
	for _, r := range rows {
		if err := enqueueSession(ctx, gdb, r.SessionID); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}

func enqueueSession(ctx context.Context, gdb *gorm.DB, sessionID uuid.UUID) error {
	return gdb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ps model.PipelineState
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_id = ?", sessionID).
			Take(&ps).Error
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("pipeline_state not found for session %s", sessionID)
		}
		if err != nil {
			return err
		}
		return tx.Model(&ps).Update("t3_status", model.PipelineStatusPending).Error
	})
}
