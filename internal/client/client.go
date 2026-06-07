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
