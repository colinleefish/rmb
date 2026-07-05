package inspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/service/correction"
	"github.com/colinleefish/rmb/internal/uri"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Cat(ctx context.Context, raw string, w io.Writer) error {
	u, err := uri.Parse(raw)
	if err != nil {
		return err
	}

	switch u.Scope {
	case uri.ScopeRoot:
		return s.catRoot(w)
	case uri.ScopeProfile:
		return s.catMemoryByURI(ctx, uri.BuildProfile(), w)
	case uri.ScopePrefs, uri.ScopeEntities, uri.ScopeEvents:
		return s.catMemoryByURI(ctx, u.String(), w)
	case uri.ScopeScenes:
		if len(u.Segments) == 0 {
			return fmt.Errorf("scene id required; use `tree %s` to list scenes", u.String())
		}
		return s.catScene(ctx, u.Segments[0], w)
	case uri.ScopeAtoms:
		if len(u.Segments) == 0 {
			return fmt.Errorf("atom id required; use `tree %s` to list atoms", u.String())
		}
		return s.catAtom(ctx, u.Segments[0], w)
	case uri.ScopeTurns:
		if len(u.Segments) == 0 {
			return fmt.Errorf("turn id required; use `tree %s` to list turns", u.String())
		}
		return s.catTurn(ctx, u.Segments[0], w)
	case uri.ScopeSessions:
		return s.catSessionPath(ctx, u, w)
	default:
		return fmt.Errorf("unsupported scope %q", u.Scope)
	}
}

func (s *Service) Tree(ctx context.Context, raw string, w io.Writer) error {
	u, err := uri.Parse(raw)
	if err != nil {
		return err
	}

	if u.IsRoot() && !u.IsContainer() {
		return s.treeRoot(w)
	}

	switch u.Scope {
	case uri.ScopeSessions:
		return s.treeSession(ctx, u, w)
	case uri.ScopeScenes, uri.ScopeAtoms, uri.ScopeTurns, uri.ScopePrefs, uri.ScopeEntities, uri.ScopeEvents, uri.ScopeProfile:
		return s.treeScope(ctx, u, w)
	default:
		return fmt.Errorf("unsupported tree prefix %q", u.String())
	}
}

