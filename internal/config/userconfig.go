package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	toml "github.com/pelletier/go-toml/v2"
)

// resolveConfigPath returns the server config file for rmb serve. Precedence:
// RMB_CONFIG, then ~/.rmb.conf, then ~/.rmb/config.yaml.
func resolveConfigPath() (path string, explicit bool) {
	if v := strings.TrimSpace(os.Getenv("RMB_CONFIG")); v != "" {
		return v, true
	}
	if v := strings.TrimSpace(os.Getenv("RMB_CONF")); v != "" {
		return v, true
	}
	return resolveHomeConfigPath()
}

// resolveClientConfigPath returns the recall/client config file. It ignores
// relative RMB_CONFIG/RMB_CONF (e.g. RMB_CONFIG=.env from a project checkout)
// so agents can call search/cat from any cwd with only ~/.rmb/config.yaml.
// Absolute RMB_CONFIG/RMB_CONF still override for tests and power users.
func resolveClientConfigPath() string {
	if v := strings.TrimSpace(os.Getenv("RMB_CONFIG")); v != "" && filepath.IsAbs(v) {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("RMB_CONF")); v != "" && filepath.IsAbs(v) {
		return v
	}
	path, _ := resolveHomeConfigPath()
	return path
}

func resolveHomeConfigPath() (path string, explicit bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	conf := filepath.Join(home, ".rmb.conf")
	if fileExists(conf) {
		return conf, false
	}
	yamlPath := filepath.Join(home, ".rmb", "config.yaml")
	if fileExists(yamlPath) {
		return yamlPath, false
	}
	return "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDotenvConfig(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".toml":
		return false
	default:
		return true
	}
}

// EnvValue reads a client/recall key from a non-empty environment variable,
// then from the client config file (~/.rmb.conf or ~/.rmb/config.yaml).
func EnvValue(key string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	path := resolveClientConfigPath()
	if path == "" {
		return ""
	}
	if isDotenvConfig(path) {
		return readDotenvKey(path, key)
	}
	return structuredEnvLookup(path, key)
}

func readDotenvKey(path, key string) string {
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
		return unquoteEnvValue(strings.TrimSpace(v))
	}
	return ""
}

func unquoteEnvValue(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

func structuredEnvLookup(path, key string) string {
	fc, err := decodeFileConfig(path)
	if err != nil {
		return ""
	}
	switch key {
	case "RMB_URL":
		return strings.TrimSpace(fc.Client.URL)
	case "RMB_USERNAME", "USERNAME":
		return strings.TrimSpace(fc.Auth.Username)
	case "RMB_PASSWORD", "PASSWORD":
		return strings.TrimSpace(fc.Auth.Password)
	default:
		return ""
	}
}

func decodeFileConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, err
	}
	var fc fileConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return fileConfig{}, fmt.Errorf("decode yaml %q: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &fc); err != nil {
			return fileConfig{}, fmt.Errorf("decode toml %q: %w", path, err)
		}
	default:
		return fileConfig{}, fmt.Errorf("unsupported structured config %q", path)
	}
	return fc, nil
}

func loadStructuredFileConfig(cfg *Config, path string, explicitPath bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicitPath {
			return nil
		}
		return fmt.Errorf("read config %q: %w", path, err)
	}

	var fc fileConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return fmt.Errorf("decode config %q: %w", path, err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &fc); err != nil {
			return fmt.Errorf("decode config %q: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported structured config %q", path)
	}
	return applyFileConfig(cfg, path, fc)
}
