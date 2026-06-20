// Package correction implements the human authority layer: append-only
// corrections that overlay distilled memory. See docs/corrections.md.
package correction

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInvalidInput marks a bad correction request (caller error → HTTP 400).
var ErrInvalidInput = errors.New("invalid correction input")

// ErrNotFound marks a retract of a correction that is missing or already retired.
var ErrNotFound = errors.New("correction not found")

// Summary is the recall/inspect overlay view of an active correction.
type Summary struct {
	URI       string    `json:"uri"`
	Statement string    `json:"statement"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateInput describes a new human correction: a content overlay (positive or
// negative) on one or more existing memories.
type CreateInput struct {
	TargetURIs []string
	Statement  string
}

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: time.Now}
}

// Create validates and inserts an append-only correction, returning the new row.
func (s *Service) Create(ctx context.Context, in CreateInput) (model.Correction, error) {
	targets, err := normalizeTargets(in.TargetURIs)
	if err != nil {
		return model.Correction{}, err
	}
	if len(targets) == 0 {
		return model.Correction{}, fmt.Errorf("%w: a correction requires at least one target URI", ErrInvalidInput)
	}

	statement := strings.TrimSpace(in.Statement)
	if statement == "" {
		return model.Correction{}, fmt.Errorf("%w: a correction requires a statement", ErrInvalidInput)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return model.Correction{}, fmt.Errorf("generate correction id: %w", err)
	}
	now := s.now().UTC()
	row := model.Correction{
		ID:         id,
		URI:        uri.BuildCorrection(id.String()),
		Author:     "human",
		TargetURIs: pgarray.TextArray(targets),
		Statement:  &statement,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return model.Correction{}, fmt.Errorf("insert correction: %w", err)
	}
	return row, nil
}

// Retract retires a specific correction by its URI (sets superseded_at). This is
// per-correction identity supersession — it does not touch other corrections that
// happen to share a target. Returns the retracted correction's target URIs (so
// callers can re-distill those memories) or ErrNotFound if none matches.
func (s *Service) Retract(ctx context.Context, correctionURI string) ([]string, error) {
	u, err := uri.Parse(strings.TrimSpace(correctionURI))
	if err != nil || u.Scope != uri.ScopeCorrections {
		return nil, fmt.Errorf("%w: not a correction URI: %q", ErrInvalidInput, correctionURI)
	}
	var row model.Correction
	err = s.db.WithContext(ctx).
		Where("uri = ? AND superseded_at IS NULL", u.String()).
		Take(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, u.String())
	}
	if err != nil {
		return nil, fmt.Errorf("load correction: %w", err)
	}
	if err := s.db.WithContext(ctx).
		Model(&model.Correction{}).
		Where("id = ?", row.ID).
		Update("superseded_at", s.now().UTC()).Error; err != nil {
		return nil, fmt.Errorf("retract correction: %w", err)
	}
	return []string(row.TargetURIs), nil
}

// List returns active corrections, newest-first. When target is non-empty, only
// corrections whose target_uris include that URI are returned.
func (s *Service) List(ctx context.Context, target string) ([]model.Correction, error) {
	q := s.db.WithContext(ctx).Where("superseded_at IS NULL")
	if t := strings.TrimSpace(target); t != "" {
		u, err := uri.Parse(t)
		if err != nil {
			return nil, fmt.Errorf("%w: target %q: %v", ErrInvalidInput, target, err)
		}
		q = q.Where("target_uris && ?", pgarray.TextArray([]string{u.String()}))
	}
	var rows []model.Correction
	if err := q.Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list corrections: %w", err)
	}
	return rows, nil
}

// ForTargets returns active corrections overlapping the given target URIs, keyed
// by each matched target URI and ordered newest-first within each key.
func ForTargets(ctx context.Context, db *gorm.DB, targetURIs []string) (map[string][]Summary, error) {
	wanted := make(map[string]struct{}, len(targetURIs))
	for _, u := range targetURIs {
		if u != "" {
			wanted[u] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return map[string][]Summary{}, nil
	}
	list := make([]string, 0, len(wanted))
	for u := range wanted {
		list = append(list, u)
	}

	var rows []model.Correction
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL AND target_uris && ?", pgarray.TextArray(list)).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load corrections: %w", err)
	}

	out := make(map[string][]Summary)
	for _, r := range rows {
		sum := Summary{URI: r.URI, CreatedAt: r.CreatedAt}
		if r.Statement != nil {
			sum.Statement = *r.Statement
		}
		for _, t := range r.TargetURIs {
			if _, ok := wanted[t]; ok {
				out[t] = append(out[t], sum)
			}
		}
	}
	return out, nil
}

// normalizeTargets validates, trims, and de-duplicates target URIs.
func normalizeTargets(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		t := strings.TrimSpace(r)
		if t == "" {
			continue
		}
		u, err := uri.Parse(t)
		if err != nil {
			return nil, fmt.Errorf("%w: target %q: %v", ErrInvalidInput, r, err)
		}
		canonical := u.String()
		if _, dup := seen[canonical]; dup {
			continue
		}
		seen[canonical] = struct{}{}
		out = append(out, canonical)
	}
	return out, nil
}
