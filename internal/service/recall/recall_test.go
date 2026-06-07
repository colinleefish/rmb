package recall

import "testing"

func TestFuseRRF_combinesAndRanks(t *testing.T) {
	vec := []Match{
		{URI: "a", Tier: "memories", Snippet: "a"},
		{URI: "b", Tier: "memories", Snippet: "b"},
	}
	fts := []Match{
		{URI: "b", Tier: "memories", Snippet: "b"},
		{URI: "c", Tier: "memories", Snippet: "c"},
	}
	fused := FuseRRF([][]Match{vec, fts}, 60, 10)

	if len(fused) != 3 {
		t.Fatalf("got %d fused want 3", len(fused))
	}
	// b appears in both lists, so it should rank first.
	if fused[0].URI != "b" {
		t.Fatalf("expected b first, got %q", fused[0].URI)
	}
	// Scores must be descending.
	for i := 1; i < len(fused); i++ {
		if fused[i-1].Rank < fused[i].Rank {
			t.Fatalf("not descending at %d: %+v", i, fused)
		}
	}
}

func TestFuseRRF_topK(t *testing.T) {
	lists := [][]Match{{
		{URI: "a"}, {URI: "b"}, {URI: "c"}, {URI: "d"},
	}}
	fused := FuseRRF(lists, 60, 2)
	if len(fused) != 2 {
		t.Fatalf("topK not applied: got %d", len(fused))
	}
	if fused[0].URI != "a" || fused[1].URI != "b" {
		t.Fatalf("unexpected order: %+v", fused)
	}
}

func TestFuseRRF_empty(t *testing.T) {
	if got := FuseRRF(nil, 60, 5); len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestFuseRRF_positionMatters(t *testing.T) {
	// Same URI ranked higher in one list should beat a unique lower-ranked URI.
	listA := []Match{{URI: "top"}, {URI: "mid"}}
	listB := []Match{{URI: "top"}, {URI: "other"}}
	fused := FuseRRF([][]Match{listA, listB}, 60, 10)
	if fused[0].URI != "top" {
		t.Fatalf("expected 'top' first, got %q", fused[0].URI)
	}
}
