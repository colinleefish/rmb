// Package recall provides retrieval over the memory pyramid: lexical (full-text)
// and vector (cosine), plus rank fusion. It backs `mypast eval`, `find`, and
// `search`.
package recall

import (
	"context"
	"fmt"
	"sort"

	"github.com/colinleefish/mypast/internal/db/pgarray"
	"gorm.io/gorm"
)

// Match is a single retrieval hit. Tier is the pyramid layer ("memories",
// "scenes", "turns") and Rank is method-specific (ts_rank or cosine similarity).
type Match struct {
	URI     string  `gorm:"column:uri"`
	Tier    string  `gorm:"column:tier"`
	Rank    float64 `gorm:"column:rank"`
	Snippet string  `gorm:"column:snippet"`
}

// FTSMemories runs full-text search over active memories (body + abstract),
// returning the top-k by ts_rank.
func FTSMemories(ctx context.Context, db *gorm.DB, query string, k int) ([]Match, error) {
	if k <= 0 {
		k = 5
	}
	var out []Match
	if err := db.WithContext(ctx).Raw(`
		SELECT uri,
		       'memories' AS tier,
		       ts_rank(
		         to_tsvector('english', coalesce(body, '') || ' ' || coalesce(abstract, '')),
		         websearch_to_tsquery('english', ?)
		       ) AS rank,
		       left(coalesce(abstract, body, ''), 160) AS snippet
		FROM memories
		WHERE superseded_at IS NULL
		  AND to_tsvector('english', coalesce(body, '') || ' ' || coalesce(abstract, ''))
		      @@ websearch_to_tsquery('english', ?)
		ORDER BY rank DESC
		LIMIT ?
	`, query, query, k).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("fts memories: %w", err)
	}
	return out, nil
}

// FTSScenes runs full-text search over scenes (body + abstract).
func FTSScenes(ctx context.Context, db *gorm.DB, query string, k int) ([]Match, error) {
	if k <= 0 {
		k = 5
	}
	var out []Match
	if err := db.WithContext(ctx).Raw(`
		SELECT uri,
		       'scenes' AS tier,
		       ts_rank(
		         to_tsvector('english', coalesce(body, '') || ' ' || coalesce(abstract, '')),
		         websearch_to_tsquery('english', ?)
		       ) AS rank,
		       left(coalesce(abstract, body, ''), 160) AS snippet
		FROM scenes
		WHERE to_tsvector('english', coalesce(body, '') || ' ' || coalesce(abstract, ''))
		      @@ websearch_to_tsquery('english', ?)
		ORDER BY rank DESC
		LIMIT ?
	`, query, query, k).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("fts scenes: %w", err)
	}
	return out, nil
}

// VectorMemories returns active memories nearest to queryVec by cosine distance.
// Rank is cosine similarity (1 - distance), higher is closer.
func VectorMemories(ctx context.Context, db *gorm.DB, queryVec pgarray.Vector, k int) ([]Match, error) {
	if k <= 0 {
		k = 5
	}
	lit, err := queryVec.Value()
	if err != nil {
		return nil, fmt.Errorf("encode query vector: %w", err)
	}
	var out []Match
	if err := db.WithContext(ctx).Raw(`
		SELECT uri,
		       'memories' AS tier,
		       1 - (embedding <=> ?::vector) AS rank,
		       left(coalesce(abstract, body, ''), 160) AS snippet
		FROM memories
		WHERE superseded_at IS NULL AND embedding IS NOT NULL
		ORDER BY embedding <=> ?::vector
		LIMIT ?
	`, lit, lit, k).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("vector memories: %w", err)
	}
	return out, nil
}

// VectorScenes returns scenes nearest to queryVec by cosine distance.
func VectorScenes(ctx context.Context, db *gorm.DB, queryVec pgarray.Vector, k int) ([]Match, error) {
	if k <= 0 {
		k = 5
	}
	lit, err := queryVec.Value()
	if err != nil {
		return nil, fmt.Errorf("encode query vector: %w", err)
	}
	var out []Match
	if err := db.WithContext(ctx).Raw(`
		SELECT uri,
		       'scenes' AS tier,
		       1 - (embedding <=> ?::vector) AS rank,
		       left(coalesce(abstract, body, ''), 160) AS snippet
		FROM scenes
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> ?::vector
		LIMIT ?
	`, lit, lit, k).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("vector scenes: %w", err)
	}
	return out, nil
}

// FuseRRF combines ranked result lists using Reciprocal Rank Fusion. The fused
// score for a URI is sum over lists of 1/(kRRF + position). kRRF dampens the
// weight of low-ranked items; 60 is the common default. Snippet/Tier come from
// the first list that contains each URI. Returns the top-k fused matches.
func FuseRRF(lists [][]Match, kRRF, topK int) []Match {
	if kRRF <= 0 {
		kRRF = 60
	}
	type agg struct {
		match Match
		score float64
	}
	byURI := make(map[string]*agg)
	for _, list := range lists {
		for pos, m := range list {
			a, ok := byURI[m.URI]
			if !ok {
				a = &agg{match: m}
				byURI[m.URI] = a
			}
			a.score += 1.0 / float64(kRRF+pos+1)
		}
	}

	fused := make([]Match, 0, len(byURI))
	for _, a := range byURI {
		m := a.match
		m.Rank = a.score
		fused = append(fused, m)
	}
	sort.SliceStable(fused, func(i, j int) bool {
		return fused[i].Rank > fused[j].Rank
	})
	if topK > 0 && len(fused) > topK {
		fused = fused[:topK]
	}
	return fused
}

// FTSTurns runs full-text search over raw T0 turns. It is the baseline ceiling
// for eval: "is the information even present in the conversation evidence?".
// No FTS index exists on session_turns; eval volume is small so a scan is fine.
func FTSTurns(ctx context.Context, db *gorm.DB, query string, k int) ([]Match, error) {
	if k <= 0 {
		k = 5
	}
	var out []Match
	if err := db.WithContext(ctx).Raw(`
		SELECT 'mypast://sessions/' || session_id::text AS uri,
		       'turns' AS tier,
		       ts_rank(
		         to_tsvector('english', messages_jsonl),
		         websearch_to_tsquery('english', ?)
		       ) AS rank,
		       left(messages_jsonl, 160) AS snippet
		FROM session_turns
		WHERE to_tsvector('english', messages_jsonl) @@ websearch_to_tsquery('english', ?)
		ORDER BY rank DESC
		LIMIT ?
	`, query, query, k).Scan(&out).Error; err != nil {
		return nil, fmt.Errorf("fts turns: %w", err)
	}
	return out, nil
}
