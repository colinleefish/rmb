// Package client is the CLI's HTTP client for talking to a remote mypast server.
// It is selected automatically when MYPAST_URL is configured (env or
// ~/.mypast.conf), so the same `mypast find`/`search` commands work against a
// remote service instead of a local database.
package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/colinleefish/mypast/internal/service/recall"
)

// Client calls a remote mypast HTTP API with optional basic auth.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// Resolve returns a Client when MYPAST_URL is explicitly configured (env or
// ~/.mypast.conf), and ok=false otherwise so the caller falls back to local DB
// access. Unlike hook-submit, there is no localhost default: absence of a URL
// means "use the local database".
func Resolve() (*Client, bool) {
	base := confValue("MYPAST_URL")
	if base == "" {
		return nil, false
	}
	user := firstNonEmpty(confValue("MYPAST_USERNAME"), confValue("USERNAME"))
	pass := firstNonEmpty(confValue("MYPAST_PASSWORD"), confValue("PASSWORD"))
	return &Client{
		baseURL:    strings.TrimRight(base, "/"),
		username:   user,
		password:   pass,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, true
}

// BaseURL reports the remote target (for user-facing messages).
func (c *Client) BaseURL() string { return c.baseURL }

func (c *Client) Find(ctx context.Context, query string, k int) ([]recall.Match, error) {
	return c.recall(ctx, "/api/v1/find", query, k)
}

func (c *Client) Search(ctx context.Context, query string, k int) ([]recall.Match, error) {
	return c.recall(ctx, "/api/v1/search", query, k)
}

// CreateAssertion posts a human correction and returns the new assertion URI.
func (c *Client) CreateAssertion(ctx context.Context, kind string, targets []string, statement string) (string, error) {
	reqBody, err := json.Marshal(map[string]any{
		"kind":        kind,
		"target_uris": targets,
		"statement":   statement,
	})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}
	endpoint := c.baseURL + "/api/v1/assertions"
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
		return "", fmt.Errorf("call assertions: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error != "" {
			return "", fmt.Errorf("remote assertions: %s", e.Error)
		}
		return "", fmt.Errorf("remote assertions returned %d", resp.StatusCode)
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
		// Error responses are JSON {"error": "..."}; surface the message.
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &e) == nil && e.Error != "" {
			return "", fmt.Errorf("remote inspect/%s: %s", kind, e.Error)
		}
		return "", fmt.Errorf("remote inspect/%s returned %d", kind, resp.StatusCode)
	}
	return string(body), nil
}

func (c *Client) recall(ctx context.Context, path, query string, k int) ([]recall.Match, error) {
	q := url.Values{}
	q.Set("q", query)
	if k > 0 {
		q.Set("k", strconv.Itoa(k))
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
		return nil, fmt.Errorf("remote %s returned %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out struct {
		Items []recall.Match `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Items, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// confValue reads a key from the environment, falling back to ~/.mypast.conf
// (path overridable via MYPAST_CONF).
func confValue(key string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	confPath := strings.TrimSpace(os.Getenv("MYPAST_CONF"))
	if confPath == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			confPath = filepath.Join(home, ".mypast.conf")
		}
	}
	if confPath == "" {
		return ""
	}
	return readConfValue(confPath, key)
}

func readConfValue(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) != key {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		return v
	}
	return ""
}
