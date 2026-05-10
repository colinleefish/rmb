package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	defaultDBURL = "postgres://admin@127.0.0.1:5432/mypast_dev?sslmode=disable"
	defaultAddr  = ":8080"
)

type Config struct {
	DB         DBConfig
	Server     ServerConfig
	VLM        VLMConfig
	Summarizer SummarizerConfig
}

type DBConfig struct {
	URL string
}

type ServerConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type VLMConfig struct {
	Provider   string
	APIBase    string
	APIKey     string
	Model      string
	MaxRetries int
	Timeout    time.Duration
}

type SummarizerConfig struct {
	Enabled               bool
	PollInterval          time.Duration
	BatchSize             int
	StaleSummarizingAfter time.Duration
}

type fileConfig struct {
	DB struct {
		URL string `toml:"url"`
	} `toml:"db"`
	Server struct {
		Addr            string `toml:"addr"`
		ShutdownTimeout string `toml:"shutdown_timeout"`
	} `toml:"server"`
	VLM struct {
		Provider   string `toml:"provider"`
		APIBase    string `toml:"api_base"`
		APIKey     string `toml:"api_key"`
		Model      string `toml:"model"`
		MaxRetries *int   `toml:"max_retries"`
		Timeout    string `toml:"timeout"`
	} `toml:"vlm"`
	Summarizer struct {
		Enabled               *bool  `toml:"enabled"`
		PollInterval          string `toml:"poll_interval"`
		BatchSize             *int   `toml:"batch_size"`
		StaleSummarizingAfter string `toml:"stale_summarizing_after"`
	} `toml:"summarizer"`
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		DB: DBConfig{
			URL: defaultDBURL,
		},
		Server: ServerConfig{
			Addr:            defaultAddr,
			ShutdownTimeout: 5 * time.Second,
		},
		VLM: VLMConfig{
			Provider:   "openai",
			MaxRetries: 2,
			Timeout:    30 * time.Second,
		},
		Summarizer: SummarizerConfig{
			Enabled:               true,
			PollInterval:          15 * time.Second,
			BatchSize:             8,
			StaleSummarizingAfter: 3 * time.Minute,
		},
	}

	configPath, explicitPath := resolveConfigPath()
	if configPath != "" {
		if err := loadFileConfig(&cfg, configPath, explicitPath); err != nil {
			return Config{}, err
		}
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read dotenv %q: %w", path, err)
	}

	lines := strings.Split(string(data), "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("parse dotenv %q line %d: missing '='", path, i+1)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("parse dotenv %q line %d: empty key", path, i+1)
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %q from %q: %w", key, path, err)
		}
	}

	return nil
}

func resolveConfigPath() (string, bool) {
	if v := os.Getenv("MYPAST_CONFIG"); v != "" {
		return v, true
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	return filepath.Join(home, ".config", "mypast", "config.toml"), false
}

func loadFileConfig(cfg *Config, path string, explicitPath bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicitPath {
			return nil
		}
		return fmt.Errorf("read config %q: %w", path, err)
	}

	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("decode config %q: %w", path, err)
	}

	if fc.DB.URL != "" {
		cfg.DB.URL = fc.DB.URL
	}
	if fc.Server.Addr != "" {
		cfg.Server.Addr = fc.Server.Addr
	}
	if fc.Server.ShutdownTimeout != "" {
		timeout, err := time.ParseDuration(fc.Server.ShutdownTimeout)
		if err != nil {
			return fmt.Errorf("parse server.shutdown_timeout in %q: %w", path, err)
		}
		cfg.Server.ShutdownTimeout = timeout
	}
	if fc.VLM.Provider != "" {
		cfg.VLM.Provider = fc.VLM.Provider
	}
	if fc.VLM.APIBase != "" {
		cfg.VLM.APIBase = fc.VLM.APIBase
	}
	if fc.VLM.APIKey != "" {
		cfg.VLM.APIKey = fc.VLM.APIKey
	}
	if fc.VLM.Model != "" {
		cfg.VLM.Model = fc.VLM.Model
	}
	if fc.VLM.MaxRetries != nil {
		cfg.VLM.MaxRetries = *fc.VLM.MaxRetries
	}
	if fc.VLM.Timeout != "" {
		timeout, err := time.ParseDuration(fc.VLM.Timeout)
		if err != nil {
			return fmt.Errorf("parse vlm.timeout in %q: %w", path, err)
		}
		cfg.VLM.Timeout = timeout
	}
	if fc.Summarizer.Enabled != nil {
		cfg.Summarizer.Enabled = *fc.Summarizer.Enabled
	}
	if fc.Summarizer.PollInterval != "" {
		v, err := time.ParseDuration(fc.Summarizer.PollInterval)
		if err != nil {
			return fmt.Errorf("parse summarizer.poll_interval in %q: %w", path, err)
		}
		cfg.Summarizer.PollInterval = v
	}
	if fc.Summarizer.BatchSize != nil {
		cfg.Summarizer.BatchSize = *fc.Summarizer.BatchSize
	}
	if fc.Summarizer.StaleSummarizingAfter != "" {
		v, err := time.ParseDuration(fc.Summarizer.StaleSummarizingAfter)
		if err != nil {
			return fmt.Errorf("parse summarizer.stale_summarizing_after in %q: %w", path, err)
		}
		cfg.Summarizer.StaleSummarizingAfter = v
	}

	return nil
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("MYPAST_DB_URL"); v != "" {
		cfg.DB.URL = v
	}
	if v := os.Getenv("MYPAST_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("MYPAST_SHUTDOWN_TIMEOUT"); v != "" {
		timeout, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.Server.ShutdownTimeout = timeout
	}
	if v := os.Getenv("MYPAST_VLM_PROVIDER"); v != "" {
		cfg.VLM.Provider = v
	}
	if v := os.Getenv("MYPAST_VLM_API_BASE"); v != "" {
		cfg.VLM.APIBase = v
	}
	if v := os.Getenv("MYPAST_VLM_API_KEY"); v != "" {
		cfg.VLM.APIKey = v
	}
	if v := os.Getenv("MYPAST_VLM_MODEL"); v != "" {
		cfg.VLM.Model = v
	}
	if v := os.Getenv("MYPAST_VLM_MAX_RETRIES"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_VLM_MAX_RETRIES: %w", err)
		}
		cfg.VLM.MaxRetries = parsed
	}
	if v := os.Getenv("MYPAST_VLM_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_VLM_TIMEOUT: %w", err)
		}
		cfg.VLM.Timeout = d
	}
	if v := os.Getenv("MYPAST_SUMMARIZER_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Summarizer.Enabled = true
		case "0", "false", "no", "off":
			cfg.Summarizer.Enabled = false
		default:
			return fmt.Errorf("parse MYPAST_SUMMARIZER_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("MYPAST_SUMMARIZER_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_SUMMARIZER_POLL_INTERVAL: %w", err)
		}
		cfg.Summarizer.PollInterval = d
	}
	if v := os.Getenv("MYPAST_SUMMARIZER_BATCH_SIZE"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_SUMMARIZER_BATCH_SIZE: %w", err)
		}
		cfg.Summarizer.BatchSize = parsed
	}
	if v := os.Getenv("MYPAST_SUMMARIZER_STALE_AFTER"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_SUMMARIZER_STALE_AFTER: %w", err)
		}
		cfg.Summarizer.StaleSummarizingAfter = d
	}
	return nil
}
