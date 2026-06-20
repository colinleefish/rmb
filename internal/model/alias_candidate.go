package model

import (
	"time"

	"github.com/google/uuid"
)

// AliasCandidate statuses. A candidate is proposed by the alias-suggest worker
// and never becomes a live alias on its own: a human/agent confirms (creating a
// real alias) or rejects (permanent rejection memory). See docs/aliases.md.
const (
	AliasCandidateStatusPending   = "pending"
	AliasCandidateStatusConfirmed = "confirmed"
	AliasCandidateStatusRejected  = "rejected"
)

// AliasCandidate is a machine-proposed alias pair awaiting human confirmation.
// The suggest worker writes one row per judged directed pair (the unique
// (alias_uri, canonical_uri) index gives rejection memory so a pair is judged at
// most once). Confirmation is the only path that writes a live alias.
type AliasCandidate struct {
	ID           uuid.UUID  `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	AliasURI     string     `gorm:"column:alias_uri;type:text;not null"`
	CanonicalURI string     `gorm:"column:canonical_uri;type:text;not null"`
	Similarity   *float64   `gorm:"column:similarity;type:double precision"`
	Verdict      *string    `gorm:"column:verdict;type:text"`
	Rationale    *string    `gorm:"column:rationale;type:text"`
	Status       string     `gorm:"column:status;type:text;not null;default:pending"`
	ResolvedAt   *time.Time `gorm:"column:resolved_at;type:timestamptz"`
	CreatedAt    time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;type:timestamptz;not null"`
}

func (AliasCandidate) TableName() string { return "alias_candidates" }
