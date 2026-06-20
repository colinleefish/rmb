package alias

import (
	"testing"

	"github.com/colinleefish/mypast/internal/llm"
	"github.com/colinleefish/mypast/internal/model"
)

func set(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

func TestSelectPairsToJudge_dedupsUnorderedPairs(t *testing.T) {
	raw := []pair{
		{AURI: "mypast://entities/a", BURI: "mypast://entities/b"},
		// reverse direction of the same unordered pair must collapse
		{AURI: "mypast://entities/b", BURI: "mypast://entities/a"},
		{AURI: "mypast://entities/a", BURI: "mypast://entities/c"},
	}
	got := selectPairsToJudge(raw, nil, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique pairs, got %d: %+v", len(got), got)
	}
}

func TestSelectPairsToJudge_skipsAlreadyJudged(t *testing.T) {
	raw := []pair{
		{AURI: "mypast://entities/a", BURI: "mypast://entities/b"},
		{AURI: "mypast://entities/a", BURI: "mypast://entities/c"},
	}
	// judged set is keyed by the same unordered key, in reverse direction.
	judged := set(pairKey("mypast://entities/b", "mypast://entities/a"))
	got := selectPairsToJudge(raw, judged, nil)
	if len(got) != 1 || got[0].BURI != "mypast://entities/c" {
		t.Fatalf("expected only the a~c pair, got %+v", got)
	}
}

func TestSelectPairsToJudge_skipsAlreadyAliased(t *testing.T) {
	raw := []pair{
		{AURI: "mypast://entities/a", BURI: "mypast://entities/b"},
		{AURI: "mypast://entities/c", BURI: "mypast://entities/d"},
	}
	// b is already in an active alias (either side) → drop any pair touching it.
	aliased := set("mypast://entities/b")
	got := selectPairsToJudge(raw, nil, aliased)
	if len(got) != 1 || got[0].AURI != "mypast://entities/c" {
		t.Fatalf("expected only the c~d pair, got %+v", got)
	}
}

func TestSelectPairsToJudge_skipsSelfAndEmpty(t *testing.T) {
	raw := []pair{
		{AURI: "mypast://entities/a", BURI: "mypast://entities/a"},
		{AURI: "", BURI: "mypast://entities/b"},
		{AURI: "mypast://entities/c", BURI: ""},
	}
	if got := selectPairsToJudge(raw, nil, nil); len(got) != 0 {
		t.Fatalf("expected 0 pairs, got %+v", got)
	}
}

func TestCandidateFromVerdict_samePicksDirection(t *testing.T) {
	p := pair{AURI: "mypast://entities/aliyun-rds-instance", BURI: "mypast://entities/aliyun-rds", Sim: 0.91}
	v := llm.AliasVerdict{Same: true, CanonicalURI: p.BURI, Rationale: "same RDS instance"}
	row := candidateFromVerdict(p, v, p.Sim)

	if row.Status != model.AliasCandidateStatusPending {
		t.Fatalf("expected pending, got %q", row.Status)
	}
	if row.CanonicalURI != p.BURI || row.AliasURI != p.AURI {
		t.Fatalf("direction wrong: alias=%q canonical=%q", row.AliasURI, row.CanonicalURI)
	}
	if row.Similarity == nil || *row.Similarity != 0.91 {
		t.Fatalf("similarity not carried: %+v", row.Similarity)
	}
	if row.Rationale == nil || *row.Rationale != "same RDS instance" {
		t.Fatalf("rationale not carried: %+v", row.Rationale)
	}
}

func TestCandidateFromVerdict_differentIsRejectedDeterministic(t *testing.T) {
	p := pair{AURI: "mypast://entities/aliyun-rds-prod", BURI: "mypast://entities/aliyun-rds-dev", Sim: 0.95}
	v := llm.AliasVerdict{Same: false, Rationale: "prod vs dev are distinct"}
	row := candidateFromVerdict(p, v, p.Sim)

	if row.Status != model.AliasCandidateStatusRejected {
		t.Fatalf("expected rejected, got %q", row.Status)
	}
	// Rejected pairs store a sorted (deterministic) direction.
	a, b := sortedPair(p.AURI, p.BURI)
	if row.AliasURI != a || row.CanonicalURI != b {
		t.Fatalf("expected sorted direction (%q,%q), got (%q,%q)", a, b, row.AliasURI, row.CanonicalURI)
	}
}

func TestCandidateFromVerdict_invalidCanonicalFallsBackToRejected(t *testing.T) {
	// A "same" verdict whose canonical is neither supplied URI is untrustworthy;
	// record it as rejected rather than fabricate a direction.
	p := pair{AURI: "mypast://entities/a", BURI: "mypast://entities/b"}
	v := llm.AliasVerdict{Same: true, CanonicalURI: "mypast://entities/somewhere-else"}
	row := candidateFromVerdict(p, v, 0.9)
	if row.Status != model.AliasCandidateStatusRejected {
		t.Fatalf("expected rejected for invalid canonical, got %q", row.Status)
	}
}

func TestPairKey_orderIndependent(t *testing.T) {
	if pairKey("a", "b") != pairKey("b", "a") {
		t.Fatal("pairKey must be order-independent")
	}
}
