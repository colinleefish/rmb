package hook

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// piPayload covers hook payloads emitted by the rmb Pi extension on
// agent_settled. Key distinguishing fields:
//   - agent: always "pi" when sent by the official extension
//   - session_file / transcript_path: points into ~/.pi/agent/sessions/
//
// Shared fields with Claude/Codex (session_id, cwd, last_assistant_message)
// are present for parity; session path or agent="pi" is the discriminator.
type piPayload struct {
	Agent                string `json:"agent"`
	SessionID            string `json:"session_id"`
	SessionFile          string `json:"session_file"`
	TranscriptPath       string `json:"transcript_path"`
	Cwd                  string `json:"cwd"`
	LastAssistantMessage string `json:"last_assistant_message"`
	HookEventName        string `json:"hook_event_name"`
}

// isPiPayload returns true when the payload is Pi-originated.
// Detection priority:
//  1. agent == "pi"
//  2. session_file or transcript_path under ~/.pi/agent/
func isPiPayload(raw []byte) bool {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return false
	}
	var p piPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(p.Agent), "pi") {
		return true
	}
	for _, path := range []string{p.SessionFile, p.TranscriptPath} {
		if piSessionPath(path) {
			return true
		}
	}
	return false
}

func piSessionPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	return strings.HasPrefix(path, home+"/.pi/agent/")
}

// buildMessagesFromPiPayload returns the (user, assistant) pair for a Pi
// agent_settled event.
//
// Strategy:
//   - assistant = last_assistant_message from the payload (extension extracts
//     the final assistant text after the run settles).
//   - user = the last user message in the Pi session JSONL. Pi stores messages
//     as type:"message" entries with message.role:"user".
func buildMessagesFromPiPayload(raw []byte) (sessionID string, messages []uploadMessage, reason string, err error) {
	var p piPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", nil, "", fmt.Errorf("decode pi payload: %w", err)
	}

	sessionID = strings.ToLower(strings.TrimSpace(p.SessionID))
	if sessionID == "" {
		return "", nil, "", fmt.Errorf("pi payload missing session_id")
	}

	assistant := strings.TrimSpace(p.LastAssistantMessage)
	if assistant == "" {
		return "", nil, "", fmt.Errorf("pi payload: last_assistant_message is empty")
	}

	transcriptPath := strings.TrimSpace(p.SessionFile)
	if transcriptPath == "" {
		transcriptPath = strings.TrimSpace(p.TranscriptPath)
	}
	userText := piFindLastUserMessage(transcriptPath)

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

// piSessionLine is one line of a Pi session JSONL file.
// User prompts appear as type:"message" with message.role:"user".
type piSessionLine struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// piFindLastUserMessage scans the Pi session file and returns the text of the
// last user message. Returns "" if none is found.
func piFindLastUserMessage(transcriptPath string) string {
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
		var row piSessionLine
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if row.Type != "message" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(row.Message.Role)) != "user" {
			continue
		}
		if text := piExtractUserText(row.Message.Content); text != "" {
			last = text
		}
	}
	return last
}

func piExtractUserText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

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
