package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

// opencodeSessionNamespace is a fixed UUIDv5 namespace for mapping OpenCode
// ses_* session keys to rmb-compatible UUIDs. The same OpenCode session always
// maps to the same rmb session row.
var opencodeSessionNamespace = uuid.MustParse("c4e8f1a2-6b3d-4f9e-a1c2-8d7e6f5a4b3c")

// opencodePayload covers hook payloads emitted by the rmb OpenCode plugin on
// session idle. Key distinguishing fields:
//   - agent: always "opencode" when sent by the official plugin
//   - session_db_path: points at ~/.local/share/opencode/opencode.db (optional)
//
// Unlike Pi/Cursor, OpenCode stores transcripts in SQLite rather than JSONL, so
// the plugin includes last_user_message directly instead of a session file path.
type opencodePayload struct {
	Agent                string `json:"agent"`
	SessionID            string `json:"session_id"`
	LastUserMessage      string `json:"last_user_message"`
	LastAssistantMessage string `json:"last_assistant_message"`
	SessionDBPath        string `json:"session_db_path"`
	Cwd                  string `json:"cwd"`
	HookEventName        string `json:"hook_event_name"`
}

// isOpenCodePayload returns true when the payload is OpenCode-originated.
// Detection priority:
//  1. agent == "opencode"
//  2. session_db_path under a known OpenCode data directory
func isOpenCodePayload(raw []byte) bool {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return false
	}
	var p opencodePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(p.Agent), "opencode") {
		return true
	}
	return opencodeDBPath(p.SessionDBPath)
}

func opencodeDBPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	for _, suffix := range []string{
		"/.local/share/opencode/opencode.db",
		"/Library/Application Support/opencode/opencode.db",
	} {
		if path == home+suffix {
			return true
		}
	}
	return false
}

// buildMessagesFromOpenCodePayload returns the (user, assistant) pair for an
// OpenCode session.idle / session.status idle event.
//
// Strategy:
//   - assistant = last_assistant_message from the payload (plugin extracts the
//     final assistant text after the run settles).
//   - user = last_user_message from the payload (plugin reads session messages
//     via the OpenCode SDK).
func buildMessagesFromOpenCodePayload(raw []byte) (sessionID string, messages []uploadMessage, reason string, err error) {
	var p opencodePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", nil, "", fmt.Errorf("decode opencode payload: %w", err)
	}

	sessionID, err = opencodeRMBSessionID(p.SessionID)
	if err != nil {
		return "", nil, "", err
	}

	assistant := strings.TrimSpace(p.LastAssistantMessage)
	if assistant == "" {
		return "", nil, "", fmt.Errorf("opencode payload: last_assistant_message is empty")
	}

	userText := strings.TrimSpace(p.LastUserMessage)

	out := make([]uploadMessage, 0, 2)
	if userText != "" {
		out = append(out, uploadMessage{Role: "user", Content: userText})
	}
	out = append(out, uploadMessage{Role: "assistant", Content: assistant})

	if userText == "" {
		return sessionID, out, "last_assistant_message only (no user found)", nil
	}
	return sessionID, out, "user and assistant from payload", nil
}

// opencodeRMBSessionID maps an OpenCode session key to the UUID rmb expects.
// Native OpenCode IDs (ses_*) are hashed deterministically; values that are
// already valid UUIDs pass through unchanged (lowercased).
func opencodeRMBSessionID(raw string) (string, error) {
	sessionID := strings.TrimSpace(raw)
	if sessionID == "" {
		return "", fmt.Errorf("opencode payload missing session_id")
	}
	if parsed, err := uuid.Parse(sessionID); err == nil {
		return strings.ToLower(parsed.String()), nil
	}
	derived := uuid.NewSHA1(opencodeSessionNamespace, []byte("opencode:"+sessionID))
	return strings.ToLower(derived.String()), nil
}
