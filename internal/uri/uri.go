package uri

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	Scheme           = "rmb"
	MaxSegment       = 50
	ScopeRoot        = ""
	ScopeSessions    = "sessions"
	ScopeTurns       = "turns"
	ScopeAtoms       = "atoms"
	ScopeScenes      = "scenes"
	ScopeProfile     = "profile"
	ScopePrefs       = "preferences"
	ScopeEntities    = "entities"
	ScopeEvents      = "events"
	ScopeCorrections = "corrections"
)

var (
	ErrInvalidURI = errors.New("invalid rmb uri")
	uuidSegment   = regexp.MustCompile(
		`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`,
	)
	reservedSlug = map[string]struct{}{
		ScopeSessions:    {},
		ScopeTurns:       {},
		ScopeAtoms:       {},
		ScopeScenes:      {},
		ScopeProfile:     {},
		ScopePrefs:       {},
		ScopeEntities:    {},
		ScopeEvents:      {},
		ScopeCorrections: {},
	}
)

type URI struct {
	Scope     string
	Segments  []string
	Container bool
}

func Parse(raw string) (URI, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return URI{}, fmt.Errorf("%w: empty", ErrInvalidURI)
	}

	if strings.Contains(s, "{") || strings.Contains(s, "}") {
		return URI{}, fmt.Errorf("%w: reserved template syntax", ErrInvalidURI)
	}

	switch {
	case strings.HasPrefix(s, Scheme+"://"):
		s = strings.TrimPrefix(s, Scheme+"://")
	case strings.HasPrefix(s, "/"):
		s = strings.TrimPrefix(s, "/")
	case strings.HasPrefix(s, Scheme+":"):
		return URI{}, fmt.Errorf("%w: missing // after scheme", ErrInvalidURI)
	}

	container := strings.HasSuffix(s, "/")
	s = strings.TrimSuffix(s, "/")

	parts := splitSegments(s)
	if len(parts) == 0 {
		return URI{Scope: ScopeRoot, Container: container}, nil
	}

	scope := parts[0]
	if err := validateScope(scope); err != nil {
		return URI{}, err
	}

	segments := parts[1:]
	for _, seg := range segments {
		if err := validateSegment(seg); err != nil {
			return URI{}, err
		}
	}

	if err := validateShape(scope, segments); err != nil {
		return URI{}, err
	}

	return URI{
		Scope:     scope,
		Segments:  segments,
		Container: container,
	}, nil
}

func (u URI) String() string {
	if u.Scope == ScopeRoot && len(u.Segments) == 0 {
		if u.Container {
			return Scheme + "://"
		}
		return Scheme + "://"
	}

	var b strings.Builder
	b.WriteString(Scheme)
	b.WriteString("://")
	b.WriteString(u.Scope)
	for _, seg := range u.Segments {
		b.WriteByte('/')
		b.WriteString(seg)
	}
	if u.Container {
		b.WriteByte('/')
	}
	return b.String()
}

func (u URI) Parent() (URI, bool) {
	if len(u.Segments) == 0 {
		if u.Scope == ScopeRoot {
			return URI{}, false
		}
		return URI{Scope: ScopeRoot}, true
	}
	parent := URI{
		Scope:    u.Scope,
		Segments: append([]string(nil), u.Segments[:len(u.Segments)-1]...),
	}
	if len(parent.Segments) == 0 {
		parent.Container = false
	}
	return parent, true
}

func (u URI) IsContainer() bool {
	return u.Container
}

func (u URI) IsRoot() bool {
	return u.Scope == ScopeRoot && len(u.Segments) == 0
}

func BuildSession(sessionKey string) string {
	return Scheme + "://" + ScopeSessions + "/" + strings.ToLower(sessionKey)
}

func BuildTurn(turnID string) string {
	return Scheme + "://" + ScopeTurns + "/" + strings.ToLower(turnID)
}

func BuildAtom(atomID string) string {
	return Scheme + "://" + ScopeAtoms + "/" + strings.ToLower(atomID)
}

func BuildScene(sceneID string) string {
	return Scheme + "://" + ScopeScenes + "/" + strings.ToLower(sceneID)
}

func BuildProfile() string {
	return Scheme + "://" + ScopeProfile
}

func BuildMemory(category, segment string) string {
	return Scheme + "://" + category + "/" + segment
}

func BuildCorrection(id string) string {
	return Scheme + "://" + ScopeCorrections + "/" + strings.ToLower(id)
}

