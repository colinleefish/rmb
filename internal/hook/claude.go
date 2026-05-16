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

// buildMessagesFromClaudePayload returns the (user, assistant) pair for a
// Claude Code Stop event.
//
// Strategy (race-free):
//   - assistant = last_assistant_message from the payload itself. The
//     transcript is intentionally NOT consulted for the assistant text
//     because CC fires Stop BEFORE flushing the new assistant entry to disk.
//   - user = the last real user prompt in the transcript at fire time.
//     Tool-result entries (~2x more common than real prompts in CC) and
//     slash-command wrappers are skipped.
//
// If no real user prompt is found (e.g. brand-new session, transcript not
// yet readable), the assistant is uploaded alone.
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
	if assistant == "" {
		return "", nil, "", fmt.Errorf("claude payload: last_assistant_message is empty")
	}

	userText := claudeFindLastUserPrompt(p.TranscriptPath)

	out := make([]uploadMessage, 0, 2)
	if userText != "" {
		out = append(out, uploadMessage{Role: "user", Content: userText})
	}
	out = append(out, uploadMessage{Role: "assistant", Content: assistant})

	if userText == "" {
		return sessionID, out, "last_assistant_message only (no user found)", nil
	}
	return sessionID, out, "user from transcript + assistant from payload", nil
}

// claudeTranscriptRow is one line of a Claude Code transcript JSONL.
// CC mixes many entry types (user, assistant, attachment, system,
// permission-mode, file-history-snapshot, last-prompt, ...). For our purposes
// we only care about `type:"user"` entries — and even those we must filter,
// because CC re-uses `type:"user"` for tool_result and slash-command records.
type claudeTranscriptRow struct {
	Type    string `json:"type"`
	Message struct {
		Role string `json:"role"`
		// Content is decoded loosely because it can be either a JSON string
		// (typical for user prompts) or a JSON array of typed blocks
		// (tool_results, mixed content, etc.).
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// claudeFindLastUserPrompt scans transcriptPath linearly and returns the text
// of the LAST real user prompt. Returns "" if none is found.
//
// Filtered out:
//   - non-user types (assistant, system, attachment, permission-mode, ...)
//   - user entries whose content is a tool_result block list
//   - user entries wrapped as slash-command records (<local-command-*>,
//     <command-name>, etc.)
//   - empty content
func claudeFindLastUserPrompt(transcriptPath string) string {
	transcriptPath = strings.TrimSpace(transcriptPath)
	if transcriptPath == "" {
		return ""
	}
	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 8*1024*1024)
	last := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row claudeTranscriptRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		// Use top-level type as the source of truth; message.role can be
		// set for non-prompt records too.
		if strings.ToLower(strings.TrimSpace(row.Type)) != "user" {
			continue
		}
		if text := extractRealUserText(row.Message.Content); text != "" {
			last = text
		}
	}
	return last
}

// extractRealUserText returns the user prompt text from a CC user entry, or
// "" if the entry is not a real prompt (tool_result, slash command, empty).
func extractRealUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// String content (typical for user-typed prompts).
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		s := strings.TrimSpace(asString)
		if s == "" || isClaudeCommandWrapper(s) {
			return ""
		}
		return s
	}

	// Array of blocks. tool_result blocks → reject the whole entry.
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var textParts []string
	for _, b := range blocks {
		t := strings.ToLower(strings.TrimSpace(b.Type))
		if t == "tool_result" {
			return ""
		}
		if t == "text" {
			if txt := strings.TrimSpace(b.Text); txt != "" {
				textParts = append(textParts, txt)
			}
		}
	}
	if len(textParts) == 0 {
		return ""
	}
	joined := strings.Join(textParts, "\n")
	if isClaudeCommandWrapper(joined) {
		return ""
	}
	return joined
}

// isClaudeCommandWrapper detects slash-command / local-command records that
// CC encodes as user entries but are not real user prompts.
func isClaudeCommandWrapper(s string) bool {
	if strings.HasPrefix(s, "<local-command-") {
		return true
	}
	if strings.Contains(s, "<command-name>") {
		return true
	}
	if strings.Contains(s, "<command-message>") {
		return true
	}
	if strings.Contains(s, "<local-command-stdout>") {
		return true
	}
	if strings.Contains(s, "<local-command-caveat>") {
		return true
	}
	return false
}
