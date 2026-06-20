package pgarray

import "testing"

func TestTextArrayValue_quotedURI(t *testing.T) {
	v, err := TextArray{"rmb://sessions/x/atoms/y"}.Value()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}
	want := `{"rmb://sessions/x/atoms/y"}`
	if got != want {
		t.Fatalf("Value() = %q, want %q", got, want)
	}
}

func TestTextArrayScanRoundTrip(t *testing.T) {
	src := TextArray{
		"rmb://sessions/a/atoms/1",
		"rmb://sessions/b/atoms/2",
	}
	val, err := src.Value()
	if err != nil {
		t.Fatal(err)
	}
	var dst TextArray
	if err := dst.Scan(val.(string)); err != nil {
		t.Fatal(err)
	}
	if len(dst) != 2 || dst[0] != src[0] || dst[1] != src[1] {
		t.Fatalf("round-trip mismatch: %+v", []string(dst))
	}
}
