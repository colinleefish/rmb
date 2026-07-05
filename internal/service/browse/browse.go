package browse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
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

// ListParams carries server-side pagination, search, and sort options for the
// browse list endpoints. Limit/Offset are assumed already clamped by the
// handler. Sort is matched against a per-entity allowlist (never interpolated
// raw), so it is safe even though it reaches an ORDER BY clause.
type ListParams struct {
	Limit  int
	Offset int
	Query  string
	Sort   string
	Order  string
}

// sortColumns maps the allowlisted sort keys (which equal the UI column ids) to
// their physical SQL columns, plus the default key when the request omits or
// sends an unknown sort. Keeping the allowlist here is the injection guard.
type sortColumns struct {
	allowed    map[string]string
	defaultKey string
}

func (sc sortColumns) clause(sort, order string) string {
	col, ok := sc.allowed[sort]
	if !ok {
		col = sc.allowed[sc.defaultKey]
	}
	dir := "DESC"
	if strings.EqualFold(order, "asc") {
		dir = "ASC"
	}
	return col + " " + dir
}

// applySearch ORs a case-insensitive substring match across the given columns.
// cols come from a fixed per-entity allowlist (not user input), so joining them
// into SQL is safe; the query value itself is always parameterized.
func applySearch(q *gorm.DB, query string, cols []string) *gorm.DB {
	query = strings.TrimSpace(query)
	if query == "" || len(cols) == 0 {
		return q
	}
	like := "%" + query + "%"
	parts := make([]string, len(cols))
	args := make([]any, len(cols))
	for i, c := range cols {
		parts[i] = c + " ILIKE ?"
		args[i] = like
	}
	return q.Where(strings.Join(parts, " OR "), args...)
}

var (
	memorySearchCols = []string{"abstract", "body", "slug", "uri", "category"}
	memorySort       = sortColumns{
		allowed:    map[string]string{"updated": "updated_at", "category": "category", "version": "version"},
		defaultKey: "updated",
	}

	atomSearchCols = []string{"content", "category", "scene_name", "slug", "uri"}
	atomSort       = sortColumns{
		allowed:    map[string]string{"created": "created_at", "category": "category", "priority": "priority"},
		defaultKey: "created",
	}

	sceneSearchCols = []string{"display_name", "abstract", "body", "CAST(id AS text)"}
	sceneSort       = sortColumns{
		allowed:    map[string]string{"updated": "updated_at", "created": "created_at"},
		defaultKey: "updated",
	}

	taskSearchCols = []string{"kind", "status", "error", "CAST(session_id AS text)"}
	taskSort       = sortColumns{
		allowed:    map[string]string{"created": "created_at", "kind": "kind", "status": "status", "progress": "progress"},
		defaultKey: "created",
	}

	sessionSearchCols = []string{"session_key", "abstract", "status"}
	sessionSort       = sortColumns{
		allowed:    map[string]string{"updated": "updated_at", "created": "created_at", "status": "status"},
		defaultKey: "updated",
	}
)

type Overview struct {
	Counts struct {
		Sessions       int64 `json:"sessions"`
		Turns          int64 `json:"turns"`
		Atoms          int64 `json:"atoms"`
		Scenes         int64 `json:"scenes"`
		Memories       int64 `json:"memories"`
		PipelineStates int64 `json:"pipeline_states"`
		Tasks          int64 `json:"tasks"`
		Corrections    int64 `json:"corrections"`
	} `json:"counts"`
}

