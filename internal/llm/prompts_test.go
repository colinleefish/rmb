package llm

import (
	"strings"
	"testing"
)

func TestBuildDistillMemoryPromptWithCorrections(t *testing.T) {
	prompt := buildDistillMemoryPrompt(
		"entities",
		"song-xin-yang",
		`{"facts":[{"uri":"a","priority":1,"content":"works in a shopping mall"}]}`,
		[]string{"she works at a bank", "she is a woman"},
	)

	if !strings.Contains(prompt, "AUTHORITATIVE human corrections") {
		t.Fatalf("prompt should include the corrections header:\n%s", prompt)
	}
	for _, want := range []string{"- she works at a bank", "- she is a woman"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt should include correction %q:\n%s", want, prompt)
		}
	}
	if !strings.Contains(prompt, "works in a shopping mall") {
		t.Fatalf("prompt should still include the supplied facts:\n%s", prompt)
	}
	if strings.Contains(prompt, "{{CORRECTIONS}}") {
		t.Fatalf("template placeholder must be replaced:\n%s", prompt)
	}
}

func TestBuildDistillMemoryPromptNoCorrections(t *testing.T) {
	prompt := buildDistillMemoryPrompt("preferences", "explicit-commit", `{"facts":[]}`, nil)

	if strings.Contains(prompt, "AUTHORITATIVE human corrections") {
		t.Fatalf("no corrections header expected when none supplied:\n%s", prompt)
	}
	if strings.Contains(prompt, "{{CORRECTIONS}}") {
		t.Fatalf("template placeholder must be replaced even when empty:\n%s", prompt)
	}
	// Blank-only corrections collapse to no block.
	blank := buildDistillMemoryPrompt("preferences", "x", `{"facts":[]}`, []string{"   ", ""})
	if strings.Contains(blank, "AUTHORITATIVE human corrections") {
		t.Fatalf("blank corrections must not render a header:\n%s", blank)
	}
}
