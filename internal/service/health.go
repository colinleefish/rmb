package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type HealthStatus struct {
	Status   string `json:"status"`
	DB       string `json:"db"`
	PGVector string `json:"pgvector"`
}

type HealthService struct {
	db *gorm.DB
}

func NewHealthService(db *gorm.DB) *HealthService {
	return &HealthService{db: db}
}

func (s *HealthService) Check(ctx context.Context) (HealthStatus, error) {
	var pgVersion string
	if err := s.db.WithContext(ctx).Raw(`SELECT version()`).Scan(&pgVersion).Error; err != nil {
		return HealthStatus{}, fmt.Errorf("query postgres version: %w", err)
	}

	var vectorVersion string
	_ = s.db.WithContext(ctx).Raw(
		`SELECT extversion FROM pg_extension WHERE extname='vector'`,
	).Scan(&vectorVersion).Error

	return HealthStatus{
		Status:   "ok",
		DB:       pgVersion,
		PGVector: vectorVersion,
	}, nil
}
