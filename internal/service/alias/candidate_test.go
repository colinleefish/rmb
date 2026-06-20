package alias

import (
	"context"
	"errors"
	"testing"
)

// These exercise the input-validation branches that return before any DB access,
// matching the nil-db Service style used elsewhere in this package.

func TestConfirmCandidate_rejectsBadID(t *testing.T) {
	s := &Service{}
	_, err := s.ConfirmCandidate(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRejectCandidate_rejectsBadID(t *testing.T) {
	s := &Service{}
	err := s.RejectCandidate(context.Background(), "not-a-uuid")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
