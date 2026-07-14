package extract

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/config"
	"github.com/colinleefish/rmb/internal/db"
	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AtomExtractor interface {
	ExtractAtoms(ctx context.Context, messagesJSONL string) (string, error)
}

type Worker struct {
	db     *gorm.DB
	llm    AtomExtractor
	config config.ExtractionConfig
	now    func() time.Time
}

type sessionCandidate struct {
	SessionID uuid.UUID
}

func NewWorker(db *gorm.DB, llm AtomExtractor, cfg config.ExtractionConfig) *Worker {
	return &Worker{
		db:     db,
		llm:    llm,
		config: cfg,
		now:    time.Now,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("t1 extraction worker disabled")
		return nil
	}
	if w.llm == nil {
		return fmt.Errorf("t1 extraction worker requires llm client")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid extraction poll interval: %s", w.config.PollInterval)
	}

	log.Printf(
		"t1 extraction worker started poll_interval=%s every_n=%d idle=%s warmup=%v",
		w.config.PollInterval,
		w.config.EveryN,
		w.config.IdleSeconds,
		w.config.Warmup,
	)

	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("t1 extraction worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

func (w *Worker) runOneCycle(ctx context.Context) {
	ids, err := w.selectCandidateSessions(ctx)
	if err != nil {
		log.Printf("t1 worker select candidates failed: %v", err)
		return
	}
	for _, id := range ids {
		if err := w.processSession(ctx, id); err != nil {
			log.Printf("t1 worker process session %s failed: %v", id, err)
		}
	}
}

func (w *Worker) selectCandidateSessions(ctx context.Context) ([]uuid.UUID, error) {
	limit := w.config.BatchSessions
	if limit <= 0 {
		limit = 8
	}

	rows := make([]sessionCandidate, 0, limit)
	if err := w.db.WithContext(ctx).Raw(`
		SELECT DISTINCT st.session_id
		FROM session_turns st
		WHERE st.t1_extracted_at IS NULL
		ORDER BY st.session_id
		LIMIT ?
	`, limit).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query candidate sessions: %w", err)
	}

	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.SessionID)
	}
	return out, nil
}

type extractBatch struct {
	SessionKey     string
	SessionID      uuid.UUID
	Turns          []model.SessionTurn
	MessagesJSONL  string
	WarmupThreshold int
}

func (w *Worker) processSession(ctx context.Context, sessionID uuid.UUID) error {
	batch, err := w.prepareBatch(ctx, sessionID)
	if err != nil || batch == nil {
		return err
	}

	raw, err := w.llm.ExtractAtoms(ctx, batch.MessagesJSONL)
	if err != nil {
		return w.handleProcessError(ctx, sessionID, fmt.Errorf("llm extract: %w", err))
	}

	parsed, err := parseExtractResponse(raw)
	if err != nil {
		return w.handleProcessError(ctx, sessionID, fmt.Errorf("parse extract response: %w", err))
	}

	if err := w.persistBatch(ctx, sessionID, batch, parsed); err != nil {
		return w.handleProcessError(ctx, sessionID, err)
	}
	return nil
}

func (w *Worker) prepareBatch(ctx context.Context, sessionID uuid.UUID) (*extractBatch, error) {
	var out *extractBatch
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := db.TrySessionAdvisoryXactLock(tx, sessionID)
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
			return fmt.Errorf("load session: %w", err)
		}

		ps, err := w.loadOrCreatePipeline(tx, sessionID)
		if err != nil {
			return err
		}

		var turns []model.SessionTurn
		if err := tx.Where("session_id = ? AND t1_extracted_at IS NULL", sessionID).
			Order("id ASC").
			Limit(w.maxTurnsPerBatch()).
			Find(&turns).Error; err != nil {
			return fmt.Errorf("load unextracted turns: %w", err)
		}
		if len(turns) == 0 {
			if ps.T1Status == model.PipelineStatusPending {
				return tx.Model(&ps).Updates(map[string]any{
					"t1_status":               model.PipelineStatusIdle,
					"t1_turns_since_advanced": 0,
				}).Error
			}
			return nil
		}

		lastTurnAt := turns[len(turns)-1].CreatedAt
		if !shouldRunT1(
			w.now().UTC(),
			ps.T1Status,
			len(turns),
			ps.T1TurnsSinceAdvanced,
			ps.WarmupThreshold,
			w.config.EveryN,
			w.config.Warmup,
			w.config.IdleSeconds,
			lastTurnAt,
		) {
			return nil
		}

		if err := tx.Model(&ps).Update("t1_status", model.PipelineStatusRunning).Error; err != nil {
			return fmt.Errorf("mark t1 running: %w", err)
		}

		out = &extractBatch{
			SessionKey:      session.SessionKey,
			SessionID:       sessionID,
			Turns:           turns,
			MessagesJSONL:   mergeTurnMessagesJSONL(turns, w.config.MaxCharsPerBatch),
			WarmupThreshold: ps.WarmupThreshold,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (w *Worker) persistBatch(
	ctx context.Context,
	sessionID uuid.UUID,
	batch *extractBatch,
	parsed []llmAtom,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := db.TrySessionAdvisoryXactLock(tx, sessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		turnIndex := buildTurnIndex(batch.Turns)
		now := w.now().UTC()
		var firstAtomID uuid.UUID

		for _, a := range parsed {
			atomID, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate atom id: %w", err)
			}
			sourceIDs, err := resolveSourceTurnIDs(a.SourceTurnIndices, turnIndex)
			if err != nil {
				log.Printf("t1 worker atom source fallback session=%s: %v", batch.SessionKey, err)
				sourceIDs = []uuid.UUID{batch.Turns[0].ID}
			}

			var slugPtr *string
			if a.Slug != "" && a.Category != model.AtomCategoryProfile {
				sanitized, err := uri.SanitizeSlug(a.Slug)
				if err == nil {
					slugPtr = &sanitized
				}
			}

			sceneName := strings.TrimSpace(a.SceneName)
			var scenePtr *string
			if sceneName != "" {
				scenePtr = &sceneName
			}

			priority := a.Priority
			if priority == 0 {
				priority = 50
			}

			if firstAtomID == uuid.Nil {
				firstAtomID = atomID
			}

			row := model.Atom{
				ID:            atomID,
				SessionID:     sessionID,
				Category:      a.Category,
				Priority:      priority,
				SceneName:     scenePtr,
				Slug:          slugPtr,
				Content:       a.Content,
				SourceTurnIDs: pgarray.UUIDArray(sourceIDs),
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("insert atom: %w", err)
			}
		}

		turnIDs := make([]uuid.UUID, 0, len(batch.Turns))
		for _, t := range batch.Turns {
			turnIDs = append(turnIDs, t.ID)
		}
		if err := tx.Model(&model.SessionTurn{}).
			Where("id IN ? AND t1_extracted_at IS NULL", turnIDs).
			Update("t1_extracted_at", now).Error; err != nil {
			return fmt.Errorf("mark turns extracted: %w", err)
		}

		nextWarmup := nextWarmupThreshold(batch.WarmupThreshold, w.config.EveryN, w.config.Warmup)
		if err := tx.Model(&model.PipelineState{}).
			Where("session_id = ?", sessionID).
			Updates(map[string]any{
				"t1_status":               model.PipelineStatusIdle,
				"t1_advanced_at":          now,
				"t1_turns_since_advanced": 0,
				"warmup_threshold":        nextWarmup,
				"t2_status":               model.PipelineStatusPending,
			}).Error; err != nil {
			return fmt.Errorf("update pipeline_state: %w", err)
		}

		resultURI := ""
		if firstAtomID != uuid.Nil {
			resultURI = uri.BuildAtom(firstAtomID.String())
		}
		if err := w.completePendingTasks(tx, sessionID, resultURI); err != nil {
			return err
		}

		log.Printf(
			"t1 worker extracted session=%s turns=%d atoms=%d",
			batch.SessionKey,
			len(batch.Turns),
			len(parsed),
		)
		return nil
	})
}

