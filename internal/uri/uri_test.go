package uri

import "testing"

func TestParseAndString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"mypast://", "mypast://"},
		{"/sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0",
			"mypast://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0"},
		{"mypast://profile", "mypast://profile"},
		{"mypast://preferences/coffee", "mypast://preferences/coffee"},
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
	want := "mypast://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/3"
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
	u, err := Parse("mypast://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !u.IsContainer() {
		t.Fatalf("expected container uri")
	}
}
