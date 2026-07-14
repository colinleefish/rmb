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
	"github.com/colinleefish/rmb/internal/service/skill"
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

// Search runs hybrid recall against the remote server. scopes filters which
// tiers are searched ("memory", "scene", "skill"); nil uses the server default
// (memory,scene,skill). k=0 uses the server default (5).
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

// SkillFile is one file in a skill bundle upload.
type SkillFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// SkillSummary is catalog metadata from the server.
type SkillSummary struct {
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// PutSkillResult is returned after uploading a skill bundle.
type PutSkillResult struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
	NoOp    bool   `json:"no_op"`
}

// PutSkill uploads a skill bundle to the remote server.
func (c *Client) PutSkill(ctx context.Context, slug string, files []SkillFile) (PutSkillResult, error) {
	reqBody, err := json.Marshal(map[string]any{"files": files})
	if err != nil {
		return PutSkillResult{}, fmt.Errorf("encode request: %w", err)
	}
	endpoint := c.baseURL + "/api/v1/skills/" + url.PathEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return PutSkillResult{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return PutSkillResult{}, fmt.Errorf("call skills put: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return PutSkillResult{}, apiError("skills put", resp.StatusCode, body)
	}
	var out PutSkillResult
	if err := json.Unmarshal(body, &out); err != nil {
		return PutSkillResult{}, fmt.Errorf("decode response: %w", err)
	}
	return out, nil
}

// ListSkills returns the active skill catalog.
func (c *Client) ListSkills(ctx context.Context) ([]SkillSummary, error) {
	endpoint := c.baseURL + "/api/v1/browse/skills?limit=500"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call skills list: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, apiError("skills list", resp.StatusCode, body)
	}
	var out struct {
		Items []struct {
			URI         string   `json:"uri"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	items := make([]SkillSummary, 0, len(out.Items))
	for _, it := range out.Items {
		items = append(items, SkillSummary{
			URI:         it.URI,
			Name:        it.Name,
			Description: it.Description,
			Tags:        append([]string(nil), it.Tags...),
		})
	}
	return items, nil
}

// GetSkill fetches full skill detail including file contents.
func (c *Client) GetSkill(ctx context.Context, slug string) (skill.Detail, error) {
	endpoint := c.baseURL + "/api/v1/browse/skills/" + url.PathEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return skill.Detail{}, fmt.Errorf("build request: %w", err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return skill.Detail{}, fmt.Errorf("call skills get: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode != http.StatusOK {
		return skill.Detail{}, apiError("skills get", resp.StatusCode, body)
	}
	var out skill.Detail
	if err := json.Unmarshal(body, &out); err != nil {
		return skill.Detail{}, fmt.Errorf("decode response: %w", err)
	}
	return out, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
