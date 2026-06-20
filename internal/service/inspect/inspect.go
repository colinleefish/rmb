package inspect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/colinleefish/mem9/internal/model"
	"github.com/colinleefish/mem9/internal/service/alias"
	"github.com/colinleefish/mem9/internal/service/correction"
	"github.com/colinleefish/mem9/internal/uri"
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
	case uri.ScopeScenes, uri.ScopePrefs, uri.ScopeEntities, uri.ScopeEvents, uri.ScopeProfile:
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
	_, err := fmt.Fprintln(w, "mem9 root — use `mem9 tree mem9://` to list scopes")
	return err
}

func (s *Service) treeRoot(w io.Writer) error {
	scopes := []string{
		uri.BuildProfile(),
		"mem9://sessions/",
		"mem9://scenes/",
		"mem9://preferences/",
		"mem9://entities/",
		"mem9://events/",
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

	case sessionPathTurn:
		var session model.Session
		if err := s.db.WithContext(ctx).
			Where("session_key = ?", sessionKey).
			Take(&session).Error; err != nil {
			return fmt.Errorf("load session: %w", err)
		}

		var turns []model.SessionTurn
		if err := s.db.WithContext(ctx).
			Where("session_id = ?", session.ID).
			Order("created_at asc, id asc").
			Find(&turns).Error; err != nil {
			return fmt.Errorf("load turns: %w", err)
		}

		idx, err := parseTurnIndex(u.Segments[2])
		if err != nil {
			return err
		}
		if idx < 0 || idx >= len(turns) {
			return fmt.Errorf("turn index %d out of range (have %d turns)", idx, len(turns))
		}
		_, err = io.WriteString(w, turns[idx].MessagesJSONL)
		return err

	case sessionPathAtom:
		target := uri.BuildSessionAtom(sessionKey, u.Segments[2])
		var row model.Atom
		if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
			return fmt.Errorf("load atom: %w", err)
		}
		_, err := io.WriteString(w, row.Content)
		return err

	default:
		return fmt.Errorf("cat not supported for %q", u.String())
	}
}

// sessionPathKind classifies the shape of a mem9://sessions/... URI so that
// the cat dispatch (and any future reader) can route by intent rather than
// re-deriving segment arithmetic at every call site.
type sessionPathKind int

const (
	sessionPathUnknown sessionPathKind = iota
	sessionPathSession
	sessionPathTurn
	sessionPathAtom
)

func classifySessionPath(u uri.URI) sessionPathKind {
	switch {
	case len(u.Segments) == 1:
		return sessionPathSession
	case len(u.Segments) == 3 && u.Segments[1] == "turns":
		return sessionPathTurn
	case len(u.Segments) == 3 && u.Segments[1] == "atoms":
		return sessionPathAtom
	default:
		return sessionPathUnknown
	}
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
		for i := range turns {
			line := uri.BuildSessionTurn(sessionKey, i)
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
	// If target is an active alias, redirect to the canonical: the alias slug's
	// own memory row is retired once folded, so we show the canonical instead of
	// erroring on the missing active row. See docs/aliases.md.
	if canonical, isAlias, err := alias.CanonicalFor(ctx, s.db, target); err != nil {
		return err
	} else if isAlias {
		fmt.Fprintf(w, "This entity is an alias of %s.\n\n\u2192 ALIAS OF: %s\n(showing canonical below)\n\n", canonical, canonical)
		target = canonical
	}

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
	aliasOverlay, err := s.aliasBlock(ctx, target)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, text); err != nil {
		return err
	}
	if _, err := io.WriteString(w, overlay); err != nil {
		return err
	}
	_, err = io.WriteString(w, aliasOverlay)
	return err
}

