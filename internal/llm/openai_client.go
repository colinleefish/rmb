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

	var lastErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		merged, retryable, err := c.chatCompletion(ctx, req)
		if err == nil {
			return strings.TrimSpace(merged), nil
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

	return "", fmt.Errorf("llm merge overview failed: %w", lastErr)
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
