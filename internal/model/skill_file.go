package model

import (
	"time"

	"github.com/google/uuid"
)

type SkillFile struct {
	ID            uuid.UUID `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	SkillID       uuid.UUID `gorm:"column:skill_id;type:uuid;not null;index" json:"skill_id"`
	RelPath       string    `gorm:"column:rel_path;type:text;not null" json:"rel_path"`
	Content       string    `gorm:"column:content;type:text;not null" json:"content"`
	ByteSize      int       `gorm:"column:byte_size;type:int;not null" json:"byte_size"`
	ContentSHA256 string    `gorm:"column:content_sha256;type:text;not null" json:"content_sha256"`
	CreatedAt     time.Time `gorm:"column:created_at;type:timestamptz;not null" json:"created_at"`
}

func (SkillFile) TableName() string { return "skill_files" }
