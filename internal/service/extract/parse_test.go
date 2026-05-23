package extract

import "testing"

func TestParseExtractResponse(t *testing.T) {
	raw := `{"atoms":[{"category":"preferences","priority":70,"scene_name":"setup","slug":"go-style","content":"Prefers Go monoliths.","source_turn_indices":[0]}]}`
	atoms, err := parseExtractResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(atoms) != 1 || atoms[0].Category != "preferences" {
		t.Fatalf("unexpected atoms: %#v", atoms)
	}
}

func TestParseExtractResponse_skipsEmptyContent(t *testing.T) {
	raw := `{"atoms":[{"category":"events","priority":50,"content":"  ","source_turn_indices":[0]}]}`
	atoms, err := parseExtractResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(atoms) != 0 {
		t.Fatalf("expected empty atoms, got %#v", atoms)
	}
}