// aliasBlock renders active alias relationships for a memory URI as a trailing
// annotation: an "ALIAS OF" line when target is itself an alias, and/or an
// "aliases (point here)" list when target is a canonical. Returns "" if neither.
func (s *Service) aliasBlock(ctx context.Context, target string) (string, error) {
	aliasOf, aliasesOf, err := alias.ForTargets(ctx, s.db, []string{target})
	if err != nil {
		return "", err
	}
	out, isAlias := aliasOf[target]
	pointers := aliasesOf[target]
	if !isAlias && len(pointers) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("\n\n--- aliases ---\n")
	if isAlias {
		fmt.Fprintf(&b, "\u2192 ALIAS OF: %s\n", out.CanonicalURI)
	}
	for _, p := range pointers {
		line := fmt.Sprintf("\u2190 %s", p.AliasURI)
		if p.Note != "" {
			line += " (" + p.Note + ")"
		}
		b.WriteString(line + "\n")
	}
	return b.String(), nil
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
	// Redirect an active alias to its canonical (its own row is retired once
	// folded); record the redirect as alias_of below.
	aliasRedirect := ""
	if canonical, isAlias, err := alias.CanonicalFor(ctx, s.db, target); err != nil {
		return nil, err
	} else if isAlias {
		aliasRedirect = canonical
		target = canonical
	}

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
	_, aliasesOf, err := alias.ForTargets(ctx, s.db, []string{target})
	if err != nil {
		return nil, err
	}
	if aliasRedirect != "" {
		meta["alias_of"] = aliasRedirect
	}
	if ptrs := aliasesOf[target]; len(ptrs) > 0 {
		uris := make([]string, 0, len(ptrs))
		for _, p := range ptrs {
			uris = append(uris, p.AliasURI)
		}
		meta["aliases"] = uris
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
			"uri":           uri.BuildSession(sessionKey),
			"session_key":   session.SessionKey,
			"scope_key":     session.ScopeKey,
			"title":         session.Title,
			"status":        session.Status,
			"abstract":      session.Abstract,
			"created_at":    session.CreatedAt,
			"updated_at":    session.UpdatedAt,
		}, nil
	}

	if len(u.Segments) == 3 && u.Segments[1] == "turns" {
		var session model.Session
		if err := s.db.WithContext(ctx).
			Where("session_key = ?", sessionKey).
			Take(&session).Error; err != nil {
			return nil, fmt.Errorf("load session: %w", err)
		}
		var turns []model.SessionTurn
		if err := s.db.WithContext(ctx).
			Where("session_id = ?", session.ID).
			Order("created_at asc, id asc").
			Find(&turns).Error; err != nil {
			return nil, fmt.Errorf("load turns: %w", err)
		}
		idx, err := parseTurnIndex(u.Segments[2])
		if err != nil {
			return nil, err
		}
		if idx < 0 || idx >= len(turns) {
			return nil, fmt.Errorf("turn index %d out of range", idx)
		}
		turn := turns[idx]
		return map[string]any{
			"uri":        uri.BuildSessionTurn(sessionKey, idx),
			"session_id": turn.SessionID,
			"created_at": turn.CreatedAt,
			"updated_at": turn.UpdatedAt,
		}, nil
	}

	if len(u.Segments) == 3 && u.Segments[1] == "atoms" {
		target := uri.BuildSessionAtom(sessionKey, u.Segments[2])
		var row model.Atom
		if err := s.db.WithContext(ctx).Where("uri = ?", target).Take(&row).Error; err != nil {
			return nil, fmt.Errorf("load atom: %w", err)
		}
		return map[string]any{
			"uri":              row.URI,
			"session_id":       row.SessionID,
			"category":         row.Category,
			"priority":         row.Priority,
			"scene_name":       row.SceneName,
			"slug":             row.Slug,
			"source_turn_ids":  row.SourceTurnIDs,
			"created_at":       row.CreatedAt,
			"updated_at":       row.UpdatedAt,
		}, nil
	}

	return nil, fmt.Errorf("meta not supported for %q", u.String())
}

func parseTurnIndex(raw string) (int, error) {
	var idx int
	if _, err := fmt.Sscanf(raw, "%d", &idx); err != nil {
		return 0, fmt.Errorf("invalid turn index %q", raw)
	}
	return idx, nil
}

// SortedSessionKeys is used in tests.
func SortedSessionKeys(keys []string) []string {
	out := append([]string(nil), keys...)
	sort.Strings(out)
	return out
}
