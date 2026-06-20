package alias

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeMemoryURI_acceptsMergeableTiers(t *testing.T) {
	cases := map[string]struct{ uri, category string }{
		"mem9://entities/aliyun-rds":  {"mem9://entities/aliyun-rds", "entities"},
		" mem9://preferences/editor ": {"mem9://preferences/editor", "preferences"},
	}
	for in, want := range cases {
		gotURI, gotCat, err := normalizeMemoryURI(in)
		if err != nil {
			t.Fatalf("normalizeMemoryURI(%q) error: %v", in, err)
		}
		if gotURI != want.uri || gotCat != want.category {
			t.Fatalf("normalizeMemoryURI(%q) = (%q,%q), want (%q,%q)", in, gotURI, gotCat, want.uri, want.category)
		}
	}
}

func TestNormalizeMemoryURI_rejectsUnmergeableTiers(t *testing.T) {
	// Only preferences and entities can be aliased. profile (singleton) and events
	// (immutable) cannot, nor can scenes/sessions/corrections/aliases.
	for _, in := range []string{
		"mem9://profile",
		"mem9://events/2026-06-12-foo",
		"mem9://scenes/00000000-0000-0000-0000-000000000000",
		"mem9://corrections/00000000-0000-0000-0000-000000000000",
		"not-a-uri",
	} {
		if _, _, err := normalizeMemoryURI(in); err == nil {
			t.Fatalf("normalizeMemoryURI(%q): expected error, got nil", in)
		}
	}
}

func TestCreate_rejectsCrossCategory(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{
		AliasURI:     "mem9://entities/aliyun-rds",
		CanonicalURI: "mem9://preferences/editor",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for cross-category, got %v", err)
	}
}

// The branches below return before any DB access, so a nil-db Service exercises
// them safely (matching the correction service test style).
func TestCreate_rejectsNonMemoryURI(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{
		AliasURI:     "mem9://scenes/00000000-0000-0000-0000-000000000000",
		CanonicalURI: "mem9://entities/aliyun-rds",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_rejectsSelfAlias(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{
		AliasURI:     "mem9://entities/aliyun-rds",
		CanonicalURI: "mem9://entities/aliyun-rds",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRetract_rejectsNonAliasURI(t *testing.T) {
	s := &Service{}
	_, err := s.Retract(context.Background(), "mem9://entities/aliyun-rds")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
