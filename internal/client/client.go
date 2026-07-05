// Package client is the CLI's HTTP client for talking to a remote rmb server.
// It is selected automatically when RMB_URL is configured (non-empty env,
// ~/.rmb.conf, or ~/.rmb/config.yaml). Relative RMB_CONFIG from a project
// checkout is ignored so recall works from any cwd.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/colinleefish/rmb/internal/config"
	"github.com/colinleefish/rmb/internal/service/recall"
)

// Client calls a remote rmb HTTP API with optional basic auth.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// Resolve returns a Client when RMB_URL is explicitly configured (env or
// ~/.rmb.conf), and ok=false otherwise so the caller falls back to local DB
// access. Unlike hook-submit, there is no localhost default: absence of a URL
// means "use the local database".
func Resolve() (*Client, bool) {
	base := config.EnvValue("RMB_URL")
	if base == "" {
		return nil, false
	}
	user := firstNonEmpty(config.EnvValue("RMB_USERNAME"), config.EnvValue("USERNAME"))
	pass := firstNonEmpty(config.EnvValue("RMB_PASSWORD"), config.EnvValue("PASSWORD"))
	return &Client{
		baseURL:    strings.TrimRight(base, "/"),
		username:   user,
		password:   pass,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, true
}

func apiError(path string, status int, body []byte) error {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return fmt.Errorf("remote %s: %s", path, e.Error)
	}
	return fmt.Errorf("remote %s returned %d", path, status)
}

// BaseURL reports the remote target (for user-facing messages).
func (c *Client) BaseURL() string { return c.baseURL }

// Backfill triggers a server-side pipeline backfill for the given tier ("t1",
// "t2", or "t3"). When sessionKey is non-empty only that session is enqueued;
// otherwise all eligible sessions are enqueued. Returns the number of sessions
// that were enqueued.
func (c *Client) Backfill(ctx context.Context, tier, sessionKey string) (int, error) {
	endpoint := c.baseURL + "/api/v1/backfill/" + tier
	if sessionKey != "" {
		q := url.Values{}
		q.Set("session", sessionKey)
		endpoint += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("call backfill/%s: %w", tier, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return 0, apiError("backfill/"+tier, resp.StatusCode, body)
	}
	var out struct {
		Enqueued int `json:"enqueued"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return out.Enqueued, nil
}

// EmbedStatusItem is one tier's embedding coverage from the remote server.
type EmbedStatusItem struct {
	Tier     string `json:"tier"`
	Total    int64  `json:"total"`
	Embedded int64  `json:"embedded"`
	Pending  int64  `json:"pending"`
}

// EmbedStatus fetches embedding coverage across atoms, scenes, and memories.
func (c *Client) EmbedStatus(ctx context.Context) ([]EmbedStatusItem, error) {
	endpoint := c.baseURL + "/api/v1/embed/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embed/status: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("embed/status", resp.StatusCode, body)
	}
	var out struct {
		Items []EmbedStatusItem `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Items, nil
}

// Search runs hybrid recall against the remote server. scopes filters which
// tiers are searched ("memory", "scene"); nil uses the server default
// (memory,scene). k=0 uses the server default (5).
func (c *Client) Search(ctx context.Context, query string, k int, scopes []string) ([]recall.Match, error) {
	return c.recall(ctx, "/api/v1/search", query, k, scopes)
}

// CreateCorrection posts a human correction and returns the new correction URI.
func (c *Client) CreateCorrection(ctx context.Context, targets []string, statement string) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"target_uris": targets,
		"statement":   statement,
	})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}
	endpoint := c.baseURL + "/api/v1/corrections"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call corrections: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", apiError("corrections", resp.StatusCode, body)
	}
	var out struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return out.URI, nil
}

// Inspect calls cat/tree/meta on the remote server and returns the textual
// output verbatim (identical to local CLI output). kind is "cat", "tree", or "meta".
func (c *Client) Inspect(ctx context.Context, kind, uri string) (string, error) {
	q := url.Values{}
	q.Set("uri", uri)
	endpoint := c.baseURL + "/api/v1/inspect/" + kind + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call inspect/%s: %w", kind, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return "", apiError("inspect/"+kind, resp.StatusCode, body)
	}
	return string(body), nil
}

func (c *Client) recall(ctx context.Context, path, query string, k int, scopes []string) ([]recall.Match, error) {
	q := url.Values{}
	q.Set("q", query)
	if k > 0 {
		q.Set("k", strconv.Itoa(k))
	}
	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, ","))
	}
	endpoint := c.baseURL + path + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(path, resp.StatusCode, body)
	}

	var out struct {
		Items []recall.Match `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Items, nil
}

// CorrectionItem is a listed correction from the remote server.
type CorrectionItem struct {
	URI        string   `json:"uri"`
	Statement  string   `json:"statement"`
	TargetURIs []string `json:"target_uris"`
}

// ListCorrections returns active corrections; when target is non-empty, only
// those targeting it.
func (c *Client) ListCorrections(ctx context.Context, target string) ([]CorrectionItem, error) {
	endpoint := c.baseURL + "/api/v1/corrections"
	if t := strings.TrimSpace(target); t != "" {
		q := url.Values{}
		q.Set("target", t)
		endpoint += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call corrections list: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("corrections list", resp.StatusCode, body)
	}
	var out struct {
		Items []CorrectionItem `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Items, nil
}

// RetractCorrection retires a correction by its URI on the remote server.
func (c *Client) RetractCorrection(ctx context.Context, correctionURI string) error {
	q := url.Values{}
	q.Set("uri", correctionURI)
	endpoint := c.baseURL + "/api/v1/corrections?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call corrections delete: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return apiError("retract", resp.StatusCode, body)
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
