package scene

import "testing"

func TestParseBuildScenesResponse(t *testing.T) {
	valid := map[string]struct{}{
		"rmb://sessions/x/atoms/a1": {},
		"rmb://sessions/x/atoms/a2": {},
	}
	raw := `{"scenes":[{"display_name":"Setup","abstract":"Hook setup.","body":"## Setup\nConfigured hooks.","atom_uris":["rmb://sessions/x/atoms/a1","rmb://sessions/x/atoms/a2"]}]}`
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 {
		t.Fatalf("got %d scenes", len(scenes))
	}
	if scenes[0].DisplayName != "Setup" || len(scenes[0].SourceAtomURIs) != 2 {
		t.Fatalf("unexpected scene: %+v", scenes[0])
	}
}

func TestParseBuildScenesResponse_dropsUnknownURIsKeepsValid(t *testing.T) {
	valid := map[string]struct{}{
		"rmb://sessions/x/atoms/a1": {},
		"rmb://sessions/x/atoms/a2": {},
	}
	// One scene mixes a valid and an unknown URI; another has only an unknown URI.
	raw := `{"scenes":[
		{"display_name":"Keep","abstract":"a","body":"b","atom_uris":["rmb://sessions/x/atoms/a1","rmb://hallucinated"]},
		{"display_name":"Drop","abstract":"a","body":"b","atom_uris":["rmb://hallucinated"]}
	]}`
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenes) != 1 {
		t.Fatalf("expected 1 usable scene (unknown-only scene dropped), got %d", len(scenes))
	}
	if scenes[0].DisplayName != "Keep" || len(scenes[0].SourceAtomURIs) != 1 ||
		scenes[0].SourceAtomURIs[0] != "rmb://sessions/x/atoms/a1" {
		t.Fatalf("unexpected scene after dropping unknown uri: %+v", scenes[0])
	}
}

func TestParseBuildScenesResponse_allUnknownErrors(t *testing.T) {
	valid := map[string]struct{}{"rmb://sessions/x/atoms/a1": {}}
	raw := `{"scenes":[{"display_name":"X","abstract":"a","body":"b","atom_uris":["rmb://bad"]}]}`
	if _, err := parseBuildScenesResponse(raw, valid); err == nil {
		t.Fatal("expected error when no scene has any valid uri")
	}
}

func TestParseBuildScenesResponse_fence(t *testing.T) {
	valid := map[string]struct{}{"rmb://sessions/x/atoms/a1": {}}
	raw := "```json\n{\"scenes\":[{\"display_name\":\"X\",\"abstract\":\"a\",\"body\":\"b\",\"atom_uris\":[\"rmb://sessions/x/atoms/a1\"]}]}\n```"
	scenes, err := parseBuildScenesResponse(raw, valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(scenes) != 1 {
		t.Fatalf("got %d scenes", len(scenes))
	}
}
