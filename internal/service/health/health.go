package health

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type Status struct {
	Status   string `json:"status"`
	DB       string `json:"db"`
	PGVector string `json:"pgvector"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Check(ctx context.Context) (Status, error) {
	var pgVersion string
	if err := s.db.WithContext(ctx).Raw(`SELECT version()`).Scan(&pgVersion).Error; err != nil {
		return Status{}, fmt.Errorf("query postgres version: %w", err)
	}

	var vectorVersion string
	_ = s.db.WithContext(ctx).Raw(
		`SELECT extversion FROM pg_extension WHERE extname='vector'`,
	).Scan(&vectorVersion).Error

	return Status{
		Status:   "ok",
		DB:       pgVersion,
		PGVector: vectorVersion,
	}, nil
}
