package scene

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
	"github.com/colinleefish/rmb/internal/service/session"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SceneBuilder interface {
	BuildScenes(ctx context.Context, atomsJSON string) (string, error)
	SummarizeSessionAbstract(ctx context.Context, sceneAbstracts string) (string, error)
}

type Worker struct {
	db     *gorm.DB
	llm    SceneBuilder
	config config.SceneConfig
	now    func() time.Time
}

type sessionCandidate struct {
	SessionID uuid.UUID
}

type sceneBatch struct {
	SessionKey string
	SessionID  uuid.UUID
	Atoms      []model.Atom
}

func NewWorker(db *gorm.DB, llm SceneBuilder, cfg config.SceneConfig) *Worker {
	return &Worker{
		db:     db,
		llm:    llm,
		config: cfg,
		now:    time.Now,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("t2 scene worker disabled")
		return nil
	}
	if w.llm == nil {
		return fmt.Errorf("t2 scene worker requires llm client")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid scene poll interval: %s", w.config.PollInterval)
	}

	log.Printf(
		"t2 scene worker started poll_interval=%s delay_after_t1=%s",
		w.config.PollInterval,
		w.config.DelayAfterT1,
	)

	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("t2 scene worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

func (w *Worker) runOneCycle(ctx context.Context) {
	ids, err := w.selectCandidateSessions(ctx)
	if err != nil {
		log.Printf("t2 worker select candidates failed: %v", err)
		return
	}
	for _, id := range ids {
		if err := w.processSession(ctx, id); err != nil {
			log.Printf("t2 worker process session %s failed: %v", id, err)
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
		SELECT ps.session_id
		FROM pipeline_state ps
		WHERE ps.t2_status IN ('pending', 'failed')
		  AND ps.t1_status != 'running'
		ORDER BY ps.updated_at
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

func (w *Worker) processSession(ctx context.Context, sessionID uuid.UUID) error {
	batch, err := w.prepareBatch(ctx, sessionID)
	if err != nil || batch == nil {
		return err
	}

	groups := groupAtomsBySceneName(batch.Atoms)
	chunks := chunkGroups(groups, w.config.MaxAtomsPerBatch)
	validURIs := atomURISet(batch.Atoms)

	var parsed []parsedScene
	for _, chunk := range chunks {
		atomsJSON, err := serializeAtomsForLLM(chunk)
		if err != nil {
			return w.handleProcessError(ctx, sessionID, err)
		}

		raw, err := w.llm.BuildScenes(ctx, atomsJSON)
		if err != nil {
			return w.handleProcessError(ctx, sessionID, fmt.Errorf("llm build scenes: %w", err))
		}

		chunkScenes, err := parseBuildScenesResponse(raw, validURIs)
		if err != nil {
			return w.handleProcessError(ctx, sessionID, fmt.Errorf("parse build scenes: %w", err))
		}
		parsed = append(parsed, chunkScenes...)
	}

	abstract, err := w.llm.SummarizeSessionAbstract(ctx, joinSceneAbstracts(parsed))
	if err != nil {
		return w.handleProcessError(ctx, sessionID, fmt.Errorf("llm session abstract: %w", err))
	}

	if err := w.persistScenes(ctx, batch, parsed, abstract); err != nil {
		return w.handleProcessError(ctx, sessionID, err)
	}
	return nil
}

func (w *Worker) prepareBatch(ctx context.Context, sessionID uuid.UUID) (*sceneBatch, error) {
	var out *sceneBatch
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

		var ps model.PipelineState
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_id = ?", sessionID).
			Take(&ps).Error; err != nil {
			return fmt.Errorf("load pipeline_state: %w", err)
		}

		if !shouldRunT2(
			w.now().UTC(),
			ps.T1Status,
			ps.T2Status,
			ps.T1AdvancedAt,
			w.config.DelayAfterT1,
		) {
			return nil
		}

		var atoms []model.Atom
		if err := tx.Where("session_id = ?", sessionID).
			Order("created_at asc, id asc").
			Find(&atoms).Error; err != nil {
			return fmt.Errorf("load atoms: %w", err)
		}
		if len(atoms) == 0 {
			if err := tx.Model(&ps).Updates(map[string]any{
				"t2_status": model.PipelineStatusIdle,
			}).Error; err != nil {
				return fmt.Errorf("clear t2 pending without atoms: %w", err)
			}
			return nil
		}

		if err := tx.Model(&ps).Update("t2_status", model.PipelineStatusRunning).Error; err != nil {
			return fmt.Errorf("mark t2 running: %w", err)
		}

		out = &sceneBatch{
			SessionKey: session.SessionKey,
			SessionID:  sessionID,
			Atoms:      atoms,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (w *Worker) persistScenes(
	ctx context.Context,
	batch *sceneBatch,
	scenes []parsedScene,
	sessionAbstract string,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		locked, err := db.TrySessionAdvisoryXactLock(tx, batch.SessionID)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}

		now := w.now().UTC()
		dupCount := make(map[string]int, len(scenes))
		keepIDs := make([]uuid.UUID, 0, len(scenes))
		for _, s := range scenes {
			nameKey := strings.ToLower(strings.TrimSpace(s.DisplayName))
			dupCount[nameKey]++
			sceneID := sceneIDForName(batch.SessionID, s.DisplayName, dupCount[nameKey])
			keepIDs = append(keepIDs, sceneID)

			displayName := s.DisplayName
			abstract := s.Abstract
			body := s.Body
			row := model.Scene{
				ID:          sceneID,
				SessionID:   batch.SessionID,
				DisplayName: &displayName,
				Abstract:    &abstract,
				Body:        &body,
				SourceAtoms: pgarray.UUIDArray(append([]uuid.UUID(nil), s.SourceAtoms...)),
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			// Stable id: reuse the existing row (preserve created_at), refresh content.
			// Reset embedding so the embed worker re-embeds the changed abstract/body.
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.Assignments(map[string]any{
					"display_name":     gorm.Expr("EXCLUDED.display_name"),
					"abstract":         gorm.Expr("EXCLUDED.abstract"),
					"body":             gorm.Expr("EXCLUDED.body"),
					"source_atoms": gorm.Expr("EXCLUDED.source_atoms"),
					"updated_at":       gorm.Expr("EXCLUDED.updated_at"),
					"embedding":        gorm.Expr("NULL"),
				}),
			}).Create(&row).Error; err != nil {
				return fmt.Errorf("upsert scene: %w", err)
			}
		}

		// Prune scenes for this session whose name no longer appears.
		prune := tx.Where("session_id = ?", batch.SessionID)
		if len(keepIDs) > 0 {
			prune = prune.Where("id NOT IN ?", keepIDs)
		}
		if err := prune.Delete(&model.Scene{}).Error; err != nil {
			return fmt.Errorf("prune stale scenes: %w", err)
		}

		abstract := sessionAbstract
		if err := tx.Model(&model.Session{}).
			Where("id = ?", batch.SessionID).
			Update("abstract", abstract).Error; err != nil {
			return fmt.Errorf("update session abstract: %w", err)
		}

		if err := tx.Model(&model.PipelineState{}).
			Where("session_id = ?", batch.SessionID).
			Updates(map[string]any{
				"t2_status":      model.PipelineStatusIdle,
				"t2_advanced_at": now,
				"t3_status":      model.PipelineStatusPending,
			}).Error; err != nil {
			return fmt.Errorf("update pipeline_state: %w", err)
		}

		log.Printf(
			"t2 worker built scenes session=%s count=%d",
			batch.SessionKey,
			len(scenes),
		)
		return nil
	})
}

func (w *Worker) markSessionFailed(ctx context.Context, sessionID uuid.UUID, cause error) error {
	_ = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_ = tx.Model(&model.PipelineState{}).
			Where("session_id = ?", sessionID).
			Update("t2_status", model.PipelineStatusFailed).Error
		return nil
	})
	return cause
}

// EnqueueSession marks a session for T2 (used by backfill CLI).
func EnqueueSession(ctx context.Context, gdb *gorm.DB, sessionID uuid.UUID) error {
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
		return tx.Model(&ps).Updates(map[string]any{
			"t2_status": model.PipelineStatusPending,
		}).Error
	})
}

// EnqueueSessionByKey resolves session_key to id and enqueues T2.
func EnqueueSessionByKey(ctx context.Context, gdb *gorm.DB, sessionKey string) error {
	sessionKey, err := session.ValidateSessionKey(sessionKey)
	if err != nil {
		return err
	}
	var s model.Session
	if err := gdb.WithContext(ctx).Where("session_key = ?", sessionKey).Take(&s).Error; err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	return EnqueueSession(ctx, gdb, s.ID)
}

// EnqueueAllSessionsWithAtoms enqueues T2 for every session that has atoms.
func EnqueueAllSessionsWithAtoms(ctx context.Context, gdb *gorm.DB) (int, error) {
	type row struct {
		SessionID uuid.UUID
	}
	var rows []row
	if err := gdb.WithContext(ctx).Raw(`
		SELECT DISTINCT a.session_id
		FROM atoms a
		JOIN pipeline_state ps ON ps.session_id = a.session_id
		WHERE ps.t1_status != 'running'
	`).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("list sessions with atoms: %w", err)
	}
	for _, row := range rows {
		if err := EnqueueSession(ctx, gdb, row.SessionID); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}
