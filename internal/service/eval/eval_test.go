package eval

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/colinleefish/mypast/internal/service/recall"
)

func TestLoadProbes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.txt")
	content := "# comment\n\nWhere do I live?\tmypast://profile\nNo prefix query\n   \nWhat tone?\tmypast://preferences/\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	probes, err := LoadProbes(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(probes) != 3 {
		t.Fatalf("got %d probes want 3: %+v", len(probes), probes)
	}
	if probes[0].Query != "Where do I live?" || probes[0].ExpectedPrefix != "mypast://profile" {
		t.Fatalf("probe0 wrong: %+v", probes[0])
	}
	if probes[1].ExpectedPrefix != "" {
		t.Fatalf("probe1 should have no prefix: %+v", probes[1])
	}
}

func TestLoadProbes_empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.txt")
	if err := os.WriteFile(path, []byte("# only comments\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProbes(path); err == nil {
		t.Fatal("expected error for no probes")
	}
}

func TestHitByPrefix(t *testing.T) {
	matches := []recall.Match{
		{URI: "mypast://entities/tesla"},
		{URI: "mypast://preferences/ai-tone"},
	}
	if !hitByPrefix(matches, "mypast://preferences/") {
		t.Fatal("expected prefix hit")
	}
	if hitByPrefix(matches, "mypast://profile") {
		t.Fatal("unexpected hit")
	}
	if !hitByPrefix(matches, "") {
		t.Fatal("empty prefix should hit when matches exist")
	}
	if hitByPrefix(nil, "") {
		t.Fatal("empty prefix should miss when no matches")
	}
}

func TestClassify(t *testing.T) {
	probe := Probe{Query: "q", ExpectedPrefix: "mypast://profile"}

	// Hit: full surfaces expected prefix.
	hit := classify(probe, []recall.Match{{URI: "mypast://profile"}}, []recall.Match{{URI: "t"}})
	if !hit.FullHit || hit.Regression {
		t.Fatalf("expected clean hit: %+v", hit)
	}

	// Regression: baseline has evidence, full misses.
	regr := classify(probe, []recall.Match{{URI: "mypast://entities/x"}}, []recall.Match{{URI: "t"}})
	if regr.FullHit || !regr.Regression {
		t.Fatalf("expected regression: %+v", regr)
	}

	// Not in data: neither has it -> not a regression.
	miss := classify(probe, nil, nil)
	if miss.FullHit || miss.Regression {
		t.Fatalf("expected plain miss, not regression: %+v", miss)
	}
}
