package scene

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

type llmScene struct {
	DisplayName string   `json:"display_name"`
	Abstract    string   `json:"abstract"`
	Body        string   `json:"body"`
	AtomURIs    []string `json:"atom_uris"`
}

type llmBuildScenesResponse struct {
	Scenes []llmScene `json:"scenes"`
}

type parsedScene struct {
	DisplayName string
	Abstract    string
	Body        string
	SourceAtoms []uuid.UUID
}

func parseBuildScenesResponse(raw string, validURIs map[string]struct{}) ([]parsedScene, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty llm response")
	}

	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			end := len(lines)
			if strings.TrimSpace(lines[end-1]) == "```" {
				end--
			}
			raw = strings.Join(lines[1:end], "\n")
		}
	}

	var resp llmBuildScenesResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("decode scenes json: %w", err)
	}
	if len(resp.Scenes) == 0 {
		return nil, fmt.Errorf("no scenes in llm response")
	}

	out := make([]parsedScene, 0, len(resp.Scenes))
	for _, s := range resp.Scenes {
		abstract := strings.TrimSpace(s.Abstract)
		body := strings.TrimSpace(s.Body)
		if abstract == "" || body == "" {
			continue
		}
		displayName := strings.TrimSpace(s.DisplayName)
		if displayName == "" {
			displayName = defaultSceneName
		}

		// Tolerate hallucinated/unknown atom URIs: drop them rather than failing
		// the whole batch (one bad URI must not wedge a session into a retry
		// loop). A scene with no valid atom refs left is skipped.
		atomIDs := make([]uuid.UUID, 0, len(s.AtomURIs))
		seen := make(map[uuid.UUID]struct{})
		for _, u := range s.AtomURIs {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			if _, ok := validURIs[u]; !ok {
				continue
			}
			id, err := uri.ParseAtomID(u)
			if err != nil {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			atomIDs = append(atomIDs, id)
		}
		if len(atomIDs) == 0 {
			continue
		}

		out = append(out, parsedScene{
			DisplayName: displayName,
			Abstract:    abstract,
			Body:        body,
			SourceAtoms: atomIDs,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no usable scenes in llm response")
	}
	return out, nil
}

func joinSceneAbstracts(scenes []parsedScene) string {
	var b strings.Builder
	for i, s := range scenes {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(s.DisplayName)
		b.WriteString(": ")
		b.WriteString(s.Abstract)
	}
	return b.String()
}
