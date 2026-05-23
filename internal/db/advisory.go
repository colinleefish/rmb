package db

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrySessionAdvisoryXactLock acquires a transaction-scoped advisory lock for one session.
func TrySessionAdvisoryXactLock(tx *gorm.DB, sessionID uuid.UUID) (bool, error) {
	var locked bool
	if err := tx.Raw(
		`SELECT pg_try_advisory_xact_lock(hashtextextended(CAST(? AS text), 0))`,
		sessionID.String(),
	).Scan(&locked).Error; err != nil {
		return false, fmt.Errorf("acquire advisory lock: %w", err)
	}
	return locked, nil
}
