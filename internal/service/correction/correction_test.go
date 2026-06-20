package correction

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeTargets_dedupAndCanonicalize(t *testing.T) {
	got, err := normalizeTargets([]string{
		"mem9://entities/jenkins",
		" mem9://entities/jenkins ", // whitespace + duplicate
		"mem9://profile",
		"",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"mem9://entities/jenkins", "mem9://profile"}
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
func TestCreate_requiresTarget(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{Statement: "x"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_requiresStatement(t *testing.T) {
	s := &Service{}
	_, err := s.Create(context.Background(), CreateInput{
		TargetURIs: []string{"mem9://entities/jenkins"},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
