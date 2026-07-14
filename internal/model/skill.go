package model

import (
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/google/uuid"
)

type Skill struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	Slug          string     `gorm:"column:slug;type:text;not null;index" json:"slug"`
	URI           string     `gorm:"column:uri;type:text;not null;index" json:"uri"`
	Version       int        `gorm:"column:version;not null;default:1" json:"version"`
	SupersededAt  *time.Time `gorm:"column:superseded_at;type:timestamptz" json:"superseded_at,omitempty"`
	Name          string     `gorm:"column:name;type:text;not null" json:"name"`
	Description   string            `gorm:"column:description;type:text;not null" json:"description"`
	Tags          pgarray.TextArray `gorm:"column:tags;type:text[];not null" json:"tags"`
	BundleSHA256  string            `gorm:"column:bundle_sha256;type:text;not null" json:"bundle_sha256"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;not null" json:"updated_at"`
	Files         []SkillFile `gorm:"foreignKey:SkillID" json:"files,omitempty"`
}

func (Skill) TableName() string { return "skills" }
