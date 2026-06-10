// Package assertion implements the human authority layer: append-only
// corrections that overlay distilled memory. See docs/corrections.md.
package assertion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/db/pgarray"
	"github.com/colinleefish/mypast/internal/model"
	"github.com/colinleefish/mypast/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrInvalidInput marks a bad correction request (caller error → HTTP 400).
var ErrInvalidInput = errors.New("invalid assertion input")

// Summary is the recall/inspect overlay view of an active correction.
type Summary struct {
	URI       string    `json:"uri"`
	Kind      string    `json:"kind"`
	Statement string    `json:"statement"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateInput describes a new human correction. v1 supports correct + forget.
type CreateInput struct {
	Kind       string
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

// Create validates and inserts an append-only assertion, returning the new row.
func (s *Service) Create(ctx context.Context, in CreateInput) (model.Assertion, error) {
	kind := strings.TrimSpace(in.Kind)
	switch kind {
	case model.AssertionKindCorrect, model.AssertionKindForget:
	default:
		return model.Assertion{}, fmt.Errorf("%w: unsupported kind %q (v1: correct, forget)", ErrInvalidInput, in.Kind)
	}

	targets, err := normalizeTargets(in.TargetURIs)
	if err != nil {
		return model.Assertion{}, err
	}
	if len(targets) == 0 {
		return model.Assertion{}, fmt.Errorf("%w: %s requires at least one target URI", ErrInvalidInput, kind)
	}

	statement := strings.TrimSpace(in.Statement)
	if kind == model.AssertionKindCorrect && statement == "" {
		return model.Assertion{}, fmt.Errorf("%w: correct requires a statement", ErrInvalidInput)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return model.Assertion{}, fmt.Errorf("generate assertion id: %w", err)
	}
	now := s.now().UTC()
	row := model.Assertion{
		ID:         id,
		URI:        uri.BuildAssertion(id.String()),
		Author:     "human",
		Kind:       kind,
		TargetURIs: pgarray.TextArray(targets),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if statement != "" {
		row.Statement = &statement
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return model.Assertion{}, fmt.Errorf("insert assertion: %w", err)
	}
	return row, nil
}

// ForTargets returns active assertions overlapping the given target URIs, keyed
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

	var rows []model.Assertion
	if err := db.WithContext(ctx).
		Where("superseded_at IS NULL AND target_uris && ?", pgarray.TextArray(list)).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load assertions: %w", err)
	}

	out := make(map[string][]Summary)
	for _, r := range rows {
		sum := Summary{URI: r.URI, Kind: r.Kind, CreatedAt: r.CreatedAt}
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