// ParseAtomID extracts the atom key UUID from a full atom URI (rmb://atoms/<id>)
// or a bare UUID string.
func ParseAtomID(raw string) (uuid.UUID, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return uuid.Nil, fmt.Errorf("%w: empty atom reference", ErrInvalidURI)
	}
	if u, err := Parse(s); err == nil {
		if u.Scope != ScopeAtoms || len(u.Segments) != 1 {
			return uuid.Nil, fmt.Errorf("%w: not an atom uri", ErrInvalidURI)
		}
		return uuid.Parse(u.Segments[0])
	}
	if !uuidSegment.MatchString(s) {
		return uuid.Nil, fmt.Errorf("%w: invalid atom id %q", ErrInvalidURI, raw)
	}
	return uuid.Parse(s)
}

// SanitizeSlug normalizes a label into a strict URI slug: lowercase ASCII,
// hyphen-separated words, CJK preserved. Underscores, spaces, and dots become hyphens.
func SanitizeSlug(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("%w: empty segment", ErrInvalidURI)
	}

	var b strings.Builder
	prevSep := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevSep = false
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a')
			prevSep = false
		case r == '-':
			if b.Len() > 0 && !prevSep {
				b.WriteByte('-')
				prevSep = true
			}
		case isSlugPreservedRune(r):
			b.WriteRune(r)
			prevSep = false
		default:
			if b.Len() > 0 && !prevSep {
				b.WriteByte('-')
				prevSep = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "", fmt.Errorf("%w: segment sanitizes to empty", ErrInvalidURI)
	}
	if len(out) > MaxSegment {
		out = out[:MaxSegment]
		out = strings.TrimRight(out, "-")
	}
	if _, forbidden := reservedSlug[strings.ToLower(out)]; forbidden {
		return "", fmt.Errorf("%w: segment %q is reserved", ErrInvalidURI, out)
	}
	return out, nil
}

// SanitizeSegment is an alias for SanitizeSlug (memory URI path segments).
func SanitizeSegment(raw string) (string, error) {
	return SanitizeSlug(raw)
}

func splitSegments(path string) []string {
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func validateScope(scope string) error {
	switch scope {
	case ScopeSessions, ScopeTurns, ScopeAtoms, ScopeScenes, ScopeProfile, ScopePrefs, ScopeEntities, ScopeEvents, ScopeCorrections:
		return nil
	default:
		return fmt.Errorf("%w: unknown scope %q", ErrInvalidURI, scope)
	}
}

func validateSegment(seg string) error {
	if seg == "" {
		return fmt.Errorf("%w: empty path segment", ErrInvalidURI)
	}
	if len(seg) > MaxSegment {
		return fmt.Errorf("%w: segment too long", ErrInvalidURI)
	}
	return nil
}

func validateShape(scope string, segments []string) error {
	switch scope {
	case ScopeProfile:
		if len(segments) != 0 {
			return fmt.Errorf("%w: profile is a singleton", ErrInvalidURI)
		}
	case ScopeSessions:
		if len(segments) == 0 {
			return nil
		}
		if len(segments) == 1 {
			if !uuidSegment.MatchString(segments[0]) {
				return fmt.Errorf("%w: session id must be uuid", ErrInvalidURI)
			}
			return nil
		}
		return fmt.Errorf("%w: invalid sessions path", ErrInvalidURI)
	case ScopeTurns:
		if len(segments) == 0 {
			return nil
		}
		if len(segments) != 1 {
			return fmt.Errorf("%w: turns require one id segment", ErrInvalidURI)
		}
		if !uuidSegment.MatchString(segments[0]) {
			return fmt.Errorf("%w: turn id must be uuid", ErrInvalidURI)
		}
	case ScopeAtoms:
		if len(segments) == 0 {
			return nil
		}
		if len(segments) != 1 {
			return fmt.Errorf("%w: atoms require one id segment", ErrInvalidURI)
		}
		if !uuidSegment.MatchString(segments[0]) {
			return fmt.Errorf("%w: atom id must be uuid", ErrInvalidURI)
		}
	case ScopeScenes:
		// Zero segments is the category container (e.g. rmb://scenes/),
		// which `tree` lists; one segment is a single scene.
		if len(segments) == 0 {
			return nil
		}
		if len(segments) != 1 {
			return fmt.Errorf("%w: scenes require one id segment", ErrInvalidURI)
		}
	case ScopeCorrections:
		if len(segments) != 1 {
			return fmt.Errorf("%w: corrections require one id segment", ErrInvalidURI)
		}
		if !uuidSegment.MatchString(segments[0]) {
			return fmt.Errorf("%w: correction id must be uuid", ErrInvalidURI)
		}
	case ScopePrefs, ScopeEntities, ScopeEvents:
		// Zero segments is the category container (e.g. rmb://entities/),
		// which `tree` lists; one segment is a single memory.
		if len(segments) == 0 {
			return nil
		}
		if len(segments) != 1 {
			return fmt.Errorf("%w: %s requires one segment", ErrInvalidURI, scope)
		}
	}
	return nil
}

func isSlugPreservedRune(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Cyrillic, r)
}
