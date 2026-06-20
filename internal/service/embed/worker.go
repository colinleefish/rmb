package embed

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/colinleefish/rmb/internal/config"
	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/llm"
	"gorm.io/gorm"
)

type Embedder interface {
	Embed(ctx context.Context, inputs []string) ([][]float32, error)
}

type Worker struct {
	db     *gorm.DB
	llm    Embedder
	config config.EmbedConfig
}

func NewWorker(database *gorm.DB, embedder Embedder, cfg config.EmbedConfig) *Worker {
	return &Worker{db: database, llm: embedder, config: cfg}
}

func (w *Worker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("embed worker disabled")
		return nil
	}
	if w.llm == nil {
		return fmt.Errorf("embed worker requires embedding client")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid embed poll interval: %s", w.config.PollInterval)
	}

	log.Printf("embed worker started poll_interval=%s batch=%d", w.config.PollInterval, w.batchSize())

	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("embed worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

func (w *Worker) batchSize() int {
	if w.config.BatchSize > 0 {
		return w.config.BatchSize
	}
	return 32
}

// runOneCycle embeds one batch per tier. Transient LLM errors leave rows NULL
// for retry on the next tick; non-transient errors are logged and skipped.
func (w *Worker) runOneCycle(ctx context.Context) {
	for _, tier := range []struct {
		name string
		fn   func(context.Context) (int, error)
	}{
		{"atoms", w.embedAtoms},
		{"scenes", w.embedScenes},
		{"memories", w.embedMemories},
	} {
		n, err := tier.fn(ctx)
		if err != nil {
			if llm.IsTransientError(err) {
				log.Printf("embed worker %s transient error: %v", tier.name, err)
			} else {
				log.Printf("embed worker %s failed: %v", tier.name, err)
			}
			continue
		}
		if n > 0 {
			log.Printf("embed worker %s embedded=%d", tier.name, n)
		}
	}
}

type embedRow struct {
	Key  string
	Text string
}

func (w *Worker) embedAtoms(ctx context.Context) (int, error) {
	var rows []embedRow
	if err := w.db.WithContext(ctx).Raw(`
		SELECT uri AS key, content AS text
		FROM atoms
		WHERE embedding IS NULL AND content <> ''
		ORDER BY created_at
		LIMIT ?
	`, w.batchSize()).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("select atoms: %w", err)
	}
	return w.embedAndStore(ctx, "atoms", "uri", rows)
}

func (w *Worker) embedScenes(ctx context.Context) (int, error) {
	var rows []embedRow
	if err := w.db.WithContext(ctx).Raw(`
		SELECT uri AS key, COALESCE(NULLIF(TRIM(abstract), ''), COALESCE(body, '')) AS text
		FROM scenes
		WHERE embedding IS NULL
		ORDER BY created_at
		LIMIT ?
	`, w.batchSize()).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("select scenes: %w", err)
	}
	return w.embedAndStore(ctx, "scenes", "uri", rows)
}

func (w *Worker) embedMemories(ctx context.Context) (int, error) {
	// Key by id: uri is not unique across versions, so updating by uri could
	// touch superseded rows.
	var rows []embedRow
	if err := w.db.WithContext(ctx).Raw(`
		SELECT id::text AS key, COALESCE(NULLIF(TRIM(abstract), ''), COALESCE(body, '')) AS text
		FROM memories
		WHERE embedding IS NULL AND superseded_at IS NULL
		ORDER BY created_at
		LIMIT ?
	`, w.batchSize()).Scan(&rows).Error; err != nil {
		return 0, fmt.Errorf("select memories: %w", err)
	}
	return w.embedAndStore(ctx, "memories", "id", rows)
}

func (w *Worker) embedAndStore(ctx context.Context, table, keyCol string, rows []embedRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	inputs := make([]string, len(rows))
	for i, r := range rows {
		text := r.Text
		if text == "" {
			text = "(empty)"
		}
		inputs[i] = text
	}

	vectors, err := w.llm.Embed(ctx, inputs)
	if err != nil {
		return 0, err
	}
	if len(vectors) != len(rows) {
		return 0, fmt.Errorf("embed returned %d vectors for %d rows", len(vectors), len(rows))
	}

	written := 0
	for i, r := range rows {
		vec := pgarray.Vector(vectors[i])
		if err := w.db.WithContext(ctx).
			Table(table).
			Where(keyCol+" = ?", r.Key).
			Update("embedding", vec).Error; err != nil {
			return written, fmt.Errorf("update %s embedding: %w", table, err)
		}
		written++
	}
	return written, nil
}