func (s *Service) Meta(ctx context.Context, raw string, w io.Writer) error {
	u, err := uri.Parse(raw)
	if err != nil {
		return err
	}

	var payload any
	switch u.Scope {
	case uri.ScopeProfile:
		payload, err = s.metaMemory(ctx, uri.BuildProfile())
	case uri.ScopePrefs, uri.ScopeEntities, uri.ScopeEvents:
		payload, err = s.metaMemory(ctx, u.String())
	case uri.ScopeScenes:
		payload, err = s.metaScene(ctx, u.String())
	case uri.ScopeAtoms:
		payload, err = s.metaAtom(ctx, u.String())
	case uri.ScopeTurns:
		payload, err = s.metaTurn(ctx, u.String())
	case uri.ScopeSessions:
		payload, err = s.metaSessionPath(ctx, u)
	default:
		return fmt.Errorf("unsupported meta uri %q", u.String())
	}
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func (s *Service) catRoot(w io.Writer) error {
	_, err := fmt.Fprintln(w, "rmb root — use `rmb tree rmb://` to list scopes")
	return err
}

func (s *Service) treeRoot(w io.Writer) error {
	scopes := []string{
		uri.BuildProfile(),
		"rmb://sessions/",
		"rmb://turns/",
		"rmb://atoms/",
		"rmb://scenes/",
		"rmb://preferences/",
		"rmb://entities/",
		"rmb://events/",
	}
	for _, line := range scopes {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) catSessionPath(ctx context.Context, u uri.URI, w io.Writer) error {
	if len(u.Segments) == 0 {
		return errors.New("session id required")
	}
	sessionKey := strings.ToLower(u.Segments[0])

	switch classifySessionPath(u) {
	case sessionPathSession:
		var session model.Session
		if err := s.db.WithContext(ctx).
			Where("session_key = ?", sessionKey).
			Take(&session).Error; err != nil {
			return fmt.Errorf("load session: %w", err)
		}
		text := ""
		if session.Abstract != nil {
			text = *session.Abstract
		}
		_, err := io.WriteString(w, text)
		return err

	default:
		return fmt.Errorf("cat not supported for %q", u.String())
	}
}

type sessionPathKind int

const (
	sessionPathUnknown sessionPathKind = iota
	sessionPathSession
)

func classifySessionPath(u uri.URI) sessionPathKind {
	if len(u.Segments) == 1 {
		return sessionPathSession
	}
	return sessionPathUnknown
}

func (s *Service) treeSession(ctx context.Context, u uri.URI, w io.Writer) error {
	if len(u.Segments) == 0 {
		var sessions []model.Session
		if err := s.db.WithContext(ctx).
			Order("updated_at desc").
			Limit(200).
			Find(&sessions).Error; err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		for _, session := range sessions {
			line := uri.BuildSession(session.SessionKey) + "/"
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
		return nil
	}

	sessionKey := strings.ToLower(u.Segments[0])
	var session model.Session
	if err := s.db.WithContext(ctx).
		Where("session_key = ?", sessionKey).
		Take(&session).Error; err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	if len(u.Segments) == 1 && u.IsContainer() {
		if _, err := fmt.Fprintln(w, uri.BuildSession(sessionKey)); err != nil {
			return err
		}
		var turns []model.SessionTurn
		if err := s.db.WithContext(ctx).
			Where("session_id = ?", session.ID).
			Order("created_at asc, id asc").
			Find(&turns).Error; err != nil {
			return fmt.Errorf("load turns: %w", err)
		}
		for _, turn := range turns {
			line := uri.BuildTurn(turn.ID.String())
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}

		var atomURIs []string
		if err := s.db.WithContext(ctx).Model(&model.Atom{}).
			Where("session_id = ?", session.ID).
			Order("created_at asc").
			Pluck("uri", &atomURIs).Error; err != nil {
			return fmt.Errorf("load atoms: %w", err)
		}
		for _, line := range atomURIs {
			if _, err := fmt.Fprintln(w, line); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("tree not supported for %q", u.String())
}

func (s *Service) treeScope(ctx context.Context, u uri.URI, w io.Writer) error {
	switch u.Scope {
	case uri.ScopeProfile:
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.Memory{}).
			Where("category = ? AND superseded_at IS NULL", uri.ScopeProfile).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			_, err := fmt.Fprintln(w, uri.BuildProfile())
			return err
		}
		return nil
	case uri.ScopeScenes:
		var rows []model.Scene
		q := s.db.WithContext(ctx).Order("updated_at desc").Limit(200)
		if len(u.Segments) == 1 {
			q = q.Where("uri = ?", uri.BuildScene(u.Segments[0]))
		}
		if err := q.Find(&rows).Error; err != nil {
			return fmt.Errorf("list scenes: %w", err)
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, row.URI); err != nil {
				return err
			}
		}
		return nil
	case uri.ScopeAtoms:
		var rows []model.Atom
		q := s.db.WithContext(ctx).Order("created_at asc").Limit(200)
		if len(u.Segments) == 1 {
			q = q.Where("uri = ?", uri.BuildAtom(u.Segments[0]))
		}
		if err := q.Find(&rows).Error; err != nil {
			return fmt.Errorf("list atoms: %w", err)
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, row.URI); err != nil {
				return err
			}
		}
		return nil
	case uri.ScopeTurns:
		var rows []model.SessionTurn
		q := s.db.WithContext(ctx).Order("created_at asc").Limit(200)
		if len(u.Segments) == 1 {
			q = q.Where("id = ?", u.Segments[0])
		}
		if err := q.Find(&rows).Error; err != nil {
			return fmt.Errorf("list turns: %w", err)
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, uri.BuildTurn(row.ID.String())); err != nil {
				return err
			}
		}
		return nil
	default:
		var rows []model.Memory
		q := s.db.WithContext(ctx).
			Where("category = ? AND superseded_at IS NULL", u.Scope).
			Order("updated_at desc").
			Limit(200)
		if len(u.Segments) == 1 {
			q = q.Where("uri = ?", uri.BuildMemory(u.Scope, u.Segments[0]))
		}
		if err := q.Find(&rows).Error; err != nil {
			return fmt.Errorf("list memories: %w", err)
		}
		for _, row := range rows {
			if _, err := fmt.Fprintln(w, row.URI); err != nil {
				return err
			}
		}
		return nil
	}
}

func (s *Service) catMemoryByURI(ctx context.Context, target string, w io.Writer) error {
	var row model.Memory
	if err := s.db.WithContext(ctx).
		Where("uri = ? AND superseded_at IS NULL", target).
		Take(&row).Error; err != nil {
		return fmt.Errorf("load memory: %w", err)
	}
	text := ""
	if row.Body != nil {
		text = *row.Body
	}
	overlay, err := s.correctionsBlock(ctx, target)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, text); err != nil {
		return err
	}
	_, err = io.WriteString(w, overlay)
	return err
}

// correctionsBlock renders active human corrections for a target URI as a
// trailing annotation block (newest-first), or "" if none. This is the
// read-time overlay guarantee from docs/corrections.md.
func (s *Service) correctionsBlock(ctx context.Context, target string) (string, error) {
	byTarget, err := correction.ForTargets(ctx, s.db, []string{target})
	if err != nil {
		return "", err
	}
	sums := byTarget[target]
	if len(sums) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("\n\n--- corrections (human, newest first) ---\n")
	for _, c := range sums {
		fmt.Fprintf(&b, "⚑ CORRECTION (%s): %s\n", c.CreatedAt.UTC().Format("2006-01-02"), c.Statement)
	}
	return b.String(), nil
}

