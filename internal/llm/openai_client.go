package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultMaxRetries = 2
	defaultTimeout    = 30 * time.Second
	maxErrorBodyBytes = 2048
)

type OpenAICompatibleConfig struct {
	Provider   string
	APIBase    string
	APIKey     string
	Model      string
	MaxRetries int
	Timeout    time.Duration
	// ThinkingStyle selects how the provider encodes the reasoning toggle:
	// "thinking_type" (MiMo/DeepSeek), "enable_thinking" (Qwen/gateways),
	// "reasoning_effort" (OpenAI-style), or "" to send nothing.
	ThinkingStyle string
	// ThinkingEnabled is the desired reasoning state when ThinkingStyle is set.
	ThinkingEnabled bool
}

type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxRetries int
	httpClient *http.Client
	// thinkingBody is merged into every chat request to toggle reasoning; nil
	// when no thinking style is configured (provider default).
	thinkingBody map[string]any
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func NewOpenAICompatibleClient(cfg OpenAICompatibleConfig) (*OpenAICompatibleClient, error) {
	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" {
		provider = "openai"
	}
	if provider != "openai" {
		return nil, fmt.Errorf("unsupported llm provider: %s", provider)
	}

	base := strings.TrimSpace(cfg.APIBase)
	key := strings.TrimSpace(cfg.APIKey)
	model := strings.TrimSpace(cfg.Model)
	if base == "" {
		return nil, errors.New("llm api_base is required")
	}
	if key == "" {
		return nil, errors.New("llm api_key is required")
	}
	if model == "" {
		return nil, errors.New("llm model is required")
	}

	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	thinkingBody, err := buildThinkingBody(cfg.ThinkingStyle, cfg.ThinkingEnabled)
	if err != nil {
		return nil, err
	}

	return &OpenAICompatibleClient{
		baseURL:      strings.TrimRight(base, "/"),
		apiKey:       key,
		model:        model,
		maxRetries:   maxRetries,
		httpClient:   &http.Client{Timeout: timeout},
		thinkingBody: thinkingBody,
	}, nil
}

// Thinking-toggle encodings, keyed by provider style. Each provider exposes the
// reasoning switch differently; this maps a style + desired state to the request
// fields to merge. Returns nil (omit) when the style is empty or the style has
// no representation for the requested state.
const (
	ThinkingStyleNone            = ""
	ThinkingStyleThinkingType    = "thinking_type"    // {"thinking":{"type":"disabled"|"enabled"}}
	ThinkingStyleEnableThinking  = "enable_thinking"  // {"enable_thinking": false|true}
	ThinkingStyleReasoningEffort = "reasoning_effort" // {"reasoning_effort":"none"} to disable
)

func buildThinkingBody(style string, enabled bool) (map[string]any, error) {
	switch strings.TrimSpace(style) {
	case ThinkingStyleNone:
		return nil, nil
	case ThinkingStyleThinkingType:
		state := "enabled"
		if !enabled {
			state = "disabled"
		}
		return map[string]any{"thinking": map[string]any{"type": state}}, nil
	case ThinkingStyleEnableThinking:
		return map[string]any{"enable_thinking": enabled}, nil
	case ThinkingStyleReasoningEffort:
		if enabled {
			// No single "enabled" value; leave the provider default.
			return nil, nil
		}
		return map[string]any{"reasoning_effort": "none"}, nil
	default:
		return nil, fmt.Errorf("unknown llm thinking style %q", style)
	}
}

func (c *OpenAICompatibleClient) MergeOverview(
	ctx context.Context,
	previousOverview string,
	messagesJSONL string,
) (string, error) {
	userPrompt := buildMergeOverviewPrompt(previousOverview, messagesJSONL)
	req := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You summarize chat history. Return plain text only. " +
					"Keep key facts, goals, constraints, decisions, and unresolved tasks concise.",
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	merged, err := c.completeWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm merge overview failed: %w", err)
	}
	return strings.TrimSpace(merged), nil
}

