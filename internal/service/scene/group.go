package scene

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

const defaultSceneName = "General"

// sceneNamespace is a fixed UUIDv5 namespace for deriving stable scene IDs.
// Stable scene URIs let upper tiers (T3 memories.source_scene_uris) keep valid
// references across T2 rebuilds instead of pointing at re-minted UUIDs.
var sceneNamespace = uuid.MustParse("b6f6e2c2-7c1a-4e2b-9c3d-7a1f0d2e4b88")

// sceneIDForName derives a deterministic scene id from the owning session and
// the scene's display name. The same (session, name) always yields the same id,
// so rebuilding a session's scenes reuses existing rows. `dup` disambiguates the
// rare case of two scenes sharing a name within one rebuild.
func sceneIDForName(sessionID uuid.UUID, displayName string, dup int) uuid.UUID {
	name := strings.ToLower(strings.TrimSpace(displayName))
	seed := sessionID.String() + "\x00" + name
	if dup > 1 {
		seed += "\x00" + strconv.Itoa(dup)
	}
	return uuid.NewSHA1(sceneNamespace, []byte(seed))
}

func sceneURIForName(sessionID uuid.UUID, displayName string, dup int) string {
	return uri.BuildScene(sceneIDForName(sessionID, displayName, dup).String())
}

type atomInput struct {
	URI       string  `json:"uri"`
	Category  string  `json:"category"`
	Priority  int     `json:"priority"`
	SceneName string  `json:"scene_name"`
	Slug      *string `json:"slug,omitempty"`
	Content   string  `json:"content"`
}

type atomGroup struct {
	DisplayName string
	Atoms       []model.Atom
}

func groupAtomsBySceneName(atoms []model.Atom) []atomGroup {
	byName := make(map[string][]model.Atom)
	order := make([]string, 0)
	for _, atom := range atoms {
		name := defaultSceneName
		if atom.SceneName != nil {
			if trimmed := strings.TrimSpace(*atom.SceneName); trimmed != "" {
				name = trimmed
			}
		}
		if _, ok := byName[name]; !ok {
			order = append(order, name)
		}
		byName[name] = append(byName[name], atom)
	}
	sort.Strings(order)

	out := make([]atomGroup, 0, len(order))
	for _, name := range order {
		out = append(out, atomGroup{
			DisplayName: name,
			Atoms:       byName[name],
		})
	}
	return out
}

// chunkGroups splits scene-name groups into batches whose atom counts stay
// under maxAtoms, so each BuildScenes LLM call produces a response small enough
// to return as complete JSON. A single group larger than maxAtoms is emitted
// alone (best effort) rather than split, to keep a scene_name intact.
func chunkGroups(groups []atomGroup, maxAtoms int) [][]atomGroup {
	if maxAtoms <= 0 || len(groups) == 0 {
		return [][]atomGroup{groups}
	}
	var chunks [][]atomGroup
	var cur []atomGroup
	curCount := 0
	for _, g := range groups {
		n := len(g.Atoms)
		if len(cur) > 0 && curCount+n > maxAtoms {
			chunks = append(chunks, cur)
			cur = nil
			curCount = 0
		}
		cur = append(cur, g)
		curCount += n
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	return chunks
}

func serializeAtomsForLLM(groups []atomGroup) (string, error) {
	inputs := make([]atomInput, 0)
	for _, group := range groups {
		for _, atom := range group.Atoms {
			sceneName := defaultSceneName
			if atom.SceneName != nil {
				sceneName = strings.TrimSpace(*atom.SceneName)
			}
			inputs = append(inputs, atomInput{
				URI:       atom.URI,
				Category:  atom.Category,
				Priority:  atom.Priority,
				SceneName: sceneName,
				Slug:      atom.Slug,
				Content:   atom.Content,
			})
		}
	}
	raw, err := json.Marshal(map[string]any{"atoms": inputs})
	if err != nil {
		return "", fmt.Errorf("marshal atoms for llm: %w", err)
	}
	return string(raw), nil
}

func atomURIs(atoms []model.Atom) map[string]struct{} {
	out := make(map[string]struct{}, len(atoms))
	for _, atom := range atoms {
		out[atom.URI] = struct{}{}
	}
	return out
}
