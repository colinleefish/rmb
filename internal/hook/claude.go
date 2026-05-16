package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// claudePayload covers the Claude Code Stop hook payload shape.
// Key distinguishing fields absent from Cursor payloads:
//   - last_assistant_message (the final reply text, ready to use directly)
//   - cwd
//   - stop_hook_active
//   - permission_mode
//
// transcript_path points into ~/.claude/projects/ for genuine CC invocations.
type claudePayload struct {
	SessionID            string `json:"session_id"`
	TranscriptPath       string `json:"transcript_path"`
	Cwd                  string `json:"cwd"`
	LastAssistantMessage string `json:"last_assistant_message"`
	StopHookActive       bool   `json:"stop_hook_active"`
	PermissionMode       string `json:"permission_mode"`
	HookEventName        string `json:"hook_event_name"`
}

// isClaudePayload returns true when the payload is Claude Code-originated.
// Detection priority:
//  1. last_assistant_message present (CC-exclusive field)
//  2. cwd present (CC-exclusive)
//  3. stop_hook_active present (CC-exclusive)
//  4. transcript_path starts with ~/.claude/
func isClaudePayload(raw []byte) bool {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return false
	}
	var p claudePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return false
	}
	if strings.TrimSpace(p.LastAssistantMessage) != "" {
		return true
	}
	if strings.TrimSpace(p.Cwd) != "" {
		return true
	}

	// stop_hook_active is a bool; detect via raw map to distinguish
	// explicit false from field absence.
	var raw2 map[string]json.RawMessage
	if err := json.Unmarshal(raw, &raw2); err == nil {
		if _, ok := raw2["stop_hook_active"]; ok {
			return true
		}
	}

	if tp := strings.TrimSpace(p.TranscriptPath); tp != "" {
		home, _ := os.UserHomeDir()
		if home != "" && strings.HasPrefix(tp, home+"/.claude/") {
			return true
		}
	}
	return false
}

// buildMessagesFromClaudePayload extracts the latest user+assistant pair from
// a Claude Code Stop payload. It prefers last_assistant_message (provided
// directly by CC) and pairs it with the preceding user message from the
// transcript.
func buildMessagesFromClaudePayload(raw []byte) (sessionID string, messages []uploadMessage, reason string, err error) {
	var p claudePayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", nil, "", fmt.Errorf("decode claude payload: %w", err)
	}

	sessionID = strings.ToLower(strings.TrimSpace(p.SessionID))
	if sessionID == "" {
		return "", nil, "", fmt.Errorf("claude payload missing session_id")
	}

	assistant := strings.TrimSpace(p.LastAssistantMessage)

	// Walk the CC transcript to find the matching assistant and its preceding
	// user message.
	if msgs := claudeBuildPairFromTranscript(p.TranscriptPath, assistant); len(msgs) > 0 {
		return sessionID, msgs, "latest user/assistant from transcript", nil
	}

	// Fallback: use last_assistant_message directly, no user context.
	if assistant == "" {
		return "", nil, "", fmt.Errorf("claude payload: no messages extracted and last_assistant_message is empty")
	}
	return sessionID, []uploadMessage{{Role: "assistant", Content: assistant}}, "last_assistant_message fallback", nil
}

// claudeTranscriptRow is one line of a Claude Code transcript JSONL.
// CC mixes many entry types (user, assistant, attachment, system,
// permission-mode, file-history-snapshot, last-prompt, ...) — only user and
// assistant carry conversation text. message.content can be a string (typical
// for user) OR a list of typed blocks (typical for assistant).
type claudeTranscriptRow struct {
	Type    string `json:"type"`
	Message struct {
		Role string `json:"role"`
		// Content is decoded loosely because it can be either a JSON string
		// or a JSON array of blocks. The caller normalizes it.
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type claudeTextMessage struct {
	Role string
	Text string
}

// claudeBuildPairFromTranscript finds the assistant entry matching
// assistantText (or the last assistant entry if no match / empty), then walks
// back to find the preceding user message.
func claudeBuildPairFromTranscript(transcriptPath string, assistantText string) []uploadMessage {
	msgs := claudeReadTranscript(transcriptPath)
	if len(msgs) == 0 {
		return nil
	}

	assistantText = strings.TrimSpace(assistantText)
	assistantIdx := -1

	if assistantText != "" {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "assistant" && strings.TrimSpace(msgs[i].Text) == assistantText {
				assistantIdx = i
				break
			}
		}
	}
	if assistantIdx < 0 {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "assistant" {
				assistantIdx = i
				if assistantText == "" {
					assistantText = msgs[i].Text
				}
				break
			}
		}
	}
	if assistantIdx < 0 || strings.TrimSpace(assistantText) == "" {
		return nil
	}

	userText := ""
	for i := assistantIdx - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			userText = msgs[i].Text
			break
		}
	}

	out := make([]uploadMessage, 0, 2)
	if strings.TrimSpace(userText) != "" {
		out = append(out, uploadMessage{Role: "user", Content: userText})
	}
	out = append(out, uploadMessage{Role: "assistant", Content: assistantText})
	return out
}

// claudeReadTranscript reads a CC transcript JSONL and returns the
// conversation entries reduced to (role, text). Non-conversation entry types
// (attachment, system, permission-mode, file-history-snapshot, etc.) are
// skipped. Assistant non-text blocks (thinking, tool_use, ...) are skipped.
func claudeReadTranscript(transcriptPath string) []claudeTextMessage {
	transcriptPath = strings.TrimSpace(transcriptPath)
	if transcriptPath == "" {
		return nil
	}
	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 8*1024*1024)
	out := make([]claudeTextMessage, 0, 128)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row claudeTranscriptRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		// Filter to conversation types only. Use message.role with type as
		// fallback so we tolerate either field.
		role := strings.ToLower(strings.TrimSpace(row.Message.Role))
		if role == "" {
			role = strings.ToLower(strings.TrimSpace(row.Type))
		}
		if role != "user" && role != "assistant" {
			continue
		}
		text := normalizeClaudeContent(row.Message.Content)
		if text == "" {
			continue
		}
		out = append(out, claudeTextMessage{Role: role, Text: text})
	}
	return out
}

// normalizeClaudeContent collapses CC's polymorphic content field into a
// single text string. Content is either:
//   - a JSON string (typical for user messages)
//   - a JSON array of blocks; each block has "type" and may have "text"
//     ("text" blocks contribute; "thinking", "tool_use", "tool_result",
//     "image", etc. are ignored)
func normalizeClaudeContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// String content
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	// Array of blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, b := range blocks {
		if strings.ToLower(strings.TrimSpace(b.Type)) != "text" {
			continue
		}
		if t := strings.TrimSpace(b.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, "\n")
}