// extractAtomsSystemPrompt instructs the T1 extractor with explicit category
// routing, a skip-list for transient noise, and few-shot examples. This keeps
// the profile singleton to durable first-person facts and routes preferences /
// third parties to their own categories instead of dumping them into profile.
const extractAtomsSystemPrompt = `You extract durable, long-term facts from agent chat logs into memory atoms.
Respond with a single JSON object only: {"atoms":[...]}.

Each atom has:
- category: one of profile | preferences | entities | events
- priority: int 0-100 (use -1 only for critical AI behavior rules)
- scene_name: short label grouping related atoms
- slug: kebab-case topic id. REQUIRED for preferences/entities/events; OMIT for profile
- content: one factual sentence
- source_turn_indices: 0-based indexes into the batch

Category routing (choose exactly one per fact):
- profile: DURABLE FIRST-PERSON facts about THE USER only — identity, location, role, devices, stable traits, health/taboos. Singleton; no slug.
- preferences: recurring "prefers / wants / always / never", INCLUDING rules for how the AI should behave (tone, format, workflow). One slug per topic.
- entities: any third party that is NOT the user — other people, teams, companies, projects, hosts, tools, services. One slug per entity.
- events: dated decisions, milestones, or actions taken. Immutable. One slug, date-prefixed when a date is known.

Do NOT extract (skip entirely):
- Transient session state or the AI's current mode (e.g. "AI is in Ask mode", "switched to Agent mode").
- The user's momentary emotional reactions (e.g. "user was surprised/alarmed").
- Tool/skill attachment or other UI actions within this session.
- Anything about the assistant's behavior in THIS session that is not a durable user preference.

Hard rules:
- profile is ONLY about the user. Put any third party in entities, NEVER in profile.
- "prefers / wants / always / never" goes to preferences (with a slug), NOT profile.
- Do not merge or rewrite prior facts — emit separate atoms.
- events are immutable milestones; never deduplicate them away.

Examples:
- "I live in Beijing" -> {"category":"profile","content":"The user lives in Beijing.","scene_name":"identity","source_turn_indices":[0]}
- "I prefer short answers" -> {"category":"preferences","slug":"answer-length","content":"The user prefers short answers.","scene_name":"ai-behavior","source_turn_indices":[0]}
- "Always use Go for backend services" -> {"category":"preferences","slug":"go-services","content":"The user prefers Go for backend services.","scene_name":"tech-stack","source_turn_indices":[0]}
- "姚乾坤 is an R&D engineer" -> {"category":"entities","slug":"yao-qiankun","content":"姚乾坤 is an R&D engineer.","scene_name":"people","source_turn_indices":[0]}
- "We chose Postgres-only storage on 2026-05-17" -> {"category":"events","slug":"2026-05-17-postgres-only","content":"On 2026-05-17 the team chose Postgres-only storage.","scene_name":"decisions","source_turn_indices":[0]}
- "The AI is currently in Ask mode" -> (skip: transient session state)`

// ExtractAtoms asks the model to return JSON atoms for a batch of session turns.
func (c *OpenAICompatibleClient) ExtractAtoms(ctx context.Context, messagesJSONL string) (string, error) {
	req := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.1,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: extractAtomsSystemPrompt,
			},
			{
				Role:    "user",
				Content: buildExtractAtomsPrompt(messagesJSONL),
			},
		},
	}
	out, err := c.completeWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm extract atoms failed: %w", err)
	}
	return out, nil
}

// BuildScenes asks the model to aggregate atoms into scene summaries.
func (c *OpenAICompatibleClient) BuildScenes(ctx context.Context, atomsJSON string) (string, error) {
	req := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.1,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You aggregate structured memory atoms into scene summaries. " +
					"Respond with a single JSON object only: {\"scenes\":[...]}. " +
					"Each scene has display_name (short label), abstract (~100 tokens, plain text), " +
					"body (Markdown, factual, only from input atoms), atom_uris (subset of input URIs). " +
					"Emit one scene per distinct scene_name group in the input. " +
					"Do not invent facts or URIs not present in the input.",
			},
			{
				Role:    "user",
				Content: buildBuildScenesPrompt(atomsJSON),
			},
		},
	}
	out, err := c.completeWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm build scenes failed: %w", err)
	}
	return out, nil
}

// SummarizeSessionAbstract derives a short session abstract from scene abstracts.
func (c *OpenAICompatibleClient) SummarizeSessionAbstract(
	ctx context.Context,
	sceneAbstracts string,
) (string, error) {
	req := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0.2,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You write concise session summaries. Return plain text only, " +
					"at most 100 tokens. Preserve key facts from the scene abstracts.",
			},
			{
				Role:    "user",
				Content: buildSessionAbstractPrompt(sceneAbstracts),
			},
		},
	}
	out, err := c.completeWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm summarize session abstract failed: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// DistillMemory rolls a category/slug bucket of atoms into a long-term memory.
// Temperature 0 keeps output stable across rollups so the skip-if-unchanged
// guard fires for unchanged inputs (avoids stacking near-duplicate versions).
func (c *OpenAICompatibleClient) DistillMemory(
	ctx context.Context,
	category string,
	slug string,
	atomsJSON string,
) (string, error) {
	req := chatCompletionRequest{
		Model:       c.model,
		Temperature: 0,
		Messages: []chatMessage{
			{
				Role: "system",
				Content: "You distill durable long-term memory from structured facts. " +
					"Respond with a single JSON object only: {\"abstract\":\"...\",\"body\":\"...\"}. " +
					"abstract is one line (~100 tokens) for retrieval; body is factual Markdown. " +
					"Use only the supplied facts; do not invent. Resolve contradictions in favour of the " +
					"most recent facts. Keep stable identity/preferences/entity details; for events, list " +
					"them as an immutable dated record.",
			},
			{
				Role:    "user",
				Content: buildDistillMemoryPrompt(category, slug, atomsJSON),
			},
		},
	}
	out, err := c.completeWithRetry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm distill memory failed: %w", err)
	}
	return out, nil
}

