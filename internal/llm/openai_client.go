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
				Role:    "system",
				Content: buildScenesSystemPrompt,
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
				Role:    "system",
				Content: sessionAbstractSystemPrompt,
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
				Role:    "system",
				Content: distillMemorySystemPrompt,
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
