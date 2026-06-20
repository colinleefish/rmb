package alias

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/config"
	"github.com/colinleefish/mypast/internal/db"
	"github.com/colinleefish/mypast/internal/llm"
	"github.com/colinleefish/mypast/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const suggestLockKey = "mypast-alias-suggest"

// AliasJudge decides whether two memory entries are the same entity. The chat
// LLM client satisfies it; tests supply a mock so candidate generation can be
// exercised without network access.
type AliasJudge interface {
	JudgeAlias(ctx context.Context, aURI, aBody, bURI, bBody string) (llm.AliasVerdict, error)
}

// SuggestWorker proposes alias candidates: it finds near-duplicate entity and
// preference memories by embedding similarity, asks an LLM judge whether each
// near pair is the same entity, and records the verdict in alias_candidates. It
// NEVER writes a live alias — confirmation is a separate, human-gated step. See
// docs/aliases.md.
type SuggestWorker struct {
	db     *gorm.DB
	judge  AliasJudge
	config config.AliasSuggestConfig
}

func NewSuggestWorker(database *gorm.DB, judge AliasJudge, cfg config.AliasSuggestConfig) *SuggestWorker {
	return &SuggestWorker{db: database, judge: judge, config: cfg}
}

func (w *SuggestWorker) Run(ctx context.Context) error {
	if !w.config.Enabled {
		log.Printf("alias-suggest worker disabled")
		return nil
	}
	if w.judge == nil {
		return fmt.Errorf("alias-suggest worker requires an llm judge")
	}
	if w.config.PollInterval <= 0 {
		return fmt.Errorf("invalid alias-suggest poll interval: %s", w.config.PollInterval)
	}

	log.Printf("alias-suggest worker started poll_interval=%s min_similarity=%.2f",
		w.config.PollInterval, w.minSimilarity())

	w.runOneCycle(ctx)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("alias-suggest worker stopped")
			return nil
		case <-ticker.C:
			w.runOneCycle(ctx)
		}
	}
}

func (w *SuggestWorker) batchMemories() int {
	if w.config.BatchMemories > 0 {
		return w.config.BatchMemories
	}
	return 50
}

func (w *SuggestWorker) neighbors() int {
	if w.config.Neighbors > 0 {
		return w.config.Neighbors
	}
	return 5
}

func (w *SuggestWorker) minSimilarity() float64 {
	if w.config.MinSimilarity > 0 {
		return w.config.MinSimilarity
	}
	return 0.82
}

// runOneCycle holds the same single global advisory lock pattern as the T3
// worker so only one suggest pass runs at a time without keeping a transaction
// open across LLM calls.
func (w *SuggestWorker) runOneCycle(ctx context.Context) {
	err := w.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		locked, err := db.TryGlobalAdvisoryLock(conn, suggestLockKey)
		if err != nil {
			return err
		}
		if !locked {
			return nil
		}
		defer func() { _ = db.GlobalAdvisoryUnlock(conn, suggestLockKey) }()
		return w.generate(ctx)
	})
	if err != nil {
		log.Printf("alias-suggest worker cycle failed: %v", err)
	}
}

// memoryRow is a seed/neighbor memory used for pairing. Sim is the cosine
// similarity to the seed, set only on neighbor rows.
type memoryRow struct {
	URI      string  `gorm:"column:uri"`
	Category string  `gorm:"column:category"`
	Body     string  `gorm:"column:body"`
	Sim      float64 `gorm:"column:sim"`
}

// pair is an unordered candidate pair carrying both bodies for the judge and the
// cosine similarity that surfaced it.
type pair struct {
	AURI, ABody string
	BURI, BBody string
	Sim         float64
}

func (w *SuggestWorker) generate(ctx context.Context) error {
	seeds, err := w.loadSeeds(ctx)
	if err != nil {
		return err
	}
	if len(seeds) == 0 {
		return nil
	}

	var raw []pair
	for _, seed := range seeds {
		neighbors, err := w.loadNeighbors(ctx, seed)
		if err != nil {
			return err
		}
		for _, n := range neighbors {
			raw = append(raw, pair{AURI: seed.URI, ABody: seed.Body, BURI: n.URI, BBody: n.Body, Sim: n.Sim})
		}
	}
	if len(raw) == 0 {
		return nil
	}

	judged, err := w.judgedPairKeys(ctx)
	if err != nil {
		return err
	}
	aliased, err := w.aliasedURIs(ctx)
	if err != nil {
		return err
	}

	todo := selectPairsToJudge(raw, judged, aliased)
	if len(todo) == 0 {
		return nil
	}

	inserted := 0
	for _, p := range todo {
		verdict, err := w.judge.JudgeAlias(ctx, p.AURI, p.ABody, p.BURI, p.BBody)
		if err != nil {
			if llm.IsTransientError(err) {
				log.Printf("alias-suggest transient judge error (leaving for retry): %v", err)
				return nil
			}
			log.Printf("alias-suggest judge %s ~ %s failed (skipped): %v", p.AURI, p.BURI, err)
			continue
		}
		row := candidateFromVerdict(p, verdict, p.Sim)
		ok, err := w.insertCandidate(ctx, row)
		if err != nil {
			return fmt.Errorf("insert candidate: %w", err)
		}
		if ok {
			inserted++
		}
	}
	if inserted > 0 {
		log.Printf("alias-suggest worker recorded %d candidate(s)", inserted)
	}
	return nil
}