func buildDistillMemoryPrompt(category, slug, atomsJSON string) string {
	chunk := strings.TrimSpace(atomsJSON)
	if chunk == "" {
		chunk = "(empty)"
	}
	topic := strings.TrimSpace(slug)
	if topic == "" {
		topic = "(none)"
	}
	// Defense-in-depth: clean the profile singleton even if upstream atoms are
	// mislabeled. The distiller only sees atoms already routed to this category,
	// so it can omit noise from the body (it cannot re-route atoms).
	var filter string
	if category == "profile" {
		filter = `
Keep ONLY durable first-person facts about the user (identity, location, role, devices, stable traits, health/taboos).
DROP any fact that is about a third party, transient session/AI-mode state, or a momentary emotional reaction.`
	}
	return strings.TrimSpace(`Distill these facts into a single long-term memory.

category: ` + category + `
topic/slug: ` + topic + filter + `

Return JSON only:
{"abstract":"...","body":"..."}

Facts (JSON):
` + chunk)
}

func buildBuildScenesPrompt(atomsJSON string) string {
	chunk := strings.TrimSpace(atomsJSON)
	if chunk == "" {
		chunk = "(empty)"
	}
	return strings.TrimSpace(`Aggregate these memory atoms into scene summaries.

Return JSON only:
{"scenes":[{"display_name":"...","abstract":"...","body":"...","atom_uris":["mypast://..."]}]}

Input atoms (JSON):
` + chunk)
}

func buildSessionAbstractPrompt(sceneAbstracts string) string {
	chunk := strings.TrimSpace(sceneAbstracts)
	if chunk == "" {
		chunk = "(empty)"
	}
	return strings.TrimSpace(`Write a single session abstract from these scene abstracts.

Scene abstracts:
` + chunk)
}

func (c *OpenAICompatibleClient) completeWithRetry(
	ctx context.Context,
	req chatCompletionRequest,
) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		content, retryable, err := c.chatCompletion(ctx, req)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !retryable || attempt == c.maxRetries {
			break
		}
		backoff := time.Duration(attempt*attempt) * 300 * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}
	return "", lastErr
}

func buildExtractAtomsPrompt(messagesJSONL string) string {
	chunk := strings.TrimSpace(messagesJSONL)
	if chunk == "" {
		chunk = "(empty)"
	}
	return strings.TrimSpace(`Extract structured memory atoms from this chat batch (JSONL, one message per line).

Return JSON only:
{"atoms":[{"category":"...","priority":50,"scene_name":"...","slug":"ai-tone","content":"...","source_turn_indices":[0]}]}

Chat batch:
` + chunk)
}

func buildMergeOverviewPrompt(previousOverview string, messagesJSONL string) string {
	prev := strings.TrimSpace(previousOverview)
	if prev == "" {
		prev = "(empty)"
	}
	current := strings.TrimSpace(messagesJSONL)
	if current == "" {
		current = "(empty)"
	}

	return strings.TrimSpace(`
Task:
You are given the previous session overview and one new chat chunk.
Generate the NEW full session overview in plain text.

Rules:
- Keep it concise and factual.
- Preserve still-valid context from previous overview.
- Add new decisions, constraints, tasks, and user preferences from the new chunk.
- Remove contradictions by preferring the latest chunk.
- Do not output JSON or markdown headings.

Previous overview:
` + prev + `

New chunk (JSONL):
` + current)
}

func (c *OpenAICompatibleClient) chatCompletion(
	ctx context.Context,
	reqBody chatCompletionRequest,
) (string, bool, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return "", false, fmt.Errorf("marshal llm request: %w", err)
	}

	// Merge the provider-specific thinking toggle into the request body.
	if len(c.thinkingBody) > 0 {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", false, fmt.Errorf("merge thinking body: %w", err)
		}
		for k, v := range c.thinkingBody {
			m[k] = v
		}
		raw, err = json.Marshal(m)
		if err != nil {
			return "", false, fmt.Errorf("marshal llm request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(raw),
	)
	if err != nil {
		return "", false, fmt.Errorf("build llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("perform llm request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return "", retryable, fmt.Errorf(
			"llm http %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return "", false, fmt.Errorf("decode llm response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return "", false, errors.New("llm response has no choices")
	}

	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" {
		return "", false, errors.New("llm response is empty")
	}
	return content, false, nil
}
