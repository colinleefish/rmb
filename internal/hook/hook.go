// Package hook is the adapter layer that translates agent transcript hook
// payloads (Cursor afterAgentResponse, Claude Code Stop, etc.) into mem9
// session upload API calls. Source-specific payload parsing lives in
// cursor.go (Cursor) and claude.go (Claude Code).
package hook

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

const defaultMem9URL = "http://127.0.0.1:8080"

// Submit is the single entry point for all hook invocations.
// It routes to source-specific parsing based on --source, then POSTs to the
// mem9 upload API.
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

	targetURL := resolveMem9URL()

	logf := func(action, reason string, extra ...any) error {
		msg := fmt.Sprintf("mem9 hook-submit source=%s action=%s reason=%s target=%s", source, action, reason, targetURL)
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
	if user, pass := resolveMem9Auth(); user != "" {
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

func resolveMem9URL() string {
	if v := strings.TrimSpace(os.Getenv("MEM9_URL")); v != "" {
		return v
	}
	confPath := strings.TrimSpace(os.Getenv("MEM9_CONF"))
	if confPath == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			confPath = filepath.Join(home, ".mem9.conf")
		}
	}
	if confPath != "" {
		if v := readEnvValueFromFile(confPath, "MEM9_URL"); v != "" {
			return v
		}
	}
	return defaultMem9URL
}

func resolveMem9Auth() (string, string) {
	user := strings.TrimSpace(os.Getenv("MEM9_USERNAME"))
	pass := strings.TrimSpace(os.Getenv("MEM9_PASSWORD"))
	if user == "" {
		user = strings.TrimSpace(os.Getenv("USERNAME"))
	}
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("PASSWORD"))
	}

	confPath := strings.TrimSpace(os.Getenv("MEM9_CONF"))
	if confPath == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			confPath = filepath.Join(home, ".mem9.conf")
		}
	}
	if confPath != "" {
		if v := readEnvValueFromFile(confPath, "MEM9_USERNAME"); v != "" {
			user = v
		} else if v := readEnvValueFromFile(confPath, "USERNAME"); v != "" {
			user = v
		}
		if v := readEnvValueFromFile(confPath, "MEM9_PASSWORD"); v != "" {
			pass = v
		} else if v := readEnvValueFromFile(confPath, "PASSWORD"); v != "" {
			pass = v
		}
	}
	return user, pass
}

func readEnvValueFromFile(path string, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		return strings.TrimSpace(v)
	}
	return ""
}
