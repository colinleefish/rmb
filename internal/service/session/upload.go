package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/model"
	"github.com/colinleefish/mypast/internal/uri"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sessionCategory = "sessions"

var (
	ErrInvalidUploadInput = errors.New("invalid session upload input")
	uuidPattern           = regexp.MustCompile(
		`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
	)
)

type Message struct {
	Role    string
	Content string
}

type UploadInput struct {
	SessionID string
	ScopeKey  string
	Title     string
	StartedAt *time.Time
	Messages  []Message
}

type UploadResult struct {
	URI        string
	ParentURI  string
	RootURI    string
	Category   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	MessageCnt int
	ArchiveIdx int
	TaskID     *uuid.UUID
}

// ValidateSessionKey normalizes and validates an agent session UUID string.
func ValidateSessionKey(raw string) (string, error) {
	return validateSessionID(raw)
}

type UploadService struct {
	db               *gorm.DB
	now              func() time.Time
	pipelineOnUpload bool
}

func NewUploadService(db *gorm.DB) *UploadService {
	return &UploadService{
		db:               db,
		now:              time.Now,
		pipelineOnUpload: true,
	}
}

func NewUploadServiceWithOptions(db *gorm.DB, pipelineOnUpload bool) *UploadService {
	return &UploadService{
		db:               db,
		now:              time.Now,
		pipelineOnUpload: pipelineOnUpload,
	}
}

func (s *UploadService) Upload(ctx context.Context, input UploadInput) (UploadResult, error) {
	sessionID, err := validateSessionID(input.SessionID)
	if err != nil {
		return UploadResult{}, err
	}
	if len(input.Messages) == 0 {
		return UploadResult{}, fmt.Errorf("%w: messages must not be empty", ErrInvalidUploadInput)
	}

	for i, msg := range input.Messages {
		if strings.TrimSpace(msg.Role) == "" {
			return UploadResult{}, fmt.Errorf(
				"%w: messages[%d].role is required",
				ErrInvalidUploadInput,
				i,
			)
		}
	}

	now := s.now().UTC()
	rootURI := uri.BuildSession(sessionID)

	input.SessionID = sessionID
	archiveMessagesContent, err := buildMessagesJSONL(input.Messages, now)
	if err != nil {
		return UploadResult{}, err
	}

	title := normalizeNullableText(input.Title)
	scopeKey := normalizeNullableText(input.ScopeKey)

	var session model.Session
	var turn model.SessionTurn
	var archiveIdx int
	var taskID *uuid.UUID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedSession, err := s.findOrCreateSessionForUpdate(tx, sessionID, title, scopeKey)
		if err != nil {
			return err
		}
		session = lockedSession

		turnID, err := newUUIDv7()
		if err != nil {
			return err
		}

		turn = model.SessionTurn{
			ID:            turnID,
			SessionID:     session.ID,
			TurnStatus:    model.SessionTurnStatusNotSummarized,
			MessagesJSONL: archiveMessagesContent,
		}
		if err := tx.Create(&turn).Error; err != nil {
			return fmt.Errorf("insert session turn: %w", err)
		}

		var turnCount int64
		if err := tx.Model(&model.SessionTurn{}).
			Where("session_id = ? AND id <= ?", session.ID, turn.ID).
			Count(&turnCount).Error; err != nil {
			return fmt.Errorf("count archive index: %w", err)
		}
		if turnCount <= 0 {
			return fmt.Errorf("count archive index: unexpected non-positive turn count")
		}
		archiveIdx = int(turnCount) - 1

		if s.pipelineOnUpload {
			if err := s.markPipelinePending(tx, session.ID); err != nil {
				return err
			}
			id, err := newUUIDv7()
			if err != nil {
				return err
			}
			task := model.Task{
				ID:        id,
				Kind:      model.TaskKindT1,
				Status:    model.TaskStatusPending,
				SessionID: &session.ID,
			}
			if err := tx.Create(&task).Error; err != nil {
				return fmt.Errorf("insert t1 task: %w", err)
			}
			taskID = &id
		}
		return nil
	})
	if err != nil {
		return UploadResult{}, err
	}

	turnURI := uri.BuildSessionTurn(sessionID, archiveIdx)

	return UploadResult{
		URI:        turnURI,
		ParentURI:  uri.BuildSession(sessionID) + "/",
		RootURI:    rootURI,
		Category:   sessionCategory,
		CreatedAt:  turn.CreatedAt,
		UpdatedAt:  turn.UpdatedAt,
		MessageCnt: len(input.Messages),
		ArchiveIdx: archiveIdx,
		TaskID:     taskID,
	}, nil
}

func (s *UploadService) markPipelinePending(tx *gorm.DB, sessionID uuid.UUID) error {
	var ps model.PipelineState
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("session_id = ?", sessionID).
		Take(&ps).Error
	if err == gorm.ErrRecordNotFound {
		ps = model.PipelineState{
			SessionID:       sessionID,
			T1Status:        model.PipelineStatusPending,
			T2Status:        model.PipelineStatusIdle,
			T3Status:        model.PipelineStatusIdle,
			WarmupThreshold: 2,
		}
		return tx.Create(&ps).Error
	}
	if err != nil {
		return fmt.Errorf("load pipeline_state: %w", err)
	}
	return tx.Model(&ps).Updates(map[string]any{
		"t1_status":               model.PipelineStatusPending,
		"t1_turns_since_advanced": gorm.Expr("t1_turns_since_advanced + 1"),
	}).Error
}

func (s *UploadService) findOrCreateSessionForUpdate(
	tx *gorm.DB,
	sessionKey string,
	title *string,
	scopeKey *string,
) (model.Session, error) {
	var session model.Session
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("session_key = ?", sessionKey).
		Take(&session).Error; err == nil {
		if title != nil && (session.Title == nil || strings.TrimSpace(*session.Title) == "") {
			if err := tx.Model(&model.Session{}).
				Where("id = ?", session.ID).
				Update("title", *title).Error; err != nil {
				return model.Session{}, fmt.Errorf("update session title: %w", err)
			}
			session.Title = title
		}
		if scopeKey != nil && (session.ScopeKey == nil || strings.TrimSpace(*session.ScopeKey) == "") {
			if err := tx.Model(&model.Session{}).
				Where("id = ?", session.ID).
				Update("scope_key", *scopeKey).Error; err != nil {
				return model.Session{}, fmt.Errorf("update session scope_key: %w", err)
			}
			session.ScopeKey = scopeKey
		}
		return session, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Session{}, fmt.Errorf("load session: %w", err)
	}

	id, err := newUUIDv7()
	if err != nil {
		return model.Session{}, err
	}

	session = model.Session{
		ID:         id,
		SessionKey: sessionKey,
		ScopeKey:   scopeKey,
		Status:     "active",
		Title:      title,
	}
	if err := tx.Create(&session).Error; err != nil {
		if !isUniqueViolation(err) {
			return model.Session{}, fmt.Errorf("create session: %w", err)
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("session_key = ?", sessionKey).
			Take(&session).Error; err != nil {
			return model.Session{}, fmt.Errorf("load session after create conflict: %w", err)
		}
	}

	return session, nil
}

func normalizeNullableText(raw string) *string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	return &v
}

func newUUIDv7() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate uuidv7: %w", err)
	}
	return id, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type sessionMessageLine struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func buildMessagesJSONL(messages []Message, now time.Time) (string, error) {
	lines := make([]string, 0, len(messages))
	for i, msg := range messages {
		record := sessionMessageLine{
			ID:        fmt.Sprintf("msg_%06d", i+1),
			Role:      strings.TrimSpace(msg.Role),
			Content:   msg.Content,
			CreatedAt: now.Format(time.RFC3339Nano),
		}
		raw, err := json.Marshal(record)
		if err != nil {
			return "", fmt.Errorf("encode message[%d]: %w", i, err)
		}
		lines = append(lines, string(raw))
	}
	return strings.Join(lines, "\n") + "\n", nil
}

func validateSessionID(raw string) (string, error) {
	sessionID := strings.TrimSpace(raw)
	if sessionID == "" {
		return "", fmt.Errorf("%w: session_id is required", ErrInvalidUploadInput)
	}
	if !uuidPattern.MatchString(sessionID) {
		return "", fmt.Errorf("%w: session_id must be a valid UUID", ErrInvalidUploadInput)
	}
	return strings.ToLower(sessionID), nil
}
