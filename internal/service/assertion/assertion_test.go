package assertion

import (
	"context"
	"errors"
	"testing"

	"github.com/colinleefish/mypast/internal/model"
)

func TestNormalizeTargets_dedupAndCanonicalize(t *testing.T) {
	got, err := normalizeTargets([]string{
		"mypast://entities/jenkins",
		" mypast://entities/jenkins ", // whitespace + duplicate
		"mypast://profile",
		"",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"mypast://entities/jenkins", "mypast://profile"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestNormalizeTargets_rejectsInvalidURI(t *testing.T) {
	if _, err := normalizeTargets([]string{"not-a-uri"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// The following validation branches return before any DB access, so a nil-db
// Service exercises them safely.
func TestCreate_rejectsUnknownKind(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{Kind: "bogus", TargetURIs: []string{"mypast://profile"}})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_requiresTarget(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{Kind: model.AssertionKindCorrect, Statement: "x"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_correctRequiresStatement(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{
		Kind:       model.AssertionKindCorrect,
		TargetURIs: []string{"mypast://entities/jenkins"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
