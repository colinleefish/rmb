package uri

import "testing"

func TestParseAndString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"rmb://", "rmb://"},
		{"rmb://turns/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d",
			"rmb://turns/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d"},
		{"rmb://profile", "rmb://profile"},
		{"rmb://agent", "rmb://agent"},
		{"rmb://skills/", "rmb://skills/"},
		{"rmb://skills/pdf-processing", "rmb://skills/pdf-processing"},
		{"rmb://skills/pdf-processing/scripts/extract.py",
			"rmb://skills/pdf-processing/scripts/extract.py"},
		{"rmb://preferences/coffee", "rmb://preferences/coffee"},
	}

	for _, tc := range tests {
		u, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got := u.String(); got != tc.want {
			t.Fatalf("String() = %q, want %q", got, tc.want)
		}
	}
}

func TestBuildTurn(t *testing.T) {
	got := BuildTurn("4F1916CE-2F6E-4B76-8249-4A5F4184FD8D")
	want := "rmb://turns/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildAtom(t *testing.T) {
	got := BuildAtom("019EB28E-E351-7812-91A6-328B1F77A4BD")
	want := "rmb://atoms/019eb28e-e351-7812-91a6-328b1f77a4bd"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseAtomID(t *testing.T) {
	const id = "019eb28e-e351-7812-91a6-328b1f77a4bd"
	fromURI, err := ParseAtomID(BuildAtom(id))
	if err != nil {
		t.Fatal(err)
	}
	fromBare, err := ParseAtomID(id)
	if err != nil {
		t.Fatal(err)
	}
	if fromURI != fromBare {
		t.Fatalf("uri vs bare mismatch: %v vs %v", fromURI, fromBare)
	}
	if _, err := ParseAtomID("rmb://scenes/x"); err == nil {
		t.Fatal("expected error for non-atom uri")
	}
}

func TestParseSceneID(t *testing.T) {
	const id = "b6f6e2c2-7c1a-4e2b-9c3d-7a1f0d2e4b88"
	fromURI, err := ParseSceneID(BuildScene(id))
	if err != nil {
		t.Fatal(err)
	}
	fromBare, err := ParseSceneID(id)
	if err != nil {
		t.Fatal(err)
	}
	if fromURI != fromBare {
		t.Fatalf("uri vs bare mismatch: %v vs %v", fromURI, fromBare)
	}
	if _, err := ParseSceneID("rmb://atoms/x"); err == nil {
		t.Fatal("expected error for non-scene uri")
	}
}

func TestParseRejectsNestedTurnURI(t *testing.T) {
	_, err := Parse("rmb://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0")
	if err == nil {
		t.Fatal("expected nested turn path to be rejected")
	}
}

func TestParseRejectsNestedAtomURI(t *testing.T) {
	_, err := Parse("rmb://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/atoms/019eb28e-e351-7812-91a6-328b1f77a4bd")
	if err == nil {
		t.Fatal("expected nested atom path to be rejected")
	}
}

func TestSanitizeSlugCJK(t *testing.T) {
	got, err := SanitizeSlug("李广慧")
	if err != nil {
		t.Fatalf("SanitizeSlug: %v", err)
	}
	if got != "李广慧" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeSlugKebab(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"go_style", "go-style"},
		{"Go Style", "go-style"},
		{"ai-tone", "ai-tone"},
		{"UPPER_SNAKE", "upper-snake"},
		{"2026-05-17-postgres-only", "2026-05-17-postgres-only"},
		{"colin_mom", "colin-mom"},
	}
	for _, tc := range tests {
		got, err := SanitizeSlug(tc.in)
		if err != nil {
			t.Fatalf("SanitizeSlug(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("SanitizeSlug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseRejectsReservedSlug(t *testing.T) {
	if _, err := SanitizeSlug("profile"); err == nil {
		t.Fatalf("expected reserved slug error")
	}
}

func TestContainerTrailingSlash(t *testing.T) {
	u, err := Parse("rmb://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !u.IsContainer() {
		t.Fatalf("expected container uri")
	}
}

func TestParseScopeContainers(t *testing.T) {
	cases := []string{
		"rmb://entities/",
		"rmb://preferences/",
		"rmb://events/",
		"rmb://turns/",
		"rmb://atoms/",
		"rmb://scenes/",
	}
	for _, in := range cases {
		u, err := Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if !u.IsContainer() {
			t.Fatalf("Parse(%q): expected container uri", in)
		}
		if len(u.Segments) != 0 {
			t.Fatalf("Parse(%q): expected zero segments, got %v", in, u.Segments)
		}
	}
}
