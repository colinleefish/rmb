package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// cursorPayload covers both afterAgentResponse and stop payloads from Cursor.
// The presence of cursor_version (or cursor-specific fields) distinguishes
// Cursor from Claude Code even when Cursor fires ~/.claude/settings.json hooks.
type cursorPayload struct {
	ConversationID string `json:"conversation_id"`
	SessionID      string `json:"session_id"`
	// Text is the final assistant reply text, present in afterAgentResponse.
	Text           string `json:"text"`
	TranscriptPath string `json:"transcript_path"`
	// CursorVersion is only present in Cursor-originated payloads.
	CursorVersion  string   `json:"cursor_version"`
	WorkspaceRoots []string `json:"workspace_roots"`
	// Status indicates whether the generation completed or was aborted.
	// Only "completed" generations should be uploaded; aborted ones happen
	// when the user interrupts mid-response and would otherwise cause
	// duplicate uploads of the previous turn (because the transcript still
	// shows the prior assistant reply when the aborted stop fires).
	Status string `json:"status"`
}

// isCursorPayload returns true when the payload is Cursor-originated.
// Detection priority:
//  1. cursor_version field present
//  2. workspace_roots field present
//  3. transcript_path starts with ~/.cursor/
func isCursorPayload(raw []byte) bool {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return false
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return false
	}
	if strings.TrimSpace(p.CursorVersion) != "" {
		return true
	}
	if len(p.WorkspaceRoots) > 0 {
		return true
	}
	if tp := strings.TrimSpace(p.TranscriptPath); tp != "" {
		home, _ := os.UserHomeDir()
		if home != "" && strings.HasPrefix(tp, home+"/.cursor/") {
			return true
		}
	}
	return false
}

// buildMessagesFromCursorPayload extracts the latest user+assistant pair from
// a Cursor hook payload. It uses payload.Text to locate the assistant entry in
// the transcript, then walks back to find the preceding user message.
func buildMessagesFromCursorPayload(raw []byte) (sessionID string, messages []uploadMessage, reason string, err error) {
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", nil, "", fmt.Errorf("decode cursor payload: %w", err)
	}

	// Skip non-completed generations. Aborted/interrupted generations also fire
	// stop, but their transcript still shows the previous turn's assistant —
	// uploading them would duplicate the prior pair.
	if status := strings.ToLower(strings.TrimSpace(p.Status)); status != "" && status != "completed" {
		return "", nil, "", fmt.Errorf("cursor payload status=%q is not completed", p.Status)
	}

	sessionID = strings.ToLower(strings.TrimSpace(p.ConversationID))
	if sessionID == "" {
		sessionID = strings.ToLower(strings.TrimSpace(p.SessionID))
	}
	if sessionID == "" {
		return "", nil, "", fmt.Errorf("cursor payload missing conversation/session id")
	}

	messages = cursorBuildPairFromTranscript(p.TranscriptPath, p.Text)
	if len(messages) > 0 {
		return sessionID, messages, "latest user/assistant from transcript", nil
	}

	// Fallback: no transcript match — use the text field directly.
	assistant := strings.TrimSpace(p.Text)
	if assistant == "" {
		return "", nil, "", fmt.Errorf("cursor payload: no messages extracted and text is empty")
	}
	return sessionID, []uploadMessage{{Role: "assistant", Content: assistant}}, "assistant text fallback", nil
}

// cursorTranscriptRow is the shape of a single line in a Cursor transcript JSONL.
// Cursor uses top-level "role" with "message.content[]" of typed blocks.
type cursorTranscriptRow struct {
	Role    string `json:"role"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

type cursorTextMessage struct {
	Role string
	Text string
}

// cursorBuildPairFromTranscript walks the Cursor transcript at transcriptPath,
// finds the assistant entry matching assistantText (or the last assistant
// entry), then returns the preceding user message + that assistant message as
// a pair.
func cursorBuildPairFromTranscript(transcriptPath string, assistantText string) []uploadMessage {
	msgs := cursorReadTranscript(transcriptPath)
	if len(msgs) == 0 {
		return nil
	}

	assistantText = strings.TrimSpace(assistantText)
	assistantIdx := -1

	// Try exact match first.
	if assistantText != "" {
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Role == "assistant" && strings.TrimSpace(msgs[i].Text) == assistantText {
				assistantIdx = i
				break
			}
		}
	}
	// Fall back to last assistant entry.
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

	// Walk back from the assistant to find the preceding user message.
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

// cursorReadTranscript reads a Cursor transcript JSONL and returns each entry
// reduced to (role, text). Non-text content blocks (tool_use, etc.) are skipped.
func cursorReadTranscript(transcriptPath string) []cursorTextMessage {
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
	out := make([]cursorTextMessage, 0, 128)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row cursorTranscriptRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		role := strings.ToLower(strings.TrimSpace(row.Role))
		if role == "" {
			continue
		}
		var textParts []string
		for _, part := range row.Message.Content {
			if strings.ToLower(strings.TrimSpace(part.Type)) != "text" {
				continue
			}
			if t := strings.TrimSpace(part.Text); t != "" {
				textParts = append(textParts, t)
			}
		}
		if len(textParts) == 0 {
			continue
		}
		out = append(out, cursorTextMessage{Role: role, Text: strings.Join(textParts, "\n")})
	}
	return out
}
