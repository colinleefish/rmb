package browse

import (
	"context"
	"errors"
	"fmt"

	"github.com/colinleefish/mypast/internal/model"
	"github.com/colinleefish/mypast/internal/uri"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultListLimit = 300

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type Overview struct {
	Counts struct {
		Sessions       int64 `json:"sessions"`
		Turns          int64 `json:"turns"`
		Atoms          int64 `json:"atoms"`
		Scenes         int64 `json:"scenes"`
		Memories       int64 `json:"memories"`
		PipelineStates int64 `json:"pipeline_states"`
		Tasks          int64 `json:"tasks"`
		Assertions     int64 `json:"assertions"`
	} `json:"counts"`
}

type SessionRow struct {
	ID         uuid.UUID `json:"id"`
	SessionKey string    `json:"session_key"`
	ScopeKey   *string   `json:"scope_key"`
	Title      *string   `json:"title"`
	Status     string    `json:"status"`
	Abstract   *string   `json:"abstract"`
	Overview   *string   `json:"overview_text"`
	TurnCount  int64     `json:"turn_count"`
	URI        string    `json:"uri"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
}

type SessionDetail struct {
	Session       SessionRow          `json:"session"`
	Turns         []TurnRow           `json:"turns"`
	PipelineState *model.PipelineState `json:"pipeline_state"`
	Atoms         []model.Atom        `json:"atoms"`
	Scenes        []model.Scene       `json:"scenes"`
}

type TurnRow struct {
	ID                 uuid.UUID `json:"id"`
	TurnIndex          int       `json:"turn_index"`
	URI                string    `json:"uri"`
	TurnStatus         string    `json:"turn_status"`
	SummarizeStartedAt *string   `json:"summarize_started_at"`
	MessagesJSONL      string    `json:"messages_jsonl"`
	CreatedAt          string    `json:"created_at"`
	UpdatedAt          string    `json:"updated_at"`
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	var out Overview
	tables := []struct {
		dest *int64
		name string
	}{
		{&out.Counts.Sessions, "sessions"},
		{&out.Counts.Turns, "session_turns"},
		{&out.Counts.Atoms, "atoms"},
		{&out.Counts.Scenes, "scenes"},
		{&out.Counts.Memories, "memories"},
		{&out.Counts.PipelineStates, "pipeline_state"},
		{&out.Counts.Tasks, "tasks"},
	}
	for _, t := range tables {
		if err := s.db.WithContext(ctx).Table(t.name).Count(t.dest).Error; err != nil {
			return Overview{}, fmt.Errorf("count %s: %w", t.name, err)
		}
	}
	// Assertions are append-only; count only active (non-retracted) ones so the
	// badge matches what the assertions list shows.
	if err := s.db.WithContext(ctx).
		Table("assertions").
		Where("superseded_at IS NULL").
		Count(&out.Counts.Assertions).Error; err != nil {
		return Overview{}, fmt.Errorf("count assertions: %w", err)
	}
	return out, nil
}

func (s *Service) ListSessions(ctx context.Context) ([]SessionRow, error) {
	var sessions []model.Session
	if err := s.db.WithContext(ctx).
		Order("updated_at desc").
		Limit(defaultListLimit).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	rows := make([]SessionRow, 0, len(sessions))
	for _, session := range sessions {
		var turnCount int64
		if err := s.db.WithContext(ctx).Model(&model.SessionTurn{}).
			Where("session_id = ?", session.ID).
			Count(&turnCount).Error; err != nil {
			return nil, fmt.Errorf("count turns: %w", err)
		}
		rows = append(rows, sessionToRow(session, turnCount))
	}
	return rows, nil
}

func (s *Service) GetSession(ctx context.Context, sessionKey string) (SessionDetail, error) {
	var session model.Session
	if err := s.db.WithContext(ctx).
		Where("session_key = ?", sessionKey).
		Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return SessionDetail{}, gorm.ErrRecordNotFound
		}
		return SessionDetail{}, fmt.Errorf("load session: %w", err)
	}

	var turns []model.SessionTurn
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", session.ID).
		Order("created_at asc, id asc").
		Find(&turns).Error; err != nil {
		return SessionDetail{}, fmt.Errorf("load turns: %w", err)
	}

	turnRows := make([]TurnRow, 0, len(turns))
	for i, turn := range turns {
		turnRows = append(turnRows, turnToRow(session.SessionKey, i, turn))
	}

	var pipeline *model.PipelineState
	var ps model.PipelineState
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", session.ID).
		Take(&ps).Error; err == nil {
		pipeline = &ps
	} else if err != gorm.ErrRecordNotFound {
		return SessionDetail{}, fmt.Errorf("load pipeline_state: %w", err)
	}

	var atoms []model.Atom
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", session.ID).
		Order("created_at asc").
		Find(&atoms).Error; err != nil {
		return SessionDetail{}, fmt.Errorf("load atoms: %w", err)
	}

	var scenes []model.Scene
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", session.ID).
		Order("created_at asc, uri asc").
		Find(&scenes).Error; err != nil {
		return SessionDetail{}, fmt.Errorf("load scenes: %w", err)
	}

	return SessionDetail{
		Session:       sessionToRow(session, int64(len(turns))),
		Turns:         turnRows,
		PipelineState: pipeline,
		Atoms:         atoms,
		Scenes:        scenes,
	}, nil
}

func (s *Service) ListAtoms(ctx context.Context) ([]model.Atom, error) {
	var rows []model.Atom
	if err := s.db.WithContext(ctx).
		Order("created_at desc").
		Limit(defaultListLimit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list atoms: %w", err)
	}
	return rows, nil
}

func (s *Service) ListScenes(ctx context.Context) ([]model.Scene, error) {
	var rows []model.Scene
	if err := s.db.WithContext(ctx).
		Order("updated_at desc").
		Limit(defaultListLimit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list scenes: %w", err)
	}
	return rows, nil
}

func (s *Service) ListMemories(ctx context.Context) ([]model.Memory, error) {
	var rows []model.Memory
	if err := s.db.WithContext(ctx).
		Where("superseded_at IS NULL").
		Order("updated_at desc").
		Limit(defaultListLimit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	return rows, nil
}

func (s *Service) ListPipelineStates(ctx context.Context) ([]PipelineStateRow, error) {
	type row struct {
		model.PipelineState
		SessionKey string `gorm:"column:session_key"`
	}

	var joined []row
	if err := s.db.WithContext(ctx).
		Table("pipeline_state ps").
		Select("ps.*, s.session_key").
		Joins("JOIN sessions s ON s.id = ps.session_id").
		Order("ps.updated_at desc").
		Limit(defaultListLimit).
		Scan(&joined).Error; err != nil {
		return nil, fmt.Errorf("list pipeline_state: %w", err)
	}

	out := make([]PipelineStateRow, 0, len(joined))
	for _, r := range joined {
		out = append(out, PipelineStateRow{
			PipelineState: r.PipelineState,
			SessionKey:    r.SessionKey,
			SessionURI:    uri.BuildSession(r.SessionKey),
		})
	}
	return out, nil
}

type PipelineStateRow struct {
	model.PipelineState
	SessionKey string `json:"session_key"`
	SessionURI string `json:"session_uri"`
}

func (s *Service) ListTasks(ctx context.Context) ([]model.Task, error) {
	var rows []model.Task
	if err := s.db.WithContext(ctx).
		Order("created_at desc").
		Limit(defaultListLimit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return rows, nil
}

func sessionToRow(session model.Session, turnCount int64) SessionRow {
	return SessionRow{
		ID:         session.ID,
		SessionKey: session.SessionKey,
		ScopeKey:   session.ScopeKey,
		Title:      session.Title,
		Status:     session.Status,
		Abstract:   session.Abstract,
		Overview:   session.OverviewText,
		TurnCount:  turnCount,
		URI:        uri.BuildSession(session.SessionKey),
		CreatedAt:  session.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:  session.UpdatedAt.UTC().Format(timeRFC3339),
	}
}

func turnToRow(sessionKey string, idx int, turn model.SessionTurn) TurnRow {
	var started *string
	if turn.SummarizeStartedAt != nil {
		v := turn.SummarizeStartedAt.UTC().Format(timeRFC3339)
		started = &v
	}
	return TurnRow{
		ID:                 turn.ID,
		TurnIndex:          idx,
		URI:                uri.BuildSessionTurn(sessionKey, idx),
		TurnStatus:         turn.TurnStatus,
		SummarizeStartedAt: started,
		MessagesJSONL:      turn.MessagesJSONL,
		CreatedAt:          turn.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:          turn.UpdatedAt.UTC().Format(timeRFC3339),
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
