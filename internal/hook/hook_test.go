package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── cursor.go ────────────────────────────────────────────────────────────────

func TestIsCursorPayload(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{
			"cursor_version present",
			`{"cursor_version":"3.5.1","conversation_id":"abc"}`,
			true,
		},
		{
			"workspace_roots present",
			`{"workspace_roots":["/home/user/proj"],"conversation_id":"abc"}`,
			true,
		},
		{
			"claude payload no cursor fields",
			`{"session_id":"abc","last_assistant_message":"hi","cwd":"/home"}`,
			false,
		},
		{
			"empty payload",
			`{}`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCursorPayload([]byte(tt.raw)); got != tt.want {
				t.Fatalf("isCursorPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildMessagesFromCursorPayload_SkipsAborted(t *testing.T) {
	payload := map[string]any{
		"cursor_version":  "3.5.1",
		"conversation_id": "abc",
		"transcript_path": "/nonexistent.jsonl",
		"status":          "aborted",
	}
	raw, _ := json.Marshal(payload)

	_, _, _, err := buildMessagesFromCursorPayload(raw)
	if err == nil {
		t.Fatal("expected error for aborted status")
	}
	if !strings.Contains(err.Error(), "not completed") {
		t.Fatalf("expected 'not completed' in error, got: %v", err)
	}
}

func TestBuildMessagesFromCursorPayload_LatestPair(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "session.jsonl")
	rawTranscript := strings.Join([]string{
		`{"role":"user","message":{"content":[{"type":"text","text":"q1"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"text","text":"a1"}]}}`,
		`{"role":"user","message":{"content":[{"type":"text","text":"q2"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"text","text":"a2"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(transcriptPath, []byte(rawTranscript), 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	payload := map[string]any{
		"cursor_version":  "3.5.1",
		"conversation_id": "33f5678b-06ec-4d43-9f57-3eac0e437d07",
		"text":            "a2",
		"transcript_path": transcriptPath,
	}
	raw, _ := json.Marshal(payload)

	sid, msgs, reason, err := buildMessagesFromCursorPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sid != "33f5678b-06ec-4d43-9f57-3eac0e437d07" {
		t.Fatalf("session_id = %q", sid)
	}
	if reason != "latest user/assistant from transcript" {
		t.Fatalf("reason = %q", reason)
	}
	if len(msgs) != 2 || msgs[0].Role != "user" || msgs[0].Content != "q2" {
		t.Fatalf("msgs = %v", msgs)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "a2" {
		t.Fatalf("msgs[1] = %v", msgs[1])
	}
}

// ── claude.go ────────────────────────────────────────────────────────────────

func TestIsClaudePayload(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{
			"last_assistant_message present",
			`{"session_id":"abc","last_assistant_message":"hello","cwd":"/home"}`,
			true,
		},
		{
			"cwd present, no last_assistant_message",
			`{"session_id":"abc","cwd":"/home"}`,
			true,
		},
		{
			"stop_hook_active present (false)",
			`{"session_id":"abc","stop_hook_active":false}`,
			true,
		},
		{
			"cursor payload is not claude",
			`{"cursor_version":"3.5.1","conversation_id":"abc","text":"hi"}`,
			false,
		},
		{
			"empty payload",
			`{}`,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClaudePayload([]byte(tt.raw)); got != tt.want {
				t.Fatalf("isClaudePayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildMessagesFromClaudePayload_RaceFreePairing(t *testing.T) {
	// Simulates the real CC race: at hook fire time, the transcript has the
	// CURRENT user prompt (on disk) but the CURRENT assistant text has NOT
	// been flushed yet. The previous turn's assistant IS on disk. The
	// transcript also contains tool_result and slash-command user records
	// that must be skipped.
	transcriptPath := filepath.Join(t.TempDir(), "session.jsonl")
	rawTranscript := strings.Join([]string{
		`{"type":"permission-mode","sessionId":"x"}`,
		`{"type":"file-history-snapshot","sessionId":"x"}`,
		// Previous turn — fully recorded
		`{"type":"user","message":{"role":"user","content":"first question"}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"first reply"}]}}`,
		// Tool round: assistant runs a tool, gets a tool_result back as
		// type=user — must NOT be picked as the user prompt
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"file1\nfile2"}]}}`,
		// Slash-command artifact — must NOT be picked
		`{"type":"user","message":{"role":"user","content":"<command-name>/help</command-name>"}}`,
		// CURRENT user prompt — this is what we want
		`{"type":"user","message":{"role":"user","content":"what time"}}`,
		// NOTE: the current assistant reply ("it's noon") is NOT yet in the
		// transcript — that's the race we're working around.
	}, "\n") + "\n"
	if err := os.WriteFile(transcriptPath, []byte(rawTranscript), 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	payload := map[string]any{
		"session_id":             "dad2a60d-c2f5-4682-a008-c0ee4f415338",
		"transcript_path":        transcriptPath,
		"cwd":                    "/home/user",
		"last_assistant_message": "it's noon",
		"stop_hook_active":       false,
		"hook_event_name":        "Stop",
	}
	raw, _ := json.Marshal(payload)

	sid, msgs, reason, err := buildMessagesFromClaudePayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sid != "dad2a60d-c2f5-4682-a008-c0ee4f415338" {
		t.Fatalf("session_id = %q", sid)
	}
	if reason != "user from transcript + assistant from payload" {
		t.Fatalf("reason = %q", reason)
	}
	if len(msgs) != 2 {
		t.Fatalf("len = %d, want 2; msgs = %v", len(msgs), msgs)
	}
	if msgs[0].Role != "user" || msgs[0].Content != "what time" {
		t.Fatalf("msgs[0] = %#v, want user/what time", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "it's noon" {
		t.Fatalf("msgs[1] = %#v, want assistant/it's noon", msgs[1])
	}
}

func TestBuildMessagesFromClaudePayload_AssistantOnlyWhenNoUser(t *testing.T) {
	// Brand-new session: transcript has metadata but no real user prompt yet.
	transcriptPath := filepath.Join(t.TempDir(), "session.jsonl")
	rawTranscript := strings.Join([]string{
		`{"type":"permission-mode","sessionId":"x"}`,
		`{"type":"file-history-snapshot","sessionId":"x"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(transcriptPath, []byte(rawTranscript), 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	payload := map[string]any{
		"session_id":             "abc",
		"transcript_path":        transcriptPath,
		"cwd":                    "/home/user",
		"last_assistant_message": "hello",
		"stop_hook_active":       false,
	}
	raw, _ := json.Marshal(payload)

	_, msgs, reason, err := buildMessagesFromClaudePayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != "last_assistant_message only (no user found)" {
		t.Fatalf("reason = %q", reason)
	}
	if len(msgs) != 1 || msgs[0].Role != "assistant" || msgs[0].Content != "hello" {
		t.Fatalf("msgs = %v", msgs)
	}
}

func TestBuildMessagesFromClaudePayload_EmptyAssistantErrors(t *testing.T) {
	payload := map[string]any{
		"session_id":             "abc",
		"transcript_path":        "/nonexistent.jsonl",
		"cwd":                    "/home/user",
		"last_assistant_message": "",
	}
	raw, _ := json.Marshal(payload)

	_, _, _, err := buildMessagesFromClaudePayload(raw)
	if err == nil {
		t.Fatal("expected error when last_assistant_message empty")
	}
	if !strings.Contains(err.Error(), "last_assistant_message is empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── hook.go — Submit integration ─────────────────────────────────────────────

func TestSubmit_Cursor_UploadsToAPI(t *testing.T) {
	transcriptPath := filepath.Join(t.TempDir(), "session.jsonl")
	rawTranscript := strings.Join([]string{
		`{"role":"user","message":{"content":[{"type":"text","text":"hello"}]}}`,
		`{"role":"assistant","message":{"content":[{"type":"text","text":"world"}]}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(transcriptPath, []byte(rawTranscript), 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	var gotPath string
	var gotBody uploadRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("MYPAST_URL", srv.URL)

	payload := map[string]any{
		"hook_event_name": "afterAgentResponse",
		"cursor_version":  "3.5.1",
		"conversation_id": "33f5678b-06ec-4d43-9f57-3eac0e437d07",
		"text":            "world",
		"transcript_path": transcriptPath,
	}
	raw, _ := json.Marshal(payload)

	var out bytes.Buffer
	if err := Submit(context.Background(), SubmitInput{Source: "cursor", StdinJSON: raw, OutputSink: &out}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	if gotPath != "/api/v1/sessions/33f5678b-06ec-4d43-9f57-3eac0e437d07/upload" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Content != "hello" || gotBody.Messages[1].Content != "world" {
		t.Fatalf("body messages = %v", gotBody.Messages)
	}
	if !strings.Contains(out.String(), "action=upload") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestSubmit_CC_SkipsNonClaudePayload(t *testing.T) {
	// A payload with no claude-identifying fields should be skipped.
	payload := map[string]any{
		"hook_event_name": "stop",
		"some_field":      "some_value",
	}
	raw, _ := json.Marshal(payload)

	var out bytes.Buffer
	if err := Submit(context.Background(), SubmitInput{Source: "cc", StdinJSON: raw, OutputSink: &out}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !strings.Contains(out.String(), "action=skip") {
		t.Fatalf("expected skip, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "not a claude payload") {
		t.Fatalf("expected 'not a claude payload', got: %q", out.String())
	}
}

func TestSubmit_CC_UploadsWhenFiredByClaudeCode(t *testing.T) {
	// Mimics the real CC race: user prompt on disk, but the new assistant
	// reply isn't flushed yet — it only arrives via last_assistant_message.
	transcriptPath := filepath.Join(t.TempDir(), "session.jsonl")
	rawTranscript := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":"ping"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(transcriptPath, []byte(rawTranscript), 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}

	var gotBody uploadRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	t.Setenv("MYPAST_URL", srv.URL)

	payload := map[string]any{
		"hook_event_name":        "Stop",
		"session_id":             "dad2a60d-c2f5-4682-a008-c0ee4f415338",
		"transcript_path":        transcriptPath,
		"cwd":                    "/home/user",
		"last_assistant_message": "pong",
		"stop_hook_active":       false,
	}
	raw, _ := json.Marshal(payload)

	var out bytes.Buffer
	if err := Submit(context.Background(), SubmitInput{Source: "cc", StdinJSON: raw, OutputSink: &out}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !strings.Contains(out.String(), "action=upload") {
		t.Fatalf("expected upload, got: %q", out.String())
	}
	if len(gotBody.Messages) != 2 || gotBody.Messages[0].Content != "ping" || gotBody.Messages[1].Content != "pong" {
		t.Fatalf("body messages = %v", gotBody.Messages)
	}
}

// ── hook.go — config ─────────────────────────────────────────────────────────

func TestResolveMyPastURL(t *testing.T) {
	t.Setenv("MYPAST_URL", "")
	t.Setenv("MYPAST_CONF", t.TempDir()+"/missing.conf")
	if got := resolveMyPastURL(); got != defaultMyPastURL {
		t.Fatalf("default url = %q, want %q", got, defaultMyPastURL)
	}

	confURL := "http://localhost:28080"
	confPath := t.TempDir() + "/.mypast.conf"
	if err := os.WriteFile(confPath, []byte("MYPAST_URL="+confURL+"\n"), 0o600); err != nil {
		t.Fatalf("write conf: %v", err)
	}
	t.Setenv("MYPAST_CONF", confPath)
	if got := resolveMyPastURL(); got != confURL {
		t.Fatalf("conf url = %q, want %q", got, confURL)
	}

	envURL := "http://localhost:18080"
	t.Setenv("MYPAST_URL", envURL)
	if got := resolveMyPastURL(); got != envURL {
		t.Fatalf("env url = %q, want %q", got, envURL)
	}
}