func (w *Worker) loadOrCreatePipeline(tx *gorm.DB, sessionID uuid.UUID) (model.PipelineState, error) {
	var ps model.PipelineState
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("session_id = ?", sessionID).
		Take(&ps).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			ps = model.PipelineState{
				SessionID:       sessionID,
				T1Status:        model.PipelineStatusIdle,
				T2Status:        model.PipelineStatusIdle,
				T3Status:        model.PipelineStatusIdle,
				WarmupThreshold: 2,
			}
			if err := tx.Create(&ps).Error; err != nil {
				return model.PipelineState{}, fmt.Errorf("create pipeline_state: %w", err)
			}
			return ps, nil
		}
		return model.PipelineState{}, fmt.Errorf("load pipeline_state: %w", err)
	}
	return ps, nil
}

func (w *Worker) markSessionFailed(ctx context.Context, sessionID uuid.UUID, cause error) error {
	_ = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&model.PipelineState{}).
			Where("session_id = ?", sessionID).
			Update("t1_status", model.PipelineStatusFailed).Error
		_ = w.failPendingTasks(tx, sessionID, cause.Error())
		return nil
	})
	return cause
}

func (w *Worker) maxTurnsPerBatch() int {
	if w.config.MaxTurnsPerBatch > 0 {
		return w.config.MaxTurnsPerBatch
	}
	return 8
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

func buildTurnIndex(turns []model.SessionTurn) map[int]uuid.UUID {
	idx := make(map[int]uuid.UUID, len(turns))
	for i, t := range turns {
		idx[i] = t.ID
	}
	return idx
}

func resolveSourceTurnIDs(indices []int, turnIndex map[int]uuid.UUID) ([]uuid.UUID, error) {
	if len(indices) == 0 {
		return nil, fmt.Errorf("no source_turn_indices")
	}
	out := make([]uuid.UUID, 0, len(indices))
	seen := make(map[uuid.UUID]struct{}, len(indices))
	for _, i := range indices {
		id, ok := turnIndex[i]
		if !ok && i > 0 {
			// Some models use 1-based indices despite the 0-based prompt.
			id, ok = turnIndex[i-1]
		}
		if !ok {
			return nil, fmt.Errorf("invalid turn index %d", i)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// completePendingTasks marks all outstanding T1 tasks for the session done, including
// previously failed rows, so the UI does not show stale failures after a later success.
func (w *Worker) completePendingTasks(tx *gorm.DB, sessionID uuid.UUID, resultURI string) error {
	updates := map[string]any{
		"status":    model.TaskStatusDone,
		"progress":  100,
		"error":     nil,
	}
	if resultURI != "" {
		updates["result_uri"] = resultURI
	}
	return tx.Model(&model.Task{}).
		Where("session_id = ? AND kind = ? AND status IN ?",
			sessionID, model.TaskKindT1, []string{
				model.TaskStatusPending,
				model.TaskStatusRunning,
				model.TaskStatusFailed,
			}).
		Updates(updates).Error
}

func (w *Worker) failPendingTasks(tx *gorm.DB, sessionID uuid.UUID, errMsg string) error {
	return tx.Model(&model.Task{}).
		Where("session_id = ? AND kind = ? AND status IN ?",
			sessionID, model.TaskKindT1, []string{model.TaskStatusPending, model.TaskStatusRunning}).
		Updates(map[string]any{
			"status": model.TaskStatusFailed,
			"error":  errMsg,
		}).Error
}
