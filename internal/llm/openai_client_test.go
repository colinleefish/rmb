package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildMergeOverviewPrompt(t *testing.T) {
	prompt := buildMergeOverviewPrompt("prev text", "{\"role\":\"user\",\"content\":\"hello\"}")
	if prompt == "" {
		t.Fatalf("prompt should not be empty")
	}
	if want := "prev text"; !strings.Contains(prompt, want) {
		t.Fatalf("prompt should include previous overview")
	}
}

func TestMergeOverviewRetriesAndSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cur := atomic.AddInt32(&attempts, 1)
		if cur == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"temporary"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"new merged overview"}}]}`))
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

	got, err := client.MergeOverview(context.Background(), "old", `{"role":"user","content":"hello"}`)
	if err != nil {
		t.Fatalf("MergeOverview error: %v", err)
	}
	if got != "new merged overview" {
		t.Fatalf("unexpected overview: %q", got)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}