type SessionRow struct {
	ID         uuid.UUID `json:"id"`
	SessionKey string    `json:"session_key"`
	Status     string    `json:"status"`
	Abstract   *string   `json:"abstract"`
	TurnCount  int64     `json:"turn_count"`
	AtomCount  int64     `json:"atom_count"`
	SceneCount int64     `json:"scene_count"`
	T1Status   string    `json:"t1_status,omitempty"`
	T2Status   string    `json:"t2_status,omitempty"`
	T3Status   string    `json:"t3_status,omitempty"`
	URI        string    `json:"uri"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
	LastTurnAt *string   `json:"last_turn_at"`
}

type sessionSummary struct {
	AtomCount  int64
	SceneCount int64
	T1Status   string
	T2Status   string
	T3Status   string
}

type SessionDetail struct {
	Session       SessionRow          `json:"session"`
	Turns         []TurnRow           `json:"turns"`
	PipelineState *model.PipelineState `json:"pipeline_state"`
	Atoms         []model.Atom        `json:"atoms"`
	Scenes        []model.Scene       `json:"scenes"`
}

type TurnRow struct {
	ID            uuid.UUID `json:"id"`
	TurnIndex     int       `json:"turn_index"`
	URI           string    `json:"uri"`
	MessagesJSONL string    `json:"messages_jsonl"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
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
		{&out.Counts.PipelineStates, "pipeline_state"},
		{&out.Counts.Tasks, "tasks"},
	}
	for _, t := range tables {
		if err := s.db.WithContext(ctx).Table(t.name).Count(t.dest).Error; err != nil {
			return Overview{}, fmt.Errorf("count %s: %w", t.name, err)
		}
	}
	// Memories are versioned (each rollup supersedes the prior row). Count only
	// active versions so the badge matches the distinct memories the list shows,
	// not the full version history.
	if err := s.db.WithContext(ctx).
		Table("memories").
		Where("superseded_at IS NULL").
		Count(&out.Counts.Memories).Error; err != nil {
		return Overview{}, fmt.Errorf("count memories: %w", err)
	}
	// Corrections are append-only; count only active (non-retracted) ones so the
	// badge matches what the corrections list shows.
	if err := s.db.WithContext(ctx).
		Table("corrections").
		Where("superseded_at IS NULL").
		Count(&out.Counts.Corrections).Error; err != nil {
		return Overview{}, fmt.Errorf("count corrections: %w", err)
	}
	return out, nil
}

func (s *Service) ListSessions(ctx context.Context, p ListParams) ([]SessionRow, int64, error) {
	base := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&model.Session{}).
			Joins(`LEFT JOIN (
				SELECT session_id, MAX(created_at) AS last_turn_at
				FROM session_turns
				GROUP BY session_id
			) AS turn_stats ON turn_stats.session_id = sessions.id`)
		return applySearch(q, p.Query, sessionSearchCols)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count sessions: %w", err)
	}

	limit := p.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}

	var sessions []model.Session
	if err := base().
		Order(sessionListOrderClause(p.Sort, p.Order)).
		Limit(limit).
		Offset(p.Offset).
		Find(&sessions).Error; err != nil {
		return nil, 0, fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) == 0 {
		return []SessionRow{}, total, nil
	}

	ids := make([]uuid.UUID, len(sessions))
	for i, session := range sessions {
		ids[i] = session.ID
	}

	type turnStatsRow struct {
		SessionID  uuid.UUID
		Count      int64
		LastTurnAt *time.Time `gorm:"column:last_turn_at"`
	}
	var turnStats []turnStatsRow
	if err := s.db.WithContext(ctx).Model(&model.SessionTurn{}).
		Select("session_id, COUNT(*) AS count, MAX(created_at) AS last_turn_at").
		Where("session_id IN ?", ids).
		Group("session_id").
		Scan(&turnStats).Error; err != nil {
		return nil, 0, fmt.Errorf("load turn stats: %w", err)
	}
	turnCounts := make(map[uuid.UUID]int64, len(turnStats))
	lastTurnAt := make(map[uuid.UUID]*time.Time, len(turnStats))
	for _, row := range turnStats {
		turnCounts[row.SessionID] = row.Count
		lastTurnAt[row.SessionID] = row.LastTurnAt
	}

	summaries, err := s.loadSessionSummaries(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	rows := make([]SessionRow, 0, len(sessions))
	for _, session := range sessions {
		rows = append(rows, sessionToRow(session, turnCounts[session.ID], lastTurnAt[session.ID], summaries[session.ID]))
	}
	return rows, total, nil
}

