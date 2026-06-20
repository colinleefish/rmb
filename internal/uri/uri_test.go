package uri

import "testing"

func TestParseAndString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"mem9://", "mem9://"},
		{"/sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0",
			"mem9://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0"},
		{"mem9://profile", "mem9://profile"},
		{"mem9://preferences/coffee", "mem9://preferences/coffee"},
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

func TestBuildSessionTurn(t *testing.T) {
	got := BuildSessionTurn("4F1916CE-2F6E-4B76-8249-4A5F4184FD8D", 3)
	want := "mem9://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/3"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
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
	u, err := Parse("mem9://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !u.IsContainer() {
		t.Fatalf("expected container uri")
	}
}

// Scope-level container URIs (no leaf segment) must parse so that `tree` can
// list everything under a category. treeRoot advertises these exact URIs.
func TestParseScopeContainers(t *testing.T) {
	cases := []string{
		"mem9://entities/",
		"mem9://preferences/",
		"mem9://events/",
		"mem9://scenes/",
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
