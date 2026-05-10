package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OverviewMerger interface {
	MergeOverview(ctx context.Context, previousOverview string, messagesJSONL string) (string, error)
}

type SummarizationWorker struct {
	db     *gorm.DB
	llm    OverviewMerger
	config config.SummarizerConfig
	now    func() time.Time
}

type claimedSummarizationTask struct {
	SessionID        uuid.UUID
	RecordID         uuid.UUID
	PreviousOverview string
	MessagesJSONL    string
}

func NewSummarizationWorker(
	db *gorm.DB,
	llmClient OverviewMerger,
	cfg config.SummarizerConfig,
) *SummarizationWorker {
	return &SummarizationWorker{
		db:     db,
		llm:    llmClient,
		config: cfg,
		now:    time.Now,
	}
}

func (w *SummarizationWorker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("summarization worker disabled")
		return nil
	}
	if w.llm == nil {
		return fmt.Errorf("summarization worker requires llm client")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid poll interval: %s", w.config.PollInterval)
	}

	log.Printf(
		"summarization worker started poll_interval=%s batch_size=%d stale_after=%s",
		w.config.PollInterval,
		w.config.BatchSize,
		w.config.StaleSummarizingAfter,
	)

	// Run one cycle immediately on startup.
	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("summarization worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

func (w *SummarizationWorker) runOneCycle(ctx context.Context) {
	sessionIDs, err := w.selectCandidateSessions(ctx)
	if err != nil {
		log.Printf("summarization worker select candidates failed: %v", err)
		return
	}
	for _, sessionID := range sessionIDs {
		if err := w.processSession(ctx, sessionID); err != nil {
			log.Printf("summarization worker process session %s failed: %v", sessionID, err)
		}
	}
}

func (w *SummarizationWorker) selectCandidateSessions(ctx context.Context) ([]uuid.UUID, error) {
	limit := w.config.BatchSize
	if limit <= 0 {
		limit = 8
	}

	type row struct {
		SessionID uuid.UUID
	}
	rows := make([]row, 0, limit)
	if err := w.db.WithContext(ctx).Raw(`
		SELECT session_id
		FROM session_records
		WHERE record_status IN (?, ?)
		GROUP BY session_id
		ORDER BY MIN(created_at)
		LIMIT ?
	`,
		model.SessionRecordStatusNotSummarized,
		model.SessionRecordStatusSummarizing,
		limit,
	).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query candidate sessions: %w", err)
	}

	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.SessionID)
	}
	return out, nil
}

func (w *SummarizationWorker) processSession(ctx context.Context, sessionID uuid.UUID) error {
	task, ok, err := w.claimNextRecord(ctx, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	merged, err := w.llm.MergeOverview(ctx, task.PreviousOverview, task.MessagesJSONL)
	if err != nil {
		log.Printf(
			"summarization worker llm merge failed session=%s record=%s: %v",
			task.SessionID,
			task.RecordID,
			err,
		)
		if markErr := w.markRecordFailed(ctx, task.SessionID, task.RecordID); markErr != nil {
			return fmt.Errorf("mark failed after llm error: %w", markErr)
		}
		return nil
	}
	return w.completeRecord(ctx, task.SessionID, task.RecordID, merged)
}

func (w *SummarizationWorker) claimNextRecord(
	ctx context.Context,
	sessionID uuid.UUID,
) (claimedSummarizationTask, bool, error) {
	now := w.now().UTC()
	staleBefore := now.Add(-w.config.StaleSummarizingAfter)

	task := claimedSummarizationTask{}
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := trySessionAdvisoryXactLock(tx, sessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		var session model.Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", sessionID).
			Take(&session).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return fmt.Errorf("load session for claim: %w", err)
		}

		var head model.SessionRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("session_id = ? AND record_status <> ?", session.ID, model.SessionRecordStatusSummarized).
			Order("record_index ASC").
			Take(&head).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return fmt.Errorf("load session head record: %w", err)
		}

		if !canClaimRecord(head, staleBefore) {
			return nil
		}

		if err := tx.Model(&model.SessionRecord{}).
			Where("id = ?", head.ID).
			Updates(map[string]any{
				"record_status":        model.SessionRecordStatusSummarizing,
				"summarize_started_at": now,
			}).Error; err != nil {
			return fmt.Errorf("mark record summarizing: %w", err)
		}

		task = claimedSummarizationTask{
			SessionID:        session.ID,
			RecordID:         head.ID,
			MessagesJSONL:    head.MessagesJSONL,
			PreviousOverview: derefString(session.OverviewText),
		}
		return nil
	})
	if err != nil {
		return claimedSummarizationTask{}, false, err
	}

	if task.RecordID == uuid.Nil {
		return claimedSummarizationTask{}, false, nil
	}
	return task, true, nil
}

func (w *SummarizationWorker) completeRecord(
	ctx context.Context,
	sessionID uuid.UUID,
	recordID uuid.UUID,
	mergedOverview string,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := trySessionAdvisoryXactLock(tx, sessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		if err := tx.Model(&model.Session{}).
			Where("id = ?", sessionID).
			Update("overview_text", mergedOverview).Error; err != nil {
			return fmt.Errorf("update session overview: %w", err)
		}

		if err := tx.Model(&model.SessionRecord{}).
			Where("id = ? AND session_id = ?", recordID, sessionID).
			Updates(map[string]any{
				"record_status":        model.SessionRecordStatusSummarized,
				"summarize_started_at": nil,
			}).Error; err != nil {
			return fmt.Errorf("mark record summarized: %w", err)
		}
		return nil
	})
}

func (w *SummarizationWorker) markRecordFailed(
	ctx context.Context,
	sessionID uuid.UUID,
	recordID uuid.UUID,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := trySessionAdvisoryXactLock(tx, sessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		if err := tx.Model(&model.SessionRecord{}).
			Where("id = ? AND session_id = ?", recordID, sessionID).
			Updates(map[string]any{
				"record_status":        model.SessionRecordStatusFailed,
				"summarize_started_at": nil,
			}).Error; err != nil {
			return fmt.Errorf("mark record failed: %w", err)
		}
		return nil
	})
}

func canClaimRecord(record model.SessionRecord, staleBefore time.Time) bool {
	switch record.RecordStatus {
	case model.SessionRecordStatusNotSummarized:
		return true
	case model.SessionRecordStatusSummarizing:
		if record.SummarizeStartedAt == nil {
			return true
		}
		return record.SummarizeStartedAt.Before(staleBefore)
	case model.SessionRecordStatusFailed:
		// Cost guard: failed records require manual intervention/reset.
		return false
	default:
		return false
	}
}

func trySessionAdvisoryXactLock(tx *gorm.DB, sessionID uuid.UUID) (bool, error) {
	var locked bool
	if err := tx.Raw(
		`SELECT pg_try_advisory_xact_lock(hashtextextended(CAST(? AS text), 0))`,
		sessionID.String(),
	).Scan(&locked).Error; err != nil {
		return false, fmt.Errorf("acquire advisory lock: %w", err)
	}
	return locked, nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
