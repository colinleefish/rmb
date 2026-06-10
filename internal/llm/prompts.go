package llm

import (
	_ "embed"
	"strings"
)

// Prompt text lives in prompts/*.txt and is embedded at build time. This keeps
// the single self-contained binary (and the prompt version tied to the deployed
// commit for auditability) while making prompts easy to read and edit without
// Go string escaping. User templates use {{PLACEHOLDER}} tokens.

//go:embed prompts/extract_atoms.system.txt
var extractAtomsSystemPrompt string

//go:embed prompts/extract_atoms.user.txt
var extractAtomsUserTmpl string

//go:embed prompts/build_scenes.system.txt
var buildScenesSystemPrompt string

//go:embed prompts/build_scenes.user.txt
var buildScenesUserTmpl string

//go:embed prompts/session_abstract.system.txt
var sessionAbstractSystemPrompt string

//go:embed prompts/session_abstract.user.txt
var sessionAbstractUserTmpl string

//go:embed prompts/distill_memory.system.txt
var distillMemorySystemPrompt string

//go:embed prompts/distill_memory.user.txt
var distillMemoryUserTmpl string

//go:embed prompts/distill_memory.profile_filter.txt
var distillMemoryProfileFilter string

func orEmptyMarker(s string) string {
	if s = strings.TrimSpace(s); s != "" {
		return s
	}
	return "(empty)"
}

func buildExtractAtomsPrompt(messagesJSONL string) string {
	out := strings.ReplaceAll(extractAtomsUserTmpl, "{{BATCH}}", orEmptyMarker(messagesJSONL))
	return strings.TrimSpace(out)
}

func buildBuildScenesPrompt(atomsJSON string) string {
	out := strings.ReplaceAll(buildScenesUserTmpl, "{{ATOMS}}", orEmptyMarker(atomsJSON))
	return strings.TrimSpace(out)
}

func buildSessionAbstractPrompt(sceneAbstracts string) string {
	out := strings.ReplaceAll(sessionAbstractUserTmpl, "{{ABSTRACTS}}", orEmptyMarker(sceneAbstracts))
	return strings.TrimSpace(out)
}

func buildDistillMemoryPrompt(category, slug, atomsJSON string) string {
	topic := strings.TrimSpace(slug)
	if topic == "" {
		topic = "(none)"
	}
	// Defense-in-depth: clean the profile singleton even if upstream atoms are
	// mislabeled. The distiller only sees atoms already routed to this category,
	// so it can omit noise from the body (it cannot re-route atoms).
	filter := ""
	if category == "profile" {
		filter = "\n" + strings.TrimSpace(distillMemoryProfileFilter)
	}
	out := distillMemoryUserTmpl
	out = strings.ReplaceAll(out, "{{CATEGORY}}", category)
	out = strings.ReplaceAll(out, "{{SLUG}}", topic)
	out = strings.ReplaceAll(out, "{{FILTER}}", filter)
	out = strings.ReplaceAll(out, "{{FACTS}}", orEmptyMarker(atomsJSON))
	return strings.TrimSpace(out)
}
