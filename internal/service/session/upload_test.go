package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/colinleefish/mem9/internal/uri"
)

func TestBuildSessionTurnURI(t *testing.T) {
	got := uri.BuildSessionTurn("4f1916ce-2f6e-4b76-8249-4a5f4184fd8d", 0)
	want := "mem9://sessions/4f1916ce-2f6e-4b76-8249-4a5f4184fd8d/turns/0"
	if got != want {
		t.Fatalf("unexpected turn uri: %s", got)
	}
}

func TestValidateSessionID(t *testing.T) {
	got, err := validateSessionID("4F1916CE-2F6E-4B76-8249-4A5F4184FD8D")
	if err != nil {
		t.Fatalf("validateSessionID returned error: %v", err)
	}
	if got != "4f1916ce-2f6e-4b76-8249-4a5f4184fd8d" {
		t.Fatalf("unexpected session id: %s", got)
	}

	if _, err := validateSessionID("bad/id"); err == nil {
		t.Fatalf("expected error for invalid session id")
	}
	if _, err := validateSessionID("not-a-uuid"); err == nil {
		t.Fatalf("expected error for invalid session id")
	}
}

func TestBuildMessagesJSONL(t *testing.T) {
	uploaded := time.Date(2026, time.May, 9, 10, 0, 0, 0, time.UTC)

	raw, err := buildMessagesJSONL([]Message{
		{Role: "user", Content: "it fails on submit"},
		{Role: "assistant", Content: ""},
	}, uploaded)
	if err != nil {
		t.Fatalf("buildMessagesJSONL returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl lines, got %d", len(lines))
	}

	var line0 sessionMessageLine
	if err := json.Unmarshal([]byte(lines[0]), &line0); err != nil {
		t.Fatalf("line 0 is not valid json: %v", err)
	}
	if line0.Role != "user" || line0.Content != "it fails on submit" {
		t.Fatalf("unexpected first line payload: %#v", line0)
	}
	if line0.ID != "msg_000001" {
		t.Fatalf("unexpected first line id: %s", line0.ID)
	}
}
