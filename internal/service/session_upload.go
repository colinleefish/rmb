package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sessionCategory = "sessions"

var (
	ErrInvalidSessionUploadInput = errors.New("invalid session upload input")
	uuidPattern                  = regexp.MustCompile(
		`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
	)
)

type SessionMessage struct {
	Role    string
	Content string
}

type SessionUploadInput struct {
	SessionID string
	ScopeKey  string
	Title     string
	StartedAt *time.Time
	Messages  []SessionMessage
}

type SessionUploadResult struct {
	URI        string
	ParentURI  string
	RootURI    string
	Category   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	MessageCnt int
	ArchiveIdx int
}

type SessionUploadService struct {
	db  *gorm.DB
	now func() time.Time
}

func NewSessionUploadService(db *gorm.DB) *SessionUploadService {
	return &SessionUploadService{
		db:  db,
		now: time.Now,
	}
}

func (s *SessionUploadService) Upload(ctx context.Context, input SessionUploadInput) (SessionUploadResult, error) {
	sessionID, err := validateSessionID(input.SessionID)
	if err != nil {
		return SessionUploadResult{}, err
	}
	if len(input.Messages) == 0 {
		return SessionUploadResult{}, fmt.Errorf("%w: messages must not be empty", ErrInvalidSessionUploadInput)
	}

	for i, msg := range input.Messages {
		if strings.TrimSpace(msg.Role) == "" {
			return SessionUploadResult{}, fmt.Errorf(
				"%w: messages[%d].role is required",
				ErrInvalidSessionUploadInput,
				i,
			)
		}
	}

	now := s.now().UTC()
	rootURI := buildSessionRootURI(sessionID)

	input.SessionID = sessionID
	archiveMessagesContent, err := buildMessagesJSONL(input.Messages, now)
	if err != nil {
		return SessionUploadResult{}, err
	}

	title := normalizeNullableText(input.Title)
	scopeKey := normalizeNullableText(input.ScopeKey)

	var session model.Session
	var turn model.SessionTurn
	var archiveIdx int
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
		return nil
	})
	if err != nil {
		return SessionUploadResult{}, err
	}

	archiveMessagesURI := buildArchiveMessagesURI(rootURI, archiveIdx)

	return SessionUploadResult{
		URI:        archiveMessagesURI,
		ParentURI:  parentURIFromURI(archiveMessagesURI),
		RootURI:    rootURI,
		Category:   sessionCategory,
		CreatedAt:  turn.CreatedAt,
		UpdatedAt:  turn.UpdatedAt,
		MessageCnt: len(input.Messages),
		ArchiveIdx: archiveIdx,
	}, nil
}

func (s *SessionUploadService) findOrCreateSessionForUpdate(
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

func buildSessionRootURI(sessionID string) string {
	return "mypast://sessions/" + sessionID
}

func buildArchiveDirURI(rootURI string, archiveIdx int) string {
	return fmt.Sprintf("%s/history/%d", rootURI, archiveIdx)
}

func buildArchiveMessagesURI(rootURI string, archiveIdx int) string {
	return buildArchiveDirURI(rootURI, archiveIdx) + "/messages.jsonl"
}

func parentURIFromURI(uri string) string {
	idx := strings.LastIndex(uri, "/")
	if idx < len("mypast://x") {
		return "mypast://"
	}
	return uri[:idx]
}

type sessionMessageLine struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func buildMessagesJSONL(messages []SessionMessage, now time.Time) (string, error) {
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
		return "", fmt.Errorf("%w: session_id is required", ErrInvalidSessionUploadInput)
	}
	if !uuidPattern.MatchString(sessionID) {
		return "", fmt.Errorf("%w: session_id must be a valid UUID", ErrInvalidSessionUploadInput)
	}
	return strings.ToLower(sessionID), nil
}
