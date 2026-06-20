// Package alias implements the entity-resolution layer: append-only,
// human-authored aliases declaring that one memory URI is the same entity as
// another. Aliases live outside the memories tier (survive re-distillation) and
// form a flat star (depth 1): an alias points directly to a canonical, and a
// canonical may not itself be an active alias. See docs/aliases.md.
package alias

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/mem9/internal/model"
	"github.com/colinleefish/mem9/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInvalidInput marks a bad alias request (caller error → HTTP 400).
var ErrInvalidInput = errors.New("invalid alias input")

// ErrNotFound marks a retract of an alias that is missing or already retired.
var ErrNotFound = errors.New("alias not found")

// ErrConflict marks a request that violates an alias invariant (e.g. the alias
// URI is already aliased, or the canonical is itself an alias).
var ErrConflict = errors.New("alias conflict")

// Summary is the inspect overlay view of an active alias.
type Summary struct {
	URI          string    `json:"uri"`
	AliasURI     string    `json:"alias_uri"`
	CanonicalURI string    `json:"canonical_uri"`
	Note         string    `json:"note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateInput declares aliasURI to be the same entity as canonicalURI.
type CreateInput struct {
	AliasURI     string
	CanonicalURI string
	Note         string
}

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: time.Now}
}

// Create validates and inserts an append-only alias, returning the new row. It
// enforces the flat-star invariants: alias != canonical, both are memory-tier
// URIs, the alias is not already aliased, and the canonical is not itself an
// active alias.
func (s *Service) Create(ctx context.Context, in CreateInput) (model.Alias, error) {
	aliasURI, aliasCat, err := normalizeMemoryURI(in.AliasURI)
	if err != nil {
		return model.Alias{}, fmt.Errorf("%w: alias_uri: %v", ErrInvalidInput, err)
	}
	canonicalURI, canonicalCat, err := normalizeMemoryURI(in.CanonicalURI)
	if err != nil {
		return model.Alias{}, fmt.Errorf("%w: canonical_uri: %v", ErrInvalidInput, err)
	}
	if aliasURI == canonicalURI {
		return model.Alias{}, fmt.Errorf("%w: alias_uri and canonical_uri must differ", ErrInvalidInput)
	}
	// Same-category invariant: T3 routing merges by (category, slug), so an alias
	// only makes sense within one category — you cannot fold an entities atom into
	// a preferences bucket.
	if aliasCat != canonicalCat {
		return model.Alias{}, fmt.Errorf("%w: alias_uri and canonical_uri must be the same category (%s vs %s)", ErrInvalidInput, aliasCat, canonicalCat)
	}

	// Invariant: the alias URI must not already be an active alias of something.
	var existing int64
	if err := s.db.WithContext(ctx).Model(&model.Alias{}).
		Where("alias_uri = ? AND superseded_at IS NULL", aliasURI).
		Count(&existing).Error; err != nil {
		return model.Alias{}, fmt.Errorf("check existing alias: %w", err)
	}
	if existing > 0 {
		return model.Alias{}, fmt.Errorf("%w: %s is already an active alias; retract it first", ErrConflict, aliasURI)
	}

	// Invariant (flat star): the canonical must not itself be an active alias.
	var canonicalIsAlias int64
	if err := s.db.WithContext(ctx).Model(&model.Alias{}).
		Where("alias_uri = ? AND superseded_at IS NULL", canonicalURI).
		Count(&canonicalIsAlias).Error; err != nil {
		return model.Alias{}, fmt.Errorf("check canonical chain: %w", err)
	}
	if canonicalIsAlias > 0 {
		return model.Alias{}, fmt.Errorf("%w: canonical %s is itself an alias; aliases cannot chain", ErrConflict, canonicalURI)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return model.Alias{}, fmt.Errorf("generate alias id: %w", err)
	}
	now := s.now().UTC()
	row := model.Alias{
		ID:           id,
		URI:          uri.BuildAlias(id.String()),
		AliasURI:     aliasURI,
		CanonicalURI: canonicalURI,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if note := strings.TrimSpace(in.Note); note != "" {
		row.Note = &note
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return model.Alias{}, fmt.Errorf("insert alias: %w", err)
	}
	return row, nil
}

// Retract retires a specific alias by its record URI (sets superseded_at).
// Returns the retracted alias's canonical URI (so callers can invalidate caches)
// or ErrNotFound if none matches.
func (s *Service) Retract(ctx context.Context, aliasRecordURI string) (string, error) {
	u, err := uri.Parse(strings.TrimSpace(aliasRecordURI))
	if err != nil || u.Scope != uri.ScopeAliases {
		return "", fmt.Errorf("%w: not an alias URI: %q", ErrInvalidInput, aliasRecordURI)
	}
	var row model.Alias
	err = s.db.WithContext(ctx).
		Where("uri = ? AND superseded_at IS NULL", u.String()).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return "", fmt.Errorf("%w: %s", ErrNotFound, u.String())
	}
	if err != nil {
		return "", fmt.Errorf("load alias: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Model(&model.Alias{}).
		Where("id = ?", row.ID).
		Update("superseded_at", s.now().UTC()).Error; err != nil {
		return "", fmt.Errorf("retract alias: %w", err)
	}
	return row.CanonicalURI, nil
}

// List returns active aliases, newest-first. When uriFilter is non-empty, only
// aliases where it appears on either side (alias or canonical) are returned.
func (s *Service) List(ctx context.Context, uriFilter string) ([]model.Alias, error) {
	q := s.db.WithContext(ctx).Where("superseded_at IS NULL")
	if f := strings.TrimSpace(uriFilter); f != "" {
		u, err := uri.Parse(f)
		if err != nil {
			return nil, fmt.Errorf("%w: filter %q: %v", ErrInvalidInput, uriFilter, err)
		}
		q = q.Where("alias_uri = ? OR canonical_uri = ?", u.String(), u.String())
	}
	var rows []model.Alias
	if err := q.Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}
	return rows, nil
}

// Resolve maps a set of URIs to their active canonical URI (single hop). A URI
// not present as an active alias maps to itself. Used by recall to fold alias
// hits into their canonical before dedup.
func Resolve(ctx context.Context, db *gorm.DB, uris []string) (map[string]string, error) {
	out := make(map[string]string, len(uris))
	wanted := make([]string, 0, len(uris))
	for _, u := range uris {
		if u == "" {
			continue
		}
		if _, seen := out[u]; seen {
			continue
		}
		out[u] = u // default: maps to itself
		wanted = append(wanted, u)
	}
	if len(wanted) == 0 {
		return out, nil
	}
	var rows []model.Alias
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL AND alias_uri IN ?", wanted).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("resolve aliases: %w", err)
	}
	for _, r := range rows {
		out[r.AliasURI] = r.CanonicalURI
	}
	return out, nil
}

// ForTargets returns active aliases relevant to the given memory URIs, split
// into two maps: aliasOf[u] = the canonical u points to (u is an alias), and
// aliasesOf[u] = the list of aliases pointing at u (u is a canonical). Used by
// inspect to annotate cat/meta on either side.
func ForTargets(ctx context.Context, db *gorm.DB, uris []string) (aliasOf map[string]Summary, aliasesOf map[string][]Summary, err error) {
	aliasOf = make(map[string]Summary)
	aliasesOf = make(map[string][]Summary)
	wanted := make([]string, 0, len(uris))
	for _, u := range uris {
		if u != "" {
			wanted = append(wanted, u)
		}
	}
	if len(wanted) == 0 {
		return aliasOf, aliasesOf, nil
	}
	var rows []model.Alias
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL AND (alias_uri IN ? OR canonical_uri IN ?)", wanted, wanted).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("load aliases: %w", err)
	}
	want := make(map[string]struct{}, len(wanted))
	for _, u := range wanted {
		want[u] = struct{}{}
	}
	for _, r := range rows {
		sum := toSummary(r)
		if _, ok := want[r.AliasURI]; ok {
			aliasOf[r.AliasURI] = sum
		}
		if _, ok := want[r.CanonicalURI]; ok {
			aliasesOf[r.CanonicalURI] = append(aliasesOf[r.CanonicalURI], sum)
		}
	}
	return aliasOf, aliasesOf, nil
}

func toSummary(r model.Alias) Summary {
	sum := Summary{
		URI:          r.URI,
		AliasURI:     r.AliasURI,
		CanonicalURI: r.CanonicalURI,
		CreatedAt:    r.CreatedAt,
	}
	if r.Note != nil {
		sum.Note = *r.Note
	}
	return sum
}

// normalizeMemoryURI parses and canonicalizes a URI, requiring it to be a
// slug-routed, mergeable memory URI (preferences or entities) and returning its
// category. profile (singleton, no slug) and events (immutable, never
// re-distilled) cannot be merged, so they cannot participate in an alias; nor
// can sessions, scenes, corrections, or aliases.
func normalizeMemoryURI(raw string) (string, string, error) {
	u, err := uri.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", err
	}
	switch u.Scope {
	case uri.ScopePrefs, uri.ScopeEntities:
		return u.String(), u.Scope, nil
	default:
		return "", "", fmt.Errorf("must be a preferences or entities URI, got %q", u.Scope)
	}
}

// ActiveMap returns every active alias as alias_uri → canonical_uri. T3 uses it
// to fold aliased slugs into their canonical bucket before distillation.
func ActiveMap(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	var rows []model.Alias
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load active aliases: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.AliasURI] = r.CanonicalURI
	}
	return out, nil
}

// CanonicalFor reports the canonical URI for a memory URI when it is an active
// alias (single hop). isAlias is false when the URI is not aliased.
func CanonicalFor(ctx context.Context, db *gorm.DB, memoryURI string) (canonical string, isAlias bool, err error) {
	var row model.Alias
	err = db.WithContext(ctx).
		Where("alias_uri = ? AND superseded_at IS NULL", memoryURI).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("resolve canonical: %w", err)
	}
	return row.CanonicalURI, true, nil
}
