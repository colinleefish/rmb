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

//go:embed prompts/judge_alias.system.txt
var judgeAliasSystemPrompt string

//go:embed prompts/judge_alias.user.txt
var judgeAliasUserTmpl string

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

func buildDistillMemoryPrompt(category, slug, atomsJSON string, corrections []string) string {
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
	out = strings.ReplaceAll(out, "{{CORRECTIONS}}", buildCorrectionsBlock(corrections))
	out = strings.ReplaceAll(out, "{{FACTS}}", orEmptyMarker(atomsJSON))
	return strings.TrimSpace(out)
}

func buildJudgeAliasPrompt(aURI, aBody, bURI, bBody string) string {
	out := judgeAliasUserTmpl
	out = strings.ReplaceAll(out, "{{A_URI}}", aURI)
	out = strings.ReplaceAll(out, "{{A_BODY}}", orEmptyMarker(aBody))
	out = strings.ReplaceAll(out, "{{B_URI}}", bURI)
	out = strings.ReplaceAll(out, "{{B_BODY}}", orEmptyMarker(bBody))
	return strings.TrimSpace(out)
}

// buildCorrectionsBlock renders human corrections as an authoritative, newest-
// first list that the distiller must treat as ground truth. Empty when none.
func buildCorrectionsBlock(corrections []string) string {
	lines := make([]string, 0, len(corrections))
	for _, c := range corrections {
		if c = strings.TrimSpace(c); c != "" {
			lines = append(lines, "- "+c)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\nAUTHORITATIVE human corrections (highest priority; treat as ground truth and override any conflicting fact below; newest first):\n" +
		strings.Join(lines, "\n")
}
