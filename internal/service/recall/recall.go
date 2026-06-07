// Package recall provides lexical (full-text) retrieval over the memory pyramid.
// It is the shared foundation for `mypast eval` today and `find`/`search` later;
// vector recall will be layered on once the embed worker populates embeddings.
package recall

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Match is a single retrieval hit.
type Match struct {
	URI     string  `gorm:"column:uri"`
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