func (s *Service) catTurn(ctx context.Context, turnID string, w io.Writer) error {
	var row model.SessionTurn
	if err := s.db.WithContext(ctx).Where("id = ?", turnID).Take(&row).Error; err != nil {
		return fmt.Errorf("load turn: %w", err)
	}
	_, err := io.WriteString(w, row.MessagesJSONL)
	return err
}

func (s *Service) catAtom(ctx context.Context, atomID string, w io.Writer) error {
	var row model.Atom
	target := uri.BuildAtom(atomID)
	if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
		return fmt.Errorf("load atom: %w", err)
	}
	_, err := io.WriteString(w, row.Content)
	return err
}

func (s *Service) catScene(ctx context.Context, sceneID string, w io.Writer) error {
	var row model.Scene
	target := uri.BuildScene(sceneID)
	if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
		return fmt.Errorf("load scene: %w", err)
	}
	text := ""
	if row.Body != nil {
		text = *row.Body
	}
	_, err := io.WriteString(w, text)
	return err
}

func (s *Service) metaMemory(ctx context.Context, target string) (map[string]any, error) {
	var row model.Memory
	if err := s.db.WithContext(ctx).
		Where("uri = ? AND superseded_at IS NULL", target).
		Take(&row).Error; err != nil {
		return nil, fmt.Errorf("load memory: %w", err)
	}
	meta := map[string]any{
		"uri":                row.URI,
		"version":            row.Version,
		"category":           row.Category,
		"slug":               row.Slug,
		"abstract":           row.Abstract,
		"source_scene_uris":  row.SourceSceneURIs,
		"created_at":         row.CreatedAt,
		"updated_at":         row.UpdatedAt,
	}
	byTarget, err := correction.ForTargets(ctx, s.db, []string{target})
	if err != nil {
		return nil, err
	}
	if sums := byTarget[target]; len(sums) > 0 {
		meta["corrections"] = sums
	}
	return meta, nil
}

func (s *Service) metaScene(ctx context.Context, target string) (map[string]any, error) {
	var row model.Scene
	if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("load scene: %w", err)
	}
	return map[string]any{
		"uri":               row.URI,
		"session_id":        row.SessionID,
		"display_name":      row.DisplayName,
		"abstract":          row.Abstract,
		"source_atom_uris":  row.SourceAtomURIs,
		"created_at":        row.CreatedAt,
		"updated_at":        row.UpdatedAt,
	}, nil
}

func (s *Service) metaTurn(ctx context.Context, target string) (map[string]any, error) {
	u, err := uri.Parse(target)
	if err != nil {
		return nil, err
	}
	if len(u.Segments) != 1 {
		return nil, fmt.Errorf("turn id required")
	}
	var row model.SessionTurn
	if err := s.db.WithContext(ctx).Where("id = ?", u.Segments[0]).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("load turn: %w", err)
	}
	return map[string]any{
		"uri":        uri.BuildTurn(row.ID.String()),
		"session_id": row.SessionID,
		"created_at": row.CreatedAt,
		"updated_at": row.UpdatedAt,
	}, nil
}

func (s *Service) metaAtom(ctx context.Context, target string) (map[string]any, error) {
	var row model.Atom
	if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
		return nil, fmt.Errorf("load atom: %w", err)
	}
	return map[string]any{
		"uri":             row.URI,
		"session_id":      row.SessionID,
		"category":        row.Category,
		"priority":        row.Priority,
		"scene_name":      row.SceneName,
		"slug":            row.Slug,
		"source_turn_ids": row.SourceTurnIDs,
		"created_at":      row.CreatedAt,
		"updated_at":      row.UpdatedAt,
	}, nil
}

func (s *Service) metaSessionPath(ctx context.Context, u uri.URI) (map[string]any, error) {
	if len(u.Segments) == 0 {
		return nil, errors.New("session id required")
	}
	sessionKey := strings.ToLower(u.Segments[0])

	if len(u.Segments) == 1 {
		var session model.Session
		if err := s.db.WithContext(ctx).
			Where("session_key = ?", sessionKey).
			Take(&session).Error; err != nil {
			return nil, fmt.Errorf("load session: %w", err)
		}
		return map[string]any{
			"uri":         uri.BuildSession(sessionKey),
			"session_key": session.SessionKey,
			"status":      session.Status,
			"abstract":    session.Abstract,
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
		}, nil
	}

	return nil, fmt.Errorf("meta not supported for %q", u.String())
}

// SortedSessionKeys is used in tests.
func SortedSessionKeys(keys []string) []string {
	out := append([]string(nil), keys...)
	sort.Strings(out)
	return out
}
