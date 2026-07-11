// Package hook is the adapter layer that translates agent transcript hook
// payloads (Cursor afterAgentResponse, Claude Code Stop, etc.) into rmb
// session upload API calls. Source-specific payload parsing lives in
// cursor.go (Cursor) and claude.go (Claude Code).
package hook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/config"
)

// SubmitInput is the contract for a hook-submit invocation.
type SubmitInput struct {
	Source     string // "cursor" | "cc" | ...
	StdinJSON  []byte
	OutputSink io.Writer
}

type uploadMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type uploadRequest struct {
	Messages []uploadMessage `json:"messages"`
}

const defaultRMBURL = "http://127.0.0.1:8080"

// Submit is the single entry point for all hook invocations.
// It routes to source-specific parsing based on --source, then POSTs to the
// rmb upload API.
//
// Routing:
//   - source=cursor    → must pass isCursorPayload; use cursor extraction
//   - source=cc|claude → must pass isClaudePayload; use claude extraction
//   - anything else    → error (CLI enforces non-empty --source)
func Submit(ctx context.Context, in SubmitInput) error {
	out := in.OutputSink
	if out == nil {
		out = io.Discard
	}

	source := strings.ToLower(strings.TrimSpace(in.Source))
	if source == "" {
		return fmt.Errorf("hook-submit: source is required")
	}

	targetURL := resolveRMBURL()

	logf := func(action, reason string, extra ...any) error {
		msg := fmt.Sprintf("rmb hook-submit source=%s action=%s reason=%s target=%s", source, action, reason, targetURL)
		for i := 0; i+1 < len(extra); i += 2 {
			msg += fmt.Sprintf(" %s=%v", extra[i], extra[i+1])
		}
		_, err := fmt.Fprintln(out, msg)
		if err != nil {
			return fmt.Errorf("write hook-submit result: %w", err)
		}
		return nil
	}

	var sessionID string
	var messages []uploadMessage
	var parseReason string
	var err error

	switch source {
	case "cursor":
		if !isCursorPayload(in.StdinJSON) {
			return logf("skip", "not a cursor payload")
		}
		sessionID, messages, parseReason, err = buildMessagesFromCursorPayload(in.StdinJSON)

	case "cc", "claude":
		source = "cc"
		if !isClaudePayload(in.StdinJSON) {
			return logf("skip", "not a claude payload")
		}
		sessionID, messages, parseReason, err = buildMessagesFromClaudePayload(in.StdinJSON)

	case "codex":
		if !isCodexPayload(in.StdinJSON) {
			return logf("skip", "not a codex payload")
		}
		sessionID, messages, parseReason, err = buildMessagesFromCodexPayload(in.StdinJSON)

	case "pi":
		if !isPiPayload(in.StdinJSON) {
			return logf("skip", "not a pi payload")
		}
		sessionID, messages, parseReason, err = buildMessagesFromPiPayload(in.StdinJSON)

	default:
		return fmt.Errorf("hook-submit: unknown source %q", source)
	}

	if err != nil {
		return logf("skip", err.Error())
	}

	statusCode, err := postUpload(ctx, targetURL, sessionID, messages)
	if err != nil {
		return logf("error", err.Error(), "session_id", sessionID, "messages", len(messages))
	}

	return logf("upload", parseReason, "session_id", sessionID, "messages", len(messages), "status", statusCode)
}

func postUpload(
	ctx context.Context,
	targetURL string,
	sessionID string,
	messages []uploadMessage,
) (int, error) {
	if strings.TrimSpace(sessionID) == "" {
		return 0, fmt.Errorf("session id is required")
	}
	if len(messages) == 0 {
		return 0, fmt.Errorf("upload messages must not be empty")
	}

	body, err := json.Marshal(uploadRequest{Messages: messages})
	if err != nil {
		return 0, fmt.Errorf("encode upload request: %w", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(targetURL), "/") +
		"/api/v1/sessions/" + sessionID + "/upload"

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if user, pass := resolveRMBAuth(); user != "" {
		req.SetBasicAuth(user, pass)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("post upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return resp.StatusCode, fmt.Errorf(
			"upload request failed status=%d body=%s",
			resp.StatusCode,
			strings.TrimSpace(string(respBody)),
		)
	}
	return resp.StatusCode, nil
}

func resolveRMBURL() string {
	if v := config.EnvValue("RMB_URL"); v != "" {
		return v
	}
	return defaultRMBURL
}

func resolveRMBAuth() (string, string) {
	user := firstNonEmpty(
		config.EnvValue("RMB_USERNAME"),
		config.EnvValue("USERNAME"),
	)
	pass := firstNonEmpty(
		config.EnvValue("RMB_PASSWORD"),
		config.EnvValue("PASSWORD"),
	)
	return user, pass
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