func sessionListOrderClause(sort, order string) string {
	dir := "DESC"
	if strings.EqualFold(order, "asc") {
		dir = "ASC"
	}
	switch sort {
	case "created":
		return "sessions.created_at " + dir
	case "status":
		return "sessions.status " + dir
	default:
		return "COALESCE(turn_stats.last_turn_at, sessions.created_at) " + dir
	}
}

func (s *Service) loadSessionSummaries(ctx context.Context, sessionIDs []uuid.UUID) (map[uuid.UUID]sessionSummary, error) {
	out := make(map[uuid.UUID]sessionSummary, len(sessionIDs))
	if len(sessionIDs) == 0 {
		return out, nil
	}

	type countRow struct {
		SessionID uuid.UUID
		Count     int64
	}

	var atomCounts []countRow
	if err := s.db.WithContext(ctx).Model(&model.Atom{}).
		Select("session_id, COUNT(*) AS count").
		Where("session_id IN ?", sessionIDs).
		Group("session_id").
		Scan(&atomCounts).Error; err != nil {
		return nil, fmt.Errorf("count atoms: %w", err)
	}
	for _, row := range atomCounts {
		summary := out[row.SessionID]
		summary.AtomCount = row.Count
		out[row.SessionID] = summary
	}

	var sceneCounts []countRow
	if err := s.db.WithContext(ctx).Model(&model.Scene{}).
		Select("session_id, COUNT(*) AS count").
		Where("session_id IN ?", sessionIDs).
		Group("session_id").
		Scan(&sceneCounts).Error; err != nil {
		return nil, fmt.Errorf("count scenes: %w", err)
	}
	for _, row := range sceneCounts {
		summary := out[row.SessionID]
		summary.SceneCount = row.Count
		out[row.SessionID] = summary
	}

	var pipelineRows []model.PipelineState
	if err := s.db.WithContext(ctx).
		Where("session_id IN ?", sessionIDs).
		Find(&pipelineRows).Error; err != nil {
		return nil, fmt.Errorf("load pipeline_state: %w", err)
	}
	for _, ps := range pipelineRows {
		summary := out[ps.SessionID]
		summary.T1Status = ps.T1Status
		summary.T2Status = ps.T2Status
		summary.T3Status = ps.T3Status
		out[ps.SessionID] = summary
	}

	return out, nil
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

	var summary sessionSummary
	summary.AtomCount = int64(len(atoms))
	summary.SceneCount = int64(len(scenes))
	if pipeline != nil {
		summary.T1Status = pipeline.T1Status
		summary.T2Status = pipeline.T2Status
		summary.T3Status = pipeline.T3Status
	}

	var lastTurnAt *time.Time
	if len(turns) > 0 {
		t := turns[len(turns)-1].CreatedAt
		lastTurnAt = &t
	}

	return SessionDetail{
		Session:       sessionToRow(session, int64(len(turns)), lastTurnAt, summary),
		Turns:         turnRows,
		PipelineState: pipeline,
		Atoms:         atoms,
		Scenes:        scenes,
	}, nil
}

func (s *Service) ListAtoms(ctx context.Context, p ListParams) ([]model.Atom, int64, error) {
	base := func() *gorm.DB {
		return applySearch(s.db.WithContext(ctx).Model(&model.Atom{}), p.Query, atomSearchCols)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count atoms: %w", err)
	}
	var rows []model.Atom
	if err := base().
		Order(atomSort.clause(p.Sort, p.Order)).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list atoms: %w", err)
	}
	return rows, total, nil
}

func (s *Service) ListScenes(ctx context.Context, p ListParams) ([]model.Scene, int64, error) {
	base := func() *gorm.DB {
		return applySearch(s.db.WithContext(ctx).Model(&model.Scene{}), p.Query, sceneSearchCols)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count scenes: %w", err)
	}
	var rows []model.Scene
	if err := base().
		Order(sceneSort.clause(p.Sort, p.Order)).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list scenes: %w", err)
	}
	return rows, total, nil
}

