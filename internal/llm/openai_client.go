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
}

type OpenAICompatibleClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxRetries int
	httpClient *http.Client
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

	return &OpenAICompatibleClient{
		baseURL:    strings.TrimRight(base, "/"),
		apiKey:     key,
		model:      model,
		maxRetries: maxRetries,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
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
				Role: "system",
				Content: "You extract durable facts from agent chat logs. " +
					"Respond with a single JSON object only: {\"atoms\":[...]}. " +
					"Each atom has category (profile|preferences|entities|events), " +
					"priority (int 0-100, use -1 only for critical AI behavior rules), " +
					"scene_name (short label), slug (optional kebab-case, e.g. ai-tone or 2026-05-17-deploy; " +
					"for preferences/entities/events only), " +
					"content (one factual sentence), source_turn_indices (0-based indexes into the batch). " +
					"Do not merge or rewrite prior facts—emit separate atoms. " +
					"events are immutable milestones; never deduplicate them away.",
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
