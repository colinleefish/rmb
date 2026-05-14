package service

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	TurnIDs          []uuid.UUID
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
		FROM session_turns
		WHERE turn_status IN (?, ?)
		GROUP BY session_id
		ORDER BY MIN(created_at)
		LIMIT ?
	`,
		model.SessionTurnStatusNotSummarized,
		model.SessionTurnStatusSummarizing,
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
	task, ok, err := w.claimNextTurns(ctx, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	merged, err := w.llm.MergeOverview(ctx, task.PreviousOverview, task.MessagesJSONL)
	if err != nil {
		log.Printf(
			"summarization worker llm merge failed session=%s turns=%d: %v",
			task.SessionID,
			len(task.TurnIDs),
			err,
		)
		if markErr := w.markTurnsFailed(ctx, task.SessionID, task.TurnIDs); markErr != nil {
			return fmt.Errorf("mark failed after llm error: %w", markErr)
		}
		return nil
	}
	return w.completeTurns(ctx, task.SessionID, task.TurnIDs, merged)
}

func (w *SummarizationWorker) claimNextTurns(
	ctx context.Context,
	sessionID uuid.UUID,
) (claimedSummarizationTask, bool, error) {
	now := w.now().UTC()
	staleBefore := now.Add(-w.config.StaleSummarizingAfter)
	maxTurns := w.config.MaxTurnsPerMerge
	if maxTurns <= 0 {
		maxTurns = 1
	}
	maxChars := w.config.MaxCharsPerMerge

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

		var pending []model.SessionTurn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("session_id = ? AND turn_status <> ?", session.ID, model.SessionTurnStatusSummarized).
			Order("id ASC").
			Limit(maxTurns).
			Find(&pending).Error; err != nil {
			return fmt.Errorf("load session head turns: %w", err)
		}
		if len(pending) == 0 {
			return nil
		}

		claimable := make([]model.SessionTurn, 0, len(pending))
		for _, turn := range pending {
			if !canClaimTurn(turn, staleBefore) {
				break
			}
			claimable = append(claimable, turn)
		}
		if len(claimable) == 0 {
			return nil
		}

		turnIDs := make([]uuid.UUID, 0, len(claimable))
		for _, turn := range claimable {
			turnIDs = append(turnIDs, turn.ID)
		}

		if err := tx.Model(&model.SessionTurn{}).
			Where("id IN ? AND session_id = ?", turnIDs, session.ID).
			Updates(map[string]any{
				"turn_status":          model.SessionTurnStatusSummarizing,
				"summarize_started_at": now,
			}).Error; err != nil {
			return fmt.Errorf("mark turns summarizing: %w", err)
		}

		mergedJSONL := mergeTurnMessagesJSONL(claimable, maxChars)
		if strings.TrimSpace(mergedJSONL) == "" {
			mergedJSONL = "(empty)\n"
		}

		task = claimedSummarizationTask{
			SessionID:        session.ID,
			TurnIDs:          turnIDs,
			MessagesJSONL:    mergedJSONL,
			PreviousOverview: derefString(session.OverviewText),
		}
		return nil
	})
	if err != nil {
		return claimedSummarizationTask{}, false, err
	}

	if len(task.TurnIDs) == 0 {
		return claimedSummarizationTask{}, false, nil
	}
	return task, true, nil
}

func mergeTurnMessagesJSONL(turns []model.SessionTurn, maxChars int) string {
	var out strings.Builder
	for _, turn := range turns {
		chunk := strings.TrimSpace(turn.MessagesJSONL)
		if chunk == "" {
			continue
		}
		next := chunk
		if out.Len() > 0 {
			next = "\n" + next
		}
		if maxChars > 0 && out.Len()+len(next) > maxChars {
			remaining := maxChars - out.Len()
			if remaining <= 0 {
				break
			}
			out.WriteString(next[:remaining])
			break
		}
		out.WriteString(next)
	}
	if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
		out.WriteString("\n")
	}
	return out.String()
}

func (w *SummarizationWorker) completeTurns(
	ctx context.Context,
	sessionID uuid.UUID,
	turnIDs []uuid.UUID,
	mergedOverview string,
) error {
	if len(turnIDs) == 0 {
		return nil
	}
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

		if err := tx.Model(&model.SessionTurn{}).
			Where("id IN ? AND session_id = ?", turnIDs, sessionID).
			Updates(map[string]any{
				"turn_status":          model.SessionTurnStatusSummarized,
				"summarize_started_at": nil,
			}).Error; err != nil {
			return fmt.Errorf("mark turns summarized: %w", err)
		}
		return nil
	})
}

func (w *SummarizationWorker) markTurnsFailed(
	ctx context.Context,
	sessionID uuid.UUID,
	turnIDs []uuid.UUID,
) error {
	if len(turnIDs) == 0 {
		return nil
	}
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := trySessionAdvisoryXactLock(tx, sessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		if err := tx.Model(&model.SessionTurn{}).
			Where("id IN ? AND session_id = ?", turnIDs, sessionID).
			Updates(map[string]any{
				"turn_status":          model.SessionTurnStatusFailed,
				"summarize_started_at": nil,
			}).Error; err != nil {
			return fmt.Errorf("mark turns failed: %w", err)
		}
		return nil
	})
}

func canClaimTurn(turn model.SessionTurn, staleBefore time.Time) bool {
	switch turn.TurnStatus {
	case model.SessionTurnStatusNotSummarized:
		return true
	case model.SessionTurnStatusSummarizing:
		if turn.SummarizeStartedAt == nil {
			return true
		}
		return turn.SummarizeStartedAt.Before(staleBefore)
	case model.SessionTurnStatusFailed:
		// Cost guard: failed turns require manual intervention/reset.
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
