package alias

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/mem9/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CandidateSummary is the review view of a proposed alias.
type CandidateSummary struct {
	ID           string    `json:"id"`
	AliasURI     string    `json:"alias_uri"`
	CanonicalURI string    `json:"canonical_uri"`
	Similarity   float64   `json:"similarity"`
	Verdict      string    `json:"verdict,omitempty"`
	Rationale    string    `json:"rationale,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// ListCandidates returns alias candidates filtered by status (default
// "pending"), newest-first. Pass "all" to return every status.
func (s *Service) ListCandidates(ctx context.Context, status string) ([]CandidateSummary, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = model.AliasCandidateStatusPending
	}
	q := s.db.WithContext(ctx).Model(&model.AliasCandidate{})
	switch status {
	case "all":
	case model.AliasCandidateStatusPending,
		model.AliasCandidateStatusConfirmed,
		model.AliasCandidateStatusRejected:
		q = q.Where("status = ?", status)
	default:
		return nil, fmt.Errorf("%w: unknown status %q (want pending|confirmed|rejected|all)", ErrInvalidInput, status)
	}
	var rows []model.AliasCandidate
	if err := q.Order("created_at DESC").Limit(200).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list alias candidates: %w", err)
	}
	out := make([]CandidateSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, candidateSummary(r))
	}
	return out, nil
}

// ConfirmCandidate promotes a pending candidate into a live alias: it creates
// the alias (with the candidate's rationale as the note) and marks the candidate
// confirmed. It returns the created alias so callers can run the same post-write
// side-effects as a manual `alias set` (wake T3, supersede the alias slug).
// Invariant violations from Create (e.g. ErrConflict) are propagated and the
// candidate is left pending.
func (s *Service) ConfirmCandidate(ctx context.Context, candidateID string) (model.Alias, error) {
	id, err := uuid.Parse(strings.TrimSpace(candidateID))
	if err != nil {
		return model.Alias{}, fmt.Errorf("%w: candidate id: %v", ErrInvalidInput, err)
	}
	var cand model.AliasCandidate
	err = s.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, model.AliasCandidateStatusPending).
		Take(&cand).Error
	if err == gorm.ErrRecordNotFound {
		return model.Alias{}, fmt.Errorf("%w: no pending candidate %s", ErrNotFound, id)
	}
	if err != nil {
		return model.Alias{}, fmt.Errorf("load candidate: %w", err)
	}

	note := ""
	if cand.Rationale != nil {
		note = *cand.Rationale
	}
	row, err := s.Create(ctx, CreateInput{
		AliasURI:     cand.AliasURI,
		CanonicalURI: cand.CanonicalURI,
		Note:         note,
	})
	if err != nil {
		return model.Alias{}, err
	}

	if err := s.db.WithContext(ctx).
		Model(&model.AliasCandidate{}).
		Where("id = ?", cand.ID).
		Updates(map[string]any{
			"status":      model.AliasCandidateStatusConfirmed,
			"resolved_at": s.now().UTC(),
			"updated_at":  s.now().UTC(),
		}).Error; err != nil {
		// The alias is already live and durable; only the candidate bookkeeping
		// failed. Surface it so the caller can log, but the alias stands.
		return row, fmt.Errorf("mark candidate confirmed: %w", err)
	}
	return row, nil
}

// RejectCandidate marks a pending candidate rejected. The unique pair index then
// guarantees the same directed pair is never re-proposed.
func (s *Service) RejectCandidate(ctx context.Context, candidateID string) error {
	id, err := uuid.Parse(strings.TrimSpace(candidateID))
	if err != nil {
		return fmt.Errorf("%w: candidate id: %v", ErrInvalidInput, err)
	}
	res := s.db.WithContext(ctx).
		Model(&model.AliasCandidate{}).
		Where("id = ? AND status = ?", id, model.AliasCandidateStatusPending).
		Updates(map[string]any{
			"status":      model.AliasCandidateStatusRejected,
			"resolved_at": s.now().UTC(),
			"updated_at":  s.now().UTC(),
		})
	if res.Error != nil {
		return fmt.Errorf("reject candidate: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w: no pending candidate %s", ErrNotFound, id)
	}
	return nil
}

func candidateSummary(r model.AliasCandidate) CandidateSummary {
	sum := CandidateSummary{
		ID:           r.ID.String(),
		AliasURI:     r.AliasURI,
		CanonicalURI: r.CanonicalURI,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
	}
	if r.Similarity != nil {
		sum.Similarity = *r.Similarity
	}
	if r.Verdict != nil {
		sum.Verdict = *r.Verdict
	}
	if r.Rationale != nil {
		sum.Rationale = *r.Rationale
	}
	return sum
}
