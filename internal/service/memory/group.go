package memory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/colinleefish/rmb/internal/model"
	"github.com/colinleefish/rmb/internal/uri"
	"github.com/google/uuid"
)

// memoryBucket is one logical long-term memory target: a single profile, or a
// (category, slug) pair for preferences/entities/events. Its atoms are the
// cross-session facts that roll up into the memory at URI.
type memoryBucket struct {
	Category string
	Slug     string // empty for profile
	URI      string
	Atoms    []model.Atom
}

type atomLLMInput struct {
	URI      string `json:"uri"`
	Priority int    `json:"priority"`
	Content  string `json:"content"`
}

// groupAtomsIntoBuckets routes atoms into rollup buckets. profile atoms collapse
// into the singleton bucket; preferences/entities/events group by sanitized slug.
// Atoms in slug categories without a usable slug are skipped (returned count) in v1.
func groupAtomsIntoBuckets(atoms []model.Atom) ([]memoryBucket, int) {
	profile := make([]model.Atom, 0)
	type key struct{ category, slug string }
	slugged := make(map[key]*memoryBucket)
	order := make([]key, 0)
	skipped := 0

	for _, atom := range atoms {
		switch atom.Category {
		case model.AtomCategoryProfile:
			profile = append(profile, atom)
		case model.AtomCategoryPreferences, model.AtomCategoryEntities, model.AtomCategoryEvents:
			rawSlug := ""
			if atom.Slug != nil {
				rawSlug = strings.TrimSpace(*atom.Slug)
			}
			if rawSlug == "" {
				skipped++
				continue
			}
			slug, err := uri.SanitizeSlug(rawSlug)
			if err != nil {
				skipped++
				continue
			}
			k := key{atom.Category, slug}
			b, ok := slugged[k]
			if !ok {
				b = &memoryBucket{
					Category: atom.Category,
					Slug:     slug,
					URI:      uri.BuildMemory(atom.Category, slug),
				}
				slugged[k] = b
				order = append(order, k)
			}
			b.Atoms = append(b.Atoms, atom)
		default:
			skipped++
		}
	}

	buckets := make([]memoryBucket, 0, len(order)+1)
	if len(profile) > 0 {
		buckets = append(buckets, memoryBucket{
			Category: model.AtomCategoryProfile,
			Slug:     "",
			URI:      uri.BuildProfile(),
			Atoms:    profile,
		})
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].category != order[j].category {
			return order[i].category < order[j].category
		}
		return order[i].slug < order[j].slug
	})
	for _, k := range order {
		buckets = append(buckets, *slugged[k])
	}
	return buckets, skipped
}

// chunkAtoms splits a bucket's atoms into batches of at most maxAtoms so each
// distill LLM call stays small enough to return complete JSON.
func chunkAtoms(atoms []model.Atom, maxAtoms int) [][]model.Atom {
	if maxAtoms <= 0 || len(atoms) <= maxAtoms {
		return [][]model.Atom{atoms}
	}
	var chunks [][]model.Atom
	for i := 0; i < len(atoms); i += maxAtoms {
		end := i + maxAtoms
		if end > len(atoms) {
			end = len(atoms)
		}
		chunks = append(chunks, atoms[i:end])
	}
	return chunks
}

func serializeAtomsForLLM(atoms []model.Atom) (string, error) {
	inputs := make([]atomLLMInput, 0, len(atoms))
	for _, atom := range atoms {
		inputs = append(inputs, atomLLMInput{
			URI:      uri.BuildAtom(atom.ID.String()),
			Priority: atom.Priority,
			Content:  atom.Content,
		})
	}
	raw, err := json.Marshal(map[string]any{"facts": inputs})
	if err != nil {
		return "", fmt.Errorf("marshal atoms for llm: %w", err)
	}
	return string(raw), nil
}

// serializePartialsForLLM packages partial bodies (from chunked distills) for a
// final merge call when a bucket exceeds the per-call atom budget.
func serializePartialsForLLM(partials []string) (string, error) {
	raw, err := json.Marshal(map[string]any{"facts": partials})
	if err != nil {
		return "", fmt.Errorf("marshal partials for llm: %w", err)
	}
	return string(raw), nil
}

// buildAtomSceneIndex maps each atom key UUID to the scene URIs that cite it,
// used to populate memories.source_scene_uris provenance.
func buildAtomSceneIndex(scenes []model.Scene) map[uuid.UUID][]string {
	index := make(map[uuid.UUID][]string)
	for _, scene := range scenes {
		sceneURI := uri.BuildScene(scene.ID.String())
		for _, atomID := range scene.SourceAtoms {
			index[atomID] = append(index[atomID], sceneURI)
		}
	}
	return index
}

// sourceSceneURIsFor returns the sorted, deduplicated scene URIs that cover a
// bucket's atoms.
func sourceSceneURIsFor(bucket memoryBucket, index map[uuid.UUID][]string) []string {
	seen := make(map[string]struct{})
	for _, atom := range bucket.Atoms {
		for _, sceneURI := range index[atom.ID] {
			seen[sceneURI] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for sceneURI := range seen {
		out = append(out, sceneURI)
	}
	sort.Strings(out)
	return out
}

// equalStringSets reports whether two string slices contain the same set of
// values, ignoring order and duplicates. Used to detect whether a bucket's
// scene provenance changed since its active memory was written.
func equalStringSets(a, b []string) bool {
	seen := make(map[string]struct{}, len(a))
	for _, s := range a {
		seen[s] = struct{}{}
	}
	other := make(map[string]struct{}, len(b))
	for _, s := range b {
		other[s] = struct{}{}
	}
	if len(seen) != len(other) {
		return false
	}
	for s := range seen {
		if _, ok := other[s]; !ok {
			return false
		}
	}
	return true
}
