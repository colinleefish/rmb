package llm

import (
	"strings"
	"testing"
)

func TestParseAliasVerdict_plainJSON(t *testing.T) {
	v, err := parseAliasVerdict(`{"same":true,"canonical_uri":"rmb://entities/x","rationale":"same thing"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v.Same || v.CanonicalURI != "rmb://entities/x" || v.Rationale != "same thing" {
		t.Fatalf("unexpected verdict: %+v", v)
	}
}

func TestParseAliasVerdict_toleratesFenceAndProse(t *testing.T) {
	raw := "Here is my answer:\n```json\n{\"same\": false, \"canonical_uri\": \"rmb://entities/a\", \"rationale\": \"dev vs prod\"}\n```\n"
	v, err := parseAliasVerdict(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Same {
		t.Fatalf("expected same=false, got %+v", v)
	}
	if v.Rationale != "dev vs prod" {
		t.Fatalf("rationale not parsed: %+v", v)
	}
}

func TestParseAliasVerdict_noJSON(t *testing.T) {
	if _, err := parseAliasVerdict("I cannot decide"); err == nil {
		t.Fatal("expected error when reply has no JSON object")
	}
}

func TestBuildJudgeAliasPrompt_substitutesAll(t *testing.T) {
	p := buildJudgeAliasPrompt("rmb://entities/a", "body A", "rmb://entities/b", "body B")
	for _, want := range []string{"rmb://entities/a", "body A", "rmb://entities/b", "body B"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q:\n%s", want, p)
		}
	}
	if strings.Contains(p, "{{A_URI}}") || strings.Contains(p, "{{B_BODY}}") {
		t.Fatalf("placeholders not replaced:\n%s", p)
	}
}