// loadSeeds returns a bounded batch of active, embedded entity/preference
// memories, newest-first so freshly distilled memories are judged promptly.
func (w *SuggestWorker) loadSeeds(ctx context.Context) ([]memoryRow, error) {
	var rows []memoryRow
	if err := w.db.WithContext(ctx).Raw(`
		SELECT uri, category, COALESCE(body, abstract, '') AS body
		FROM memories
		WHERE superseded_at IS NULL
		  AND embedding IS NOT NULL
		  AND category IN ('entities', 'preferences')
		ORDER BY created_at DESC
		LIMIT ?
	`, w.batchMemories()).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load seed memories: %w", err)
	}
	return rows, nil
}

// loadNeighbors returns the nearest same-category memories to the seed above the
// similarity threshold (cosine, excluding self).
func (w *SuggestWorker) loadNeighbors(ctx context.Context, seed memoryRow) ([]memoryRow, error) {
	var rows []memoryRow
	if err := w.db.WithContext(ctx).Raw(`
		SELECT m2.uri AS uri,
		       m2.category AS category,
		       COALESCE(m2.body, m2.abstract, '') AS body,
		       1 - (m1.embedding <=> m2.embedding) AS sim
		FROM memories m1
		JOIN memories m2
		  ON m2.category = m1.category
		 AND m2.uri <> m1.uri
		 AND m2.superseded_at IS NULL
		 AND m2.embedding IS NOT NULL
		WHERE m1.uri = ?
		  AND m1.superseded_at IS NULL
		  AND m1.embedding IS NOT NULL
		  AND 1 - (m1.embedding <=> m2.embedding) >= ?
		ORDER BY m1.embedding <=> m2.embedding
		LIMIT ?
	`, seed.URI, w.minSimilarity(), w.neighbors()).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load neighbors for %s: %w", seed.URI, err)
	}
	return rows, nil
}

// judgedPairKeys returns the set of unordered pair keys already present in
// alias_candidates (any status), so a pair is judged at most once ever.
func (w *SuggestWorker) judgedPairKeys(ctx context.Context) (map[string]struct{}, error) {
	type row struct {
		AliasURI     string `gorm:"column:alias_uri"`
		CanonicalURI string `gorm:"column:canonical_uri"`
	}
	var rows []row
	if err := w.db.WithContext(ctx).
		Model(&model.AliasCandidate{}).
		Select("alias_uri", "canonical_uri").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load judged candidates: %w", err)
	}
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[pairKey(r.AliasURI, r.CanonicalURI)] = struct{}{}
	}
	return out, nil
}

// aliasedURIs returns every URI currently involved in an active alias (either
// side). A URI that is already an alias or a canonical must not be re-proposed.
func (w *SuggestWorker) aliasedURIs(ctx context.Context) (map[string]struct{}, error) {
	m, err := ActiveMap(ctx, w.db)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(m)*2)
	for a, c := range m {
		out[a] = struct{}{}
		out[c] = struct{}{}
	}
	return out, nil
}

// insertCandidate writes one candidate, relying on the unique (alias_uri,
// canonical_uri) index for idempotency. Returns false when the row already
// existed (conflict ignored).
func (w *SuggestWorker) insertCandidate(ctx context.Context, row model.AliasCandidate) (bool, error) {
	res := w.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&row)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// selectPairsToJudge dedups raw pairs to one entry per unordered URI pair and
// drops any pair that was already judged or whose either side is already in an
// active alias. Pure for testability.
func selectPairsToJudge(raw []pair, judged, aliased map[string]struct{}) []pair {
	seen := make(map[string]struct{}, len(raw))
	out := make([]pair, 0, len(raw))
	for _, p := range raw {
		if p.AURI == "" || p.BURI == "" || p.AURI == p.BURI {
			continue
		}
		key := pairKey(p.AURI, p.BURI)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if _, done := judged[key]; done {
			continue
		}
		if _, a := aliased[p.AURI]; a {
			continue
		}
		if _, b := aliased[p.BURI]; b {
			continue
		}
		out = append(out, p)
	}
	return out
}

// candidateFromVerdict builds the row to persist. When the judge says "same" and
// names a valid canonical, alias_uri is the other side; otherwise the pair is
// recorded as rejected in a deterministic (sorted) direction so it is never
// re-proposed.
func candidateFromVerdict(p pair, v llm.AliasVerdict, sim float64) model.AliasCandidate {
	simCopy := sim
	row := model.AliasCandidate{Similarity: &simCopy}
	if rationale := strings.TrimSpace(v.Rationale); rationale != "" {
		row.Rationale = &rationale
	}

	canonical := v.CanonicalURI
	validCanonical := canonical == p.AURI || canonical == p.BURI
	if v.Same && validCanonical {
		row.Status = model.AliasCandidateStatusPending
		verdict := "same"
		row.Verdict = &verdict
		row.CanonicalURI = canonical
		if canonical == p.AURI {
			row.AliasURI = p.BURI
		} else {
			row.AliasURI = p.AURI
		}
		return row
	}

	row.Status = model.AliasCandidateStatusRejected
	verdict := "different"
	row.Verdict = &verdict
	a, b := sortedPair(p.AURI, p.BURI)
	row.AliasURI = a
	row.CanonicalURI = b
	return row
}

func pairKey(a, b string) string {
	x, y := sortedPair(a, b)
	return x + "\x00" + y
}

func sortedPair(a, b string) (string, string) {
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0], pair[1]
}
