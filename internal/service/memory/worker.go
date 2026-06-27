package memory

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
	"github.com/colinleefish/rmb/internal/service/correction"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// splitCorrections separates active corrections (newest-first, as returned by
// correction.ForTargets) into their statements (for the distiller) and their URIs
// (for the provenance gate).
func splitCorrections(sums []correction.Summary) (statements, uris []string) {
	statements = make([]string, 0, len(sums))
	uris = make([]string, 0, len(sums))
	for _, s := range sums {
		if st := strings.TrimSpace(s.Statement); st != "" {
			statements = append(statements, st)
		}
		uris = append(uris, s.URI)
	}
	return statements, uris
}

const (
	globalLockKey = "rmb-t3-rollup"
	distillDelay  = 1 * time.Second
)

type MemoryDistiller interface {
	DistillMemory(ctx context.Context, category, slug, atomsJSON string, corrections []string) (string, error)
}

type Worker struct {
	db     *gorm.DB
	llm    MemoryDistiller
	config config.MemoryConfig
	now    func() time.Time
}

func NewWorker(database *gorm.DB, llm MemoryDistiller, cfg config.MemoryConfig) *Worker {
	return &Worker{
		db:     database,
		llm:    llm,
		config: cfg,
		now:    time.Now,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("t3 memory worker disabled")
		return nil
	}
	if w.llm == nil {
		return fmt.Errorf("t3 memory worker requires llm client")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid memory poll interval: %s", w.config.PollInterval)
	}

	log.Printf("t3 memory worker started poll_interval=%s", w.config.PollInterval)

	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("t3 memory worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

// runOneCycle holds a single global advisory lock on a pinned connection so only
// one rollup runs at a time, without keeping a long-running transaction open
// during LLM calls.
func (w *Worker) runOneCycle(ctx context.Context) {
	err := w.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		locked, err := db.TryGlobalAdvisoryLock(conn, globalLockKey)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}
		defer func() { _ = db.GlobalAdvisoryUnlock(conn, globalLockKey) }()
		return w.rollup(ctx)
	})
	if err != nil {
		log.Printf("t3 worker cycle failed: %v", err)
	}
}

func (w *Worker) rollup(ctx context.Context) error {
	pendingIDs, err := w.pendingSessionIDs(ctx)
	if err != nil {
		return err
	}
	if len(pendingIDs) == 0 {
		return nil
	}

	var atoms []model.Atom
	if err := w.db.WithContext(ctx).
		Order("category asc, created_at asc, uri asc").
		Find(&atoms).Error; err != nil {
		return fmt.Errorf("load atoms: %w", err)
	}

	buckets, skipped := groupAtomsIntoBuckets(atoms)
	if skipped > 0 {
		log.Printf("t3 worker skipped %d slug-less atoms in slug categories", skipped)
	}
	if len(buckets) == 0 {
		// Nothing to distill; still clear pending so they do not loop forever.
		return w.markSessionsIdle(ctx, pendingIDs, w.now().UTC())
	}

	atomURIs := make([]string, 0)
	for _, b := range buckets {
		for _, atom := range b.Atoms {
			atomURIs = append(atomURIs, atom.URI)
		}
	}

	var scenes []model.Scene
	if err := w.db.WithContext(ctx).
		Select("uri", "source_atom_uris").
		Where("source_atom_uris && ?", pgarray.TextArray(atomURIs)).
		Find(&scenes).Error; err != nil {
		return fmt.Errorf("load scenes: %w", err)
	}

	index := buildAtomSceneIndex(scenes)

	// Active human corrections, keyed by the memory URI they target. These are
	// injected into distillation as authoritative input so the regenerated body
	// reflects them (write-time injection). See docs/corrections.md.
	bucketURIs := make([]string, 0, len(buckets))
	for _, b := range buckets {
		bucketURIs = append(bucketURIs, b.URI)
	}
	corrByTarget, err := correction.ForTargets(ctx, w.db, bucketURIs)
	if err != nil {
		return fmt.Errorf("load corrections: %w", err)
	}

	// transientPending: a temporary failure (rate limit/timeout/DB) means work is
	// genuinely unfinished, so leave sessions pending to retry the whole cycle.
	// Permanent per-bucket failures (e.g. un-parseable LLM output) are logged and
	// skipped: they must not block all sessions forever or force a full re-distill
	// every tick. Such a bucket is retried only when its inputs next change.
	transientPending := false

	for _, bucket := range buckets {
		srcScenes := sourceSceneURIsFor(bucket, index)
		corrStatements, corrURIs := splitCorrections(corrByTarget[bucket.URI])

		// Provenance gate: skip the LLM call entirely when this bucket's source
		// scene set AND active correction set are unchanged since its active
		// memory was written (and events, which are immutable once materialized).
		// This is the primary fix for version churn and wasted re-distillation.
		unchanged, err := w.bucketUnchanged(ctx, bucket, srcScenes, corrURIs)
		if err != nil {
			log.Printf("t3 worker provenance check bucket=%s failed: %v", bucket.URI, err)
			transientPending = true
			break
		}
		if unchanged {
			continue
		}

		pm, err := w.distillBucket(ctx, bucket, corrStatements)
		if err != nil {
			if isTransient(err) {
				log.Printf("t3 worker transient error bucket=%s: %v", bucket.URI, err)
				transientPending = true
				break
			}
			log.Printf("t3 worker bucket=%s failed (skipped): %v", bucket.URI, err)
			continue
		}

		if err := w.persistMemory(ctx, bucket, pm, srcScenes, corrURIs); err != nil {
			if isTransient(err) {
				log.Printf("t3 worker persist transient bucket=%s: %v", bucket.URI, err)
				transientPending = true
				break
			}
			log.Printf("t3 worker persist bucket=%s failed (skipped): %v", bucket.URI, err)
			continue
		}
		time.Sleep(distillDelay)
	}

	if transientPending {
		log.Printf("t3 worker rollup incomplete (transient); leaving %d sessions pending for retry", len(pendingIDs))
		return nil
	}
	return w.markSessionsIdle(ctx, pendingIDs, w.now().UTC())
}

