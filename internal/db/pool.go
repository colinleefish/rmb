package db

import (
	"context"
	"fmt"
	"time"

	"github.com/colinleefish/mypast/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxIdleConns(4)
	sqlDB.SetMaxOpenConns(16)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.Session{}); err != nil {
		return fmt.Errorf("auto migrate sessions: %w", err)
	}
	if err := renameLegacySessionTurnMessagesColumn(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(&model.SessionTurn{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

func renameLegacySessionTurnMessagesColumn(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&model.SessionTurn{}) {
		return nil
	}

	if migrator.HasColumn(&model.SessionTurn{}, "messages_jsonl") {
		return nil
	}

	legacyColumns := []string{"message_json_l", "messages_json_l"}
	for _, oldName := range legacyColumns {
		if !migrator.HasColumn(&model.SessionTurn{}, oldName) {
			continue
		}
		if err := migrator.RenameColumn(&model.SessionTurn{}, oldName, "messages_jsonl"); err != nil {
			return fmt.Errorf(
				"rename session_turns.%s to messages_jsonl: %w",
				oldName,
				err,
			)
		}
		return nil
	}

	return nil
}