// ListMemories returns one page of active (current-version) memories plus the
// total active count. The superseded_at filter means the total reflects distinct
// memories (one per uri), not the full version history the badge used to show.
func (s *Service) ListMemories(ctx context.Context, p ListParams) ([]model.Memory, int64, error) {
	base := func() *gorm.DB {
		q := s.db.WithContext(ctx).Model(&model.Memory{}).Where("superseded_at IS NULL")
		return applySearch(q, p.Query, memorySearchCols)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count memories: %w", err)
	}
	var rows []model.Memory
	if err := base().
		Order(memorySort.clause(p.Sort, p.Order)).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list memories: %w", err)
	}
	return rows, total, nil
}

func (s *Service) ListPipelineStates(ctx context.Context, p ListParams) ([]PipelineStateRow, int64, error) {
	type row struct {
		model.PipelineState
		SessionKey string `gorm:"column:session_key"`
	}

	base := func() *gorm.DB {
		q := s.db.WithContext(ctx).
			Table("pipeline_state ps").
			Select("ps.*, s.session_key").
			Joins("JOIN sessions s ON s.id = ps.session_id")
		if t := strings.TrimSpace(p.Query); t != "" {
			like := "%" + t + "%"
			q = q.Where("s.session_key ILIKE ? OR ps.t1_status ILIKE ? OR ps.t2_status ILIKE ? OR ps.t3_status ILIKE ?",
				like, like, like, like)
		}
		return q
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pipeline_state: %w", err)
	}

	limit := p.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}

	var joined []row
	if err := base().
		Order("ps.updated_at desc").
		Limit(limit).
		Offset(p.Offset).
		Scan(&joined).Error; err != nil {
		return nil, 0, fmt.Errorf("list pipeline_state: %w", err)
	}

	out := make([]PipelineStateRow, 0, len(joined))
	for _, r := range joined {
		out = append(out, PipelineStateRow{
			PipelineState: r.PipelineState,
			SessionKey:    r.SessionKey,
			SessionURI:    uri.BuildSession(r.SessionKey),
		})
	}
	return out, total, nil
}

type PipelineStateRow struct {
	model.PipelineState
	SessionKey string `json:"session_key"`
	SessionURI string `json:"session_uri"`
}

func (s *Service) ListTasks(ctx context.Context, p ListParams) ([]model.Task, int64, error) {
	base := func() *gorm.DB {
		return applySearch(s.db.WithContext(ctx).Model(&model.Task{}), p.Query, taskSearchCols)
	}
	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}
	var rows []model.Task
	if err := base().
		Order(taskSort.clause(p.Sort, p.Order)).
		Limit(p.Limit).Offset(p.Offset).
		Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	return rows, total, nil
}

func sessionToRow(session model.Session, turnCount int64, lastTurnAt *time.Time, summary sessionSummary) SessionRow {
	row := SessionRow{
		ID:         session.ID,
		SessionKey: session.SessionKey,
		Status:     session.Status,
		Abstract:   session.Abstract,
		TurnCount:  turnCount,
		AtomCount:  summary.AtomCount,
		SceneCount: summary.SceneCount,
		T1Status:   summary.T1Status,
		T2Status:   summary.T2Status,
		T3Status:   summary.T3Status,
		URI:        uri.BuildSession(session.SessionKey),
		CreatedAt:  session.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:  session.UpdatedAt.UTC().Format(timeRFC3339),
	}
	if lastTurnAt != nil {
		formatted := lastTurnAt.UTC().Format(timeRFC3339)
		row.LastTurnAt = &formatted
	}
	return row
}

func turnToRow(sessionKey string, idx int, turn model.SessionTurn) TurnRow {
	return TurnRow{
		ID:            turn.ID,
		TurnIndex:     idx,
		URI:           uri.BuildTurn(turn.ID.String()),
		MessagesJSONL: turn.MessagesJSONL,
		CreatedAt:     turn.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:     turn.UpdatedAt.UTC().Format(timeRFC3339),
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"
