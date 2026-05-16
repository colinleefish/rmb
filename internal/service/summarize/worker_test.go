package summarize

import (
	"testing"
	"time"

	"github.com/colinleefish/mypast/internal/model"
	"github.com/google/uuid"
)

func TestCanClaimTurn(t *testing.T) {
	now := time.Date(2026, time.May, 10, 8, 0, 0, 0, time.UTC)
	staleBefore := now.Add(-2 * time.Minute)
	recent := now.Add(-30 * time.Second)
	stale := now.Add(-10 * time.Minute)

	tests := []struct {
		name string
		turn model.SessionTurn
		want bool
	}{
		{
			name: "not_summarized_is_claimable",
			turn: model.SessionTurn{
				TurnStatus: model.SessionTurnStatusNotSummarized,
			},
			want: true,
		},
		{
			name: "summarizing_with_recent_start_not_claimable",
			turn: model.SessionTurn{
				TurnStatus:         model.SessionTurnStatusSummarizing,
				SummarizeStartedAt: &recent,
			},
			want: false,
		},
		{
			name: "summarizing_with_stale_start_is_claimable",
			turn: model.SessionTurn{
				TurnStatus:         model.SessionTurnStatusSummarizing,
				SummarizeStartedAt: &stale,
			},
			want: true,
		},
		{
			name: "failed_not_claimable",
			turn: model.SessionTurn{
				TurnStatus: model.SessionTurnStatusFailed,
			},
			want: false,
		},
		{
			name: "summarized_not_claimable",
			turn: model.SessionTurn{
				TurnStatus: model.SessionTurnStatusSummarized,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canClaimTurn(tt.turn, staleBefore)
			if got != tt.want {
				t.Fatalf("canClaimTurn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeTurnMessagesJSONL(t *testing.T) {
	turns := []model.SessionTurn{
		{MessagesJSONL: "{\"id\":1}\n"},
		{MessagesJSONL: "{\"id\":2}\n"},
	}

	out := mergeTurnMessagesJSONL(turns, 0)
	if out != "{\"id\":1}\n{\"id\":2}\n" {
		t.Fatalf("unexpected merged jsonl: %q", out)
	}

	limited := mergeTurnMessagesJSONL(turns, len("{\"id\":1}\n"))
	if limited != "{\"id\":1}\n" {
		t.Fatalf("unexpected limited jsonl: %q", limited)
	}
}

func TestShouldLogFailedHeadTurnRateLimited(t *testing.T) {
	w := &Worker{
		failedHeadLogAt: make(map[uuid.UUID]time.Time),
	}
	sessionID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	base := time.Date(2026, time.May, 14, 12, 0, 0, 0, time.UTC)

	if !w.shouldLogFailedHeadTurn(sessionID, base) {
		t.Fatalf("first failed-head log should be emitted")
	}
	if w.shouldLogFailedHeadTurn(sessionID, base.Add(failedHeadLogInterval-time.Second)) {
		t.Fatalf("failed-head log should be rate-limited")
	}
	if !w.shouldLogFailedHeadTurn(sessionID, base.Add(failedHeadLogInterval)) {
		t.Fatalf("failed-head log should emit after interval")
	}
}
