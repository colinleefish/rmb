package extract

import (
	"testing"
	"time"
)

func TestShouldRunT1_pending(t *testing.T) {
	now := time.Now()
	if !shouldRunT1(now, "pending", 1, 0, 2, 8, true, time.Minute, now) {
		t.Fatal("expected pending to run")
	}
}

func TestShouldRunT1_everyN(t *testing.T) {
	now := time.Now()
	if !shouldRunT1(now, "idle", 3, 8, 8, 8, true, time.Hour, now) {
		t.Fatal("expected everyN threshold to run")
	}
}

func TestShouldRunT1_warmup(t *testing.T) {
	now := time.Now()
	if !shouldRunT1(now, "idle", 2, 2, 2, 8, true, time.Hour, now) {
		t.Fatal("expected warmup threshold to run")
	}
	if shouldRunT1(now, "idle", 1, 1, 2, 8, true, time.Hour, now) {
		t.Fatal("expected below warmup threshold to skip")
	}
}

func TestShouldRunT1_idle(t *testing.T) {
	now := time.Now()
	last := now.Add(-11 * time.Minute)
	if !shouldRunT1(now, "idle", 1, 1, 8, 8, true, 10*time.Minute, last) {
		t.Fatal("expected idle trigger")
	}
}

func TestShouldRunT1_failedWithTurns(t *testing.T) {
	now := time.Now()
	if !shouldRunT1(now, "failed", 1, 0, 2, 8, true, time.Minute, now) {
		t.Fatal("expected failed pipeline with unprocessed turns to retry")
	}
	if shouldRunT1(now, "failed", 0, 0, 2, 8, true, time.Minute, now) {
		t.Fatal("expected failed with no unprocessed turns to skip")
	}
}

func TestShouldRunT1_noTurns(t *testing.T) {
	now := time.Now()
	if shouldRunT1(now, "pending", 0, 0, 2, 8, true, time.Minute, now) {
		t.Fatal("expected no unprocessed turns to skip")
	}
}

func TestNextWarmupThreshold(t *testing.T) {
	if got := nextWarmupThreshold(2, 8, true); got != 4 {
		t.Fatalf("got %d want 4", got)
	}
	if got := nextWarmupThreshold(8, 8, true); got != 8 {
		t.Fatalf("got %d want 8", got)
	}
}