// bucketUnchanged reports whether a bucket can skip re-distillation: its active
// memory exists and (for events) is immutable, or its stored source scene set
// equals the current one. Returns false when no active row exists (must distill).
func (w *Worker) bucketUnchanged(ctx context.Context, bucket memoryBucket, srcScenes, corrURIs []string) (bool, error) {
	var active model.Memory
	err := w.db.WithContext(ctx).
		Select("source_scene_uris", "source_correction_uris").
		Where("uri = ? AND superseded_at IS NULL", bucket.URI).
		Take(&active).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("load active memory provenance: %w", err)
	}
	// events are immutable once materialized; never re-distill. Human corrections
	// on events still surface via the read-time overlay, just not in the body.
	if bucket.Category == model.AtomCategoryEvents {
		return true, nil
	}
	return equalStringSets([]string(active.SourceSceneURIs), srcScenes) &&
		equalStringSets([]string(active.SourceCorrectionURIs), corrURIs), nil
}

func (w *Worker) pendingSessionIDs(ctx context.Context) ([]uuid.UUID, error) {
	type row struct {
		SessionID uuid.UUID
	}
	var rows []row
	if err := w.db.WithContext(ctx).Raw(`
		SELECT session_id
		FROM pipeline_state
		WHERE t3_status = 'pending'
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query pending t3 sessions: %w", err)
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.SessionID)
	}
	return out, nil
}

// distillBucket produces one memory from a bucket, chunking and merging when the
// bucket exceeds the per-call atom budget so each LLM response stays complete.
func (w *Worker) distillBucket(ctx context.Context, bucket memoryBucket, corrections []string) (parsedMemory, error) {
	chunks := chunkAtoms(bucket.Atoms, w.config.MaxAtomsPerBatch)

	if len(chunks) == 1 {
		atomsJSON, err := serializeAtomsForLLM(chunks[0])
		if err != nil {
			return parsedMemory{}, err
		}
		raw, err := w.llm.DistillMemory(ctx, bucket.Category, bucket.Slug, atomsJSON, corrections)
		if err != nil {
			return parsedMemory{}, err
		}
		return parseDistillResponse(raw)
	}

	// Per-chunk passes only extract partial bodies, so corrections are withheld
	// here and applied once at the final merge, where the whole picture is in view.
	partials := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		atomsJSON, err := serializeAtomsForLLM(chunk)
		if err != nil {
			return parsedMemory{}, err
		}
		raw, err := w.llm.DistillMemory(ctx, bucket.Category, bucket.Slug, atomsJSON, nil)
		if err != nil {
			return parsedMemory{}, err
		}
		pm, err := parseDistillResponse(raw)
		if err != nil {
			return parsedMemory{}, err
		}
		partials = append(partials, pm.Body)
		time.Sleep(distillDelay)
	}

	mergedJSON, err := serializePartialsForLLM(partials)
	if err != nil {
		return parsedMemory{}, err
	}
	raw, err := w.llm.DistillMemory(ctx, bucket.Category, bucket.Slug, mergedJSON, corrections)
	if err != nil {
		return parsedMemory{}, err
	}
	return parseDistillResponse(raw)
}

func (w *Worker) persistMemory(
	ctx context.Context,
	bucket memoryBucket,
	pm parsedMemory,
	sourceSceneURIs []string,
	sourceCorrectionURIs []string,
) error {
	return w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := w.now().UTC()
		var slugPtr *string
		if bucket.Slug != "" {
			slug := bucket.Slug
			slugPtr = &slug
		}

		// events are immutable: insert once per slug, never supersede.
		if bucket.Category == model.AtomCategoryEvents {
			var existing int64
			if err := tx.Model(&model.Memory{}).
				Where("uri = ? AND superseded_at IS NULL", bucket.URI).
				Count(&existing).Error; err != nil {
				return fmt.Errorf("count event memory: %w", err)
			}
			if existing > 0 {
				return nil
			}
			return insertMemory(tx, bucket, slugPtr, pm, sourceSceneURIs, sourceCorrectionURIs, 1, now)
		}

		var active model.Memory
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uri = ? AND superseded_at IS NULL", bucket.URI).
			Take(&active).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			return insertMemory(tx, bucket, slugPtr, pm, sourceSceneURIs, sourceCorrectionURIs, 1, now)
		case err != nil:
			return fmt.Errorf("load active memory: %w", err)
		}

		// Idempotent: when the body is unchanged, avoid a new version. But if the
		// active correction set changed (e.g. a correction was retracted but the
		// LLM produced the same text), refresh the provenance in place so the gate
		// stops re-firing every cycle.
		if active.Body != nil && *active.Body == pm.Body {
			if equalStringSets([]string(active.SourceCorrectionURIs), sourceCorrectionURIs) {
				return nil
			}
			return tx.Model(&model.Memory{}).
				Where("id = ?", active.ID).
				Updates(map[string]any{
					"source_correction_uris": pgarray.TextArray(append([]string(nil), sourceCorrectionURIs...)),
					"updated_at":             now,
				}).Error
		}

		if err := tx.Model(&model.Memory{}).
			Where("id = ?", active.ID).
			Update("superseded_at", now).Error; err != nil {
			return fmt.Errorf("supersede memory: %w", err)
		}
		return insertMemory(tx, bucket, slugPtr, pm, sourceSceneURIs, sourceCorrectionURIs, active.Version+1, now)
	})
}

func insertMemory(
	tx *gorm.DB,
	bucket memoryBucket,
	slugPtr *string,
	pm parsedMemory,
	sourceSceneURIs []string,
	sourceCorrectionURIs []string,
	version int,
	now time.Time,
) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate memory id: %w", err)
	}
	abstract := pm.Abstract
	body := pm.Body
	row := model.Memory{
		ID:                   id,
		URI:                  bucket.URI,
		Category:             bucket.Category,
		Slug:                 slugPtr,
		Version:              version,
		Abstract:             &abstract,
		Body:                 &body,
		SourceSceneURIs:      pgarray.TextArray(append([]string(nil), sourceSceneURIs...)),
		SourceCorrectionURIs: pgarray.TextArray(append([]string(nil), sourceCorrectionURIs...)),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := tx.Create(&row).Error; err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}
	log.Printf("t3 worker wrote memory uri=%s version=%d", bucket.URI, version)
	return nil
}
