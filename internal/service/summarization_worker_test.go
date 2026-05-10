package service

import (
	"testing"
	"time"

	"github.com/colinleefish/mypast/internal/model"
)

func TestCanClaimRecord(t *testing.T) {
	now := time.Date(2026, time.May, 10, 8, 0, 0, 0, time.UTC)
	staleBefore := now.Add(-2 * time.Minute)
	recent := now.Add(-30 * time.Second)
	stale := now.Add(-10 * time.Minute)

	tests := []struct {
		name   string
		record model.SessionRecord
		want   bool
	}{
		{
			name: "not_summarized_is_claimable",
			record: model.SessionRecord{
				RecordStatus: model.SessionRecordStatusNotSummarized,
			},
			want: true,
		},
		{
			name: "summarizing_with_recent_start_not_claimable",
			record: model.SessionRecord{
				RecordStatus:       model.SessionRecordStatusSummarizing,
				SummarizeStartedAt: &recent,
			},
			want: false,
		},
		{
			name: "summarizing_with_stale_start_is_claimable",
			record: model.SessionRecord{
				RecordStatus:       model.SessionRecordStatusSummarizing,
				SummarizeStartedAt: &stale,
			},
			want: true,
		},
		{
			name: "failed_not_claimable",
			record: model.SessionRecord{
				RecordStatus: model.SessionRecordStatusFailed,
			},
			want: false,
		},
		{
			name: "summarized_not_claimable",
			record: model.SessionRecord{
				RecordStatus: model.SessionRecordStatusSummarized,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canClaimRecord(tt.record, staleBefore)
			if got != tt.want {
				t.Fatalf("canClaimRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}
