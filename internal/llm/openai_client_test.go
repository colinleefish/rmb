package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildThinkingBody(t *testing.T) {
	cases := []struct {
		style   string
		enabled bool
		wantKey string // "" means expect nil body
		check   func(map[string]any) bool
	}{
		{"", false, "", nil},
		{"thinking_type", false, "thinking", func(m map[string]any) bool {
			return m["thinking"].(map[string]any)["type"] == "disabled"
		}},
		{"thinking_type", true, "thinking", func(m map[string]any) bool {
			return m["thinking"].(map[string]any)["type"] == "enabled"
		}},
		{"enable_thinking", false, "enable_thinking", func(m map[string]any) bool {
			return m["enable_thinking"] == false
		}},
		{"reasoning_effort", false, "reasoning_effort", func(m map[string]any) bool {
			return m["reasoning_effort"] == "none"
		}},
		{"reasoning_effort", true, "", nil}, // enabled has no single value -> omit
	}
	for _, c := range cases {
		body, err := buildThinkingBody(c.style, c.enabled)
		if err != nil {
			t.Fatalf("style %q: unexpected error %v", c.style, err)
		}
		if c.wantKey == "" {
			if body != nil {
				t.Fatalf("style %q enabled=%v: expected nil body, got %+v", c.style, c.enabled, body)
			}
			continue
		}
		if _, ok := body[c.wantKey]; !ok {
			t.Fatalf("style %q: missing key %q in %+v", c.style, c.wantKey, body)
		}
		if c.check != nil && !c.check(body) {
			t.Fatalf("style %q enabled=%v: check failed for %+v", c.style, c.enabled, body)
		}
	}

	if _, err := buildThinkingBody("bogus", false); err == nil {
		t.Fatal("expected error for unknown style")
	}
}

func TestChatRequestMergesThinkingBody(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	client, err := NewOpenAICompatibleClient(OpenAICompatibleConfig{
		Provider:        "openai",
		APIBase:         srv.URL,
		APIKey:          "k",
		Model:           "m",
		MaxRetries:      1,
		Timeout:         5 * time.Second,
		ThinkingStyle:   "thinking_type",
		ThinkingEnabled: false,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	if _, err := client.ExtractAtoms(context.Background(), `{"role":"user","content":"hi"}`); err != nil {
		t.Fatalf("ExtractAtoms: %v", err)
	}
	think, ok := captured["thinking"].(map[string]any)
	if !ok || think["type"] != "disabled" {
		t.Fatalf("request did not carry disabled thinking: %+v", captured["thinking"])
	}
}

func TestExtractAtomsRetriesAndSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&attempts, 1)
		if cur == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"temporary"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"extracted atoms"}}]}`))
	}))
	defer srv.Close()

	client, err := NewOpenAICompatibleClient(OpenAICompatibleConfig{
		Provider:   "openai",
		APIBase:    srv.URL,
		APIKey:     "test-key",
		Model:      "test-model",
		MaxRetries: 2,
		Timeout:    5 * time.Second,
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	got, err := client.ExtractAtoms(context.Background(), `{"role":"user","content":"hello"}`)
	if err != nil {
		t.Fatalf("ExtractAtoms error: %v", err)
	}
	if got != "extracted atoms" {
		t.Fatalf("unexpected output: %q", got)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
