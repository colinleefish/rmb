// Package agentmemory manages the curated rmb://agent singleton (not T3-distilled).
package agentmemory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/db/pgarray"
	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const targetURI = "rmb://agent"

// ReplaceBody versions the active agent memory row (supersede + insert).
func ReplaceBody(ctx context.Context, db *gorm.DB, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("body must not be empty")
	}
	now := time.Now().UTC()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var active model.Memory
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("uri = ? AND superseded_at IS NULL", targetURI).
			Take(&active).Error
		version := 1
		switch {
		case err == gorm.ErrRecordNotFound:
			// first write
		case err != nil:
			return fmt.Errorf("load active agent memory: %w", err)
		default:
			if active.Body != nil && strings.TrimSpace(*active.Body) == body {
				return nil
			}
			if err := tx.Model(&model.Memory{}).
				Where("id = ?", active.ID).
				Update("superseded_at", now).Error; err != nil {
				return fmt.Errorf("supersede agent memory: %w", err)
			}
			version = active.Version + 1
		}

		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("generate memory id: %w", err)
		}
		abstract := "Agent recall guide"
		row := model.Memory{
			ID:                   id,
			URI:                  targetURI,
			Category:             uri.ScopeAgent,
			Version:              version,
			Abstract:             &abstract,
			Body:                 &body,
			SourceSceneURIs:      pgarray.TextArray{},
			SourceCorrectionURIs: pgarray.TextArray{},
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("insert agent memory: %w", err)
		}
		return nil
	})
}
