package scene

import (
	"testing"

	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

const (
	testAtomA1 = "00000000-0000-4000-8000-000000000001"
	testAtomA2 = "00000000-0000-4000-8000-000000000002"
)

func TestParseBuildScenesResponse(t *testing.T) {
	valid := map[string]struct{}{
		uri.BuildAtom(testAtomA1): {},
		uri.BuildAtom(testAtomA2): {},
	}
	raw := `{"scenes":[{"display_name":"Setup","abstract":"Hook setup.","body":"## Setup\nConfigured hooks.","atom_uris":["` + uri.BuildAtom(testAtomA1) + `","` + uri.BuildAtom(testAtomA2) + `"]}]}`
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 {
		t.Fatalf("got %d scenes", len(scenes))
	}
	wantA1 := uuid.MustParse(testAtomA1)
	wantA2 := uuid.MustParse(testAtomA2)
	if scenes[0].DisplayName != "Setup" || len(scenes[0].SourceAtoms) != 2 {
		t.Fatalf("unexpected scene: %+v", scenes[0])
	}
	if scenes[0].SourceAtoms[0] != wantA1 || scenes[0].SourceAtoms[1] != wantA2 {
		t.Fatalf("unexpected source atoms: %+v", scenes[0].SourceAtoms)
	}
}

func TestParseBuildScenesResponse_dropsUnknownURIsKeepsValid(t *testing.T) {
	valid := map[string]struct{}{
		uri.BuildAtom(testAtomA1): {},
		uri.BuildAtom(testAtomA2): {},
	}
	raw := `{"scenes":[
		{"display_name":"Keep","abstract":"a","body":"b","atom_uris":["` + uri.BuildAtom(testAtomA1) + `","rmb://hallucinated"]},
		{"display_name":"Drop","abstract":"a","body":"b","atom_uris":["rmb://hallucinated"]}
	]}`
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("expected 1 usable scene (unknown-only scene dropped), got %d", len(scenes))
	}
	wantA1 := uuid.MustParse(testAtomA1)
	if scenes[0].DisplayName != "Keep" || len(scenes[0].SourceAtoms) != 1 ||
		scenes[0].SourceAtoms[0] != wantA1 {
		t.Fatalf("unexpected scene after dropping unknown uri: %+v", scenes[0])
	}
}

func TestParseBuildScenesResponse_allUnknownErrors(t *testing.T) {
	valid := map[string]struct{}{uri.BuildAtom(testAtomA1): {}}
	raw := `{"scenes":[{"display_name":"X","abstract":"a","body":"b","atom_uris":["rmb://bad"]}]}`
	if _, err := parseBuildScenesResponse(raw, valid); err == nil {
		t.Fatal("expected error when no scene has any valid uri")
	}
}

func TestParseBuildScenesResponse_fence(t *testing.T) {
	valid := map[string]struct{}{uri.BuildAtom(testAtomA1): {}}
	raw := "```json\n{\"scenes\":[{\"display_name\":\"X\",\"abstract\":\"a\",\"body\":\"b\",\"atom_uris\":[\"" + uri.BuildAtom(testAtomA1) + "\"]}]}\n```"
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 {
		t.Fatalf("got %d scenes", len(scenes))
	}
}
