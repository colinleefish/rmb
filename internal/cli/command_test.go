package cli

import (
	"strings"
	"testing"

	"github.com/colinleefish/rmb/internal/client"
)

func TestPrintHelpSectionHeader(t *testing.T) {
	var buf strings.Builder
	printHelpSectionHeader(&buf, "profile", "Profile", "rmb://profile", "Who the user is.")
	got := buf.String()
	if !strings.Contains(got, helpSectionRule) {
		t.Fatalf("missing divider in %q", got)
	}
	if !strings.Contains(got, "[profile]\n") {
		t.Fatalf("missing marker in %q", got)
	}
	if !strings.Contains(got, "Profile  rmb://profile\n") {
		t.Fatalf("title line = %q", got)
	}
	if !strings.Contains(got, "Who the user is.\n") {
		t.Fatalf("blurb missing in %q", got)
	}
}

func TestPrintHelpUsageDivider(t *testing.T) {
	var buf strings.Builder
	printHelpUsageDivider(&buf)
	got := buf.String()
	if !strings.Contains(got, helpSectionRule) {
		t.Fatalf("missing divider in %q", got)
	}
	if !strings.Contains(got, "[usage]\n") {
		t.Fatalf("missing usage marker in %q", got)
	}
}

func TestPrintHelpSkillEntry(t *testing.T) {
	var buf strings.Builder
	printHelpSkillEntry(&buf, client.SkillSummary{
		URI:         "rmb://skills/demo",
		Description: "Does things.",
		Tags:        []string{"work", "demo"},
	})
	got := buf.String()
	if !strings.Contains(got, "rmb://skills/demo\n") {
		t.Fatalf("uri missing in %q", got)
	}
	if !strings.Contains(got, "  [desc] Does things.\n") {
		t.Fatalf("desc line = %q", got)
	}
	if !strings.Contains(got, "  [tags] work, demo\n") {
		t.Fatalf("tags line = %q", got)
	}
}

func TestFormatHelpSkillDescription(t *testing.T) {
	if got := formatHelpSkillDescription(""); got != "(no description)" {
		t.Fatalf("empty = %q", got)
	}
	if got := formatHelpSkillDescription("  hello\nworld  "); got != "hello world" {
		t.Fatalf("normalize = %q", got)
	}
	long := stringsRepeat("a", 130)
	got := formatHelpSkillDescription(long)
	if len([]rune(got)) > helpSkillDescMaxLen {
		t.Fatalf("runes = %d want <= %d", len([]rune(got)), helpSkillDescMaxLen)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("suffix = %q", got)
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
