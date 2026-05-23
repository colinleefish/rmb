package pgarray

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUIDArrayValue(t *testing.T) {
	a := uuid.MustParse("019e53d8-e94e-7496-8b76-c5f2d3258aa4")
	v, err := UUIDArray{a}.Value()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := v.(string)
	if !ok {
		t.Fatalf("expected string driver value, got %T", v)
	}
	want := "{019e53d8-e94e-7496-8b76-c5f2d3258aa4}"
	if got != want {
		t.Fatalf("Value() = %q, want %q", got, want)
	}
}

func TestUUIDArrayScanRoundTrip(t *testing.T) {
	a := uuid.MustParse("019e53d8-e94e-7496-8b76-c5f2d3258aa4")
	b := uuid.MustParse("019e5441-fe41-7cdf-88cd-feb35930a739")

	var dst UUIDArray
	if err := dst.Scan("{019e53d8-e94e-7496-8b76-c5f2d3258aa4,019e5441-fe41-7cdf-88cd-feb35930a739}"); err != nil {
		t.Fatal(err)
	}
	if len(dst) != 2 || dst[0] != a || dst[1] != b {
		t.Fatalf("scan mismatch: %+v", []uuid.UUID(dst))
	}

	val, err := dst.Value()
	if err != nil {
		t.Fatal(err)
	}
	if val.(string) != "{019e53d8-e94e-7496-8b76-c5f2d3258aa4,019e5441-fe41-7cdf-88cd-feb35930a739}" {
		t.Fatalf("round-trip value %v", val)
	}
}
