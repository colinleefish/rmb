package scene

import (
	"strings"
	"testing"

	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestGroupAtomsBySceneName(t *testing.T) {
	atoms := []model.Atom{
		{URI: "a1", SceneName: strPtr("Hook Config"), Content: "one"},
		{URI: "a2", SceneName: nil, Content: "two"},
		{URI: "a3", SceneName: strPtr("Hook Config"), Content: "three"},
	}
	groups := groupAtomsBySceneName(atoms)
	if len(groups) != 2 {
		t.Fatalf("got %d groups want 2", len(groups))
	}
	if groups[0].DisplayName != "General" || len(groups[0].Atoms) != 1 {
		t.Fatalf("unexpected general group: %+v", groups[0])
	}
	if groups[1].DisplayName != "Hook Config" || len(groups[1].Atoms) != 2 {
		t.Fatalf("unexpected hook group: %+v", groups[1])
	}
}

func TestSceneIDForName_stable(t *testing.T) {
	sid := uuid.MustParse("019e53d8-e94d-770a-9e81-601d892f9502")
	a := sceneIDForName(sid, "Hook Config", 1)
	b := sceneIDForName(sid, "hook config", 1) // case/space-insensitive
	if a != b {
		t.Fatalf("expected stable id across case/space: %v vs %v", a, b)
	}
	if a == sceneIDForName(sid, "Hook Config", 2) {
		t.Fatal("duplicate index should yield a different id")
	}
	other := uuid.MustParse("019e5441-fe41-7cdf-88cd-feb35930a739")
	if a == sceneIDForName(other, "Hook Config", 1) {
		t.Fatal("different sessions must yield different ids")
	}
	if sceneURIForName(sid, "Hook Config", 1) != uri.BuildScene(a.String()) {
		t.Fatal("sceneURIForName should wrap sceneIDForName")
	}
}

func TestChunkGroups(t *testing.T) {
	mk := func(name string, n int) atomGroup {
		atoms := make([]model.Atom, n)
		return atomGroup{DisplayName: name, Atoms: atoms}
	}
	groups := []atomGroup{mk("a", 40), mk("b", 30), mk("c", 50), mk("d", 10)}

	chunks := chunkGroups(groups, 60)
	// a(40) -> chunk1; b(30) would push to 70 -> chunk2 starts with b(30),
	// c(50) pushes to 80 -> chunk3 with c, d(10) fits -> c+d.
	if len(chunks) != 3 {
		t.Fatalf("got %d chunks want 3: %+v", len(chunks), chunks)
	}
	for i, c := range chunks {
		total := 0
		for _, g := range c {
			total += len(g.Atoms)
		}
		if total > 60 && len(c) > 1 {
			t.Fatalf("chunk %d exceeds max with multiple groups: %d", i, total)
		}
	}
}

func TestChunkGroups_oversizeGroupAlone(t *testing.T) {
	big := atomGroup{DisplayName: "big", Atoms: make([]model.Atom, 200)}
	chunks := chunkGroups([]atomGroup{big}, 60)
	if len(chunks) != 1 || len(chunks[0]) != 1 {
		t.Fatalf("oversize group should be emitted alone, got %+v", chunks)
	}
}

func TestChunkGroups_disabled(t *testing.T) {
	groups := []atomGroup{{DisplayName: "a", Atoms: make([]model.Atom, 5)}}
	if got := chunkGroups(groups, 0); len(got) != 1 {
		t.Fatalf("maxAtoms=0 should yield single chunk, got %d", len(got))
	}
}

func TestSerializeAtomsForLLM(t *testing.T) {
	groups := groupAtomsBySceneName([]model.Atom{
		{ID: uuid.MustParse("019eb28e-e351-7812-91a6-328b1f77a4bd"), Category: "entities", Priority: 50, Content: "fact"},
	})
	raw, err := serializeAtomsForLLM(groups)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "rmb://atoms/019eb28e-e351-7812-91a6-328b1f77a4bd") {
		t.Fatalf("missing uri in json: %s", raw)
	}
}
