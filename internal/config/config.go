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
	Auth       AuthConfig
	LLM        LLMConfig
	Summarizer SummarizerConfig
	Extraction ExtractionConfig
	Scene      SceneConfig
}

type AuthConfig struct {
	Username string
	Password string
}

func (a AuthConfig) Enabled() bool {
	return strings.TrimSpace(a.Username) != "" || strings.TrimSpace(a.Password) != ""
}

type DBConfig struct {
	URL string
}

type ServerConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type LLMConfig struct {
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
	MaxTurnsPerMerge      int
	MaxCharsPerMerge      int
}

type ExtractionConfig struct {
	Enabled          bool
	PollInterval     time.Duration
	EveryN           int
	IdleSeconds      time.Duration
	Warmup           bool
	BatchSessions    int
	MaxTurnsPerBatch int
	MaxCharsPerBatch int
}

type SceneConfig struct {
	Enabled          bool
	PollInterval     time.Duration
	DelayAfterT1     time.Duration
	BatchSessions    int
	MaxAtomsPerBatch int
}

type fileConfig struct {
	DB struct {
		URL string `toml:"url"`
	} `toml:"db"`
	Server struct {
		Addr            string `toml:"addr"`
		ShutdownTimeout string `toml:"shutdown_timeout"`
	} `toml:"server"`
	LLM struct {
		Provider   string `toml:"provider"`
		APIBase    string `toml:"api_base"`
		APIKey     string `toml:"api_key"`
		Model      string `toml:"model"`
		MaxRetries *int   `toml:"max_retries"`
		Timeout    string `toml:"timeout"`
	} `toml:"llm"`
	Auth struct {
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"auth"`
	Summarizer struct {
		Enabled               *bool  `toml:"enabled"`
		PollInterval          string `toml:"poll_interval"`
		BatchSize             *int   `toml:"batch_size"`
		StaleSummarizingAfter string `toml:"stale_summarizing_after"`
		MaxTurnsPerMerge      *int   `toml:"max_turns_per_merge"`
		MaxCharsPerMerge      *int   `toml:"max_chars_per_merge"`
	} `toml:"summarizer"`
	Extraction struct {
		Enabled          *bool  `toml:"enabled"`
		PollInterval     string `toml:"poll_interval"`
		EveryN           *int   `toml:"every_n"`
		IdleSeconds      string `toml:"idle_seconds"`
		Warmup           *bool  `toml:"warmup"`
		BatchSessions    *int   `toml:"batch_sessions"`
		MaxTurnsPerBatch *int   `toml:"max_turns_per_batch"`
		MaxCharsPerBatch *int   `toml:"max_chars_per_batch"`
	} `toml:"extraction"`
	Scene struct {
		Enabled          *bool  `toml:"enabled"`
		PollInterval     string `toml:"poll_interval"`
		DelayAfterT1     string `toml:"delay_after_t1"`
		BatchSessions    *int   `toml:"batch_sessions"`
		MaxAtomsPerBatch *int   `toml:"max_atoms_per_batch"`
	} `toml:"scene"`
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
		LLM: LLMConfig{
			Provider:   "openai",
			MaxRetries: 2,
			Timeout:    30 * time.Second,
		},
		Summarizer: SummarizerConfig{
			// Disabled by default — the rolling overview_text summarizer is
			// being retired in favour of the T0→T3 atom/scene/memory
			// pipeline (see docs/design-l0-l4.md). Set
			// MYPAST_SUMMARIZER_ENABLED=true to opt back in until the new
			// pipeline lands.
			Enabled:               false,
			PollInterval:          15 * time.Second,
			BatchSize:             8,
			StaleSummarizingAfter: 3 * time.Minute,
			MaxTurnsPerMerge:      4,
			MaxCharsPerMerge:      16000,
		},
		Extraction: ExtractionConfig{
			Enabled:          true,
			PollInterval:     15 * time.Second,
			EveryN:           8,
			IdleSeconds:      10 * time.Minute,
			Warmup:           true,
			BatchSessions:    8,
			MaxTurnsPerBatch: 8,
			MaxCharsPerBatch: 24000,
		},
		Scene: SceneConfig{
			Enabled:          true,
			PollInterval:     15 * time.Second,
			DelayAfterT1:     90 * time.Second,
			BatchSessions:    8,
			MaxAtomsPerBatch: 60,
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

	if cfg.Auth.Enabled() {
		if strings.TrimSpace(cfg.Auth.Username) == "" || strings.TrimSpace(cfg.Auth.Password) == "" {
			return Config{}, fmt.Errorf("auth: both USERNAME and PASSWORD must be set when either is provided")
		}
	}

	return cfg, nil
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
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
	if fc.Auth.Username != "" {
		cfg.Auth.Username = fc.Auth.Username
	}
	if fc.Auth.Password != "" {
		cfg.Auth.Password = fc.Auth.Password
	}
	if fc.LLM.Provider != "" {
		cfg.LLM.Provider = fc.LLM.Provider
	}
	if fc.LLM.APIBase != "" {
		cfg.LLM.APIBase = fc.LLM.APIBase
	}
	if fc.LLM.APIKey != "" {
		cfg.LLM.APIKey = fc.LLM.APIKey
	}
	if fc.LLM.Model != "" {
		cfg.LLM.Model = fc.LLM.Model
	}
	if fc.LLM.MaxRetries != nil {
		cfg.LLM.MaxRetries = *fc.LLM.MaxRetries
	}
	if fc.LLM.Timeout != "" {
		timeout, err := time.ParseDuration(fc.LLM.Timeout)
		if err != nil {
			return fmt.Errorf("parse llm.timeout in %q: %w", path, err)
		}
		cfg.LLM.Timeout = timeout
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
	if fc.Summarizer.MaxTurnsPerMerge != nil {
		cfg.Summarizer.MaxTurnsPerMerge = *fc.Summarizer.MaxTurnsPerMerge
	}
	if fc.Summarizer.MaxCharsPerMerge != nil {
		cfg.Summarizer.MaxCharsPerMerge = *fc.Summarizer.MaxCharsPerMerge
	}
	if fc.Extraction.Enabled != nil {
		cfg.Extraction.Enabled = *fc.Extraction.Enabled
	}
	if fc.Extraction.PollInterval != "" {
		v, err := time.ParseDuration(fc.Extraction.PollInterval)
		if err != nil {
			return fmt.Errorf("parse extraction.poll_interval in %q: %w", path, err)
		}
		cfg.Extraction.PollInterval = v
	}
	if fc.Extraction.EveryN != nil {
		cfg.Extraction.EveryN = *fc.Extraction.EveryN
	}
	if fc.Extraction.IdleSeconds != "" {
		v, err := time.ParseDuration(fc.Extraction.IdleSeconds)
		if err != nil {
			return fmt.Errorf("parse extraction.idle_seconds in %q: %w", path, err)
		}
		cfg.Extraction.IdleSeconds = v
	}
	if fc.Extraction.Warmup != nil {
		cfg.Extraction.Warmup = *fc.Extraction.Warmup
	}
	if fc.Extraction.BatchSessions != nil {
		cfg.Extraction.BatchSessions = *fc.Extraction.BatchSessions
	}
	if fc.Extraction.MaxTurnsPerBatch != nil {
		cfg.Extraction.MaxTurnsPerBatch = *fc.Extraction.MaxTurnsPerBatch
	}
	if fc.Extraction.MaxCharsPerBatch != nil {
		cfg.Extraction.MaxCharsPerBatch = *fc.Extraction.MaxCharsPerBatch
	}
	if fc.Scene.Enabled != nil {
		cfg.Scene.Enabled = *fc.Scene.Enabled
	}
	if fc.Scene.PollInterval != "" {
		v, err := time.ParseDuration(fc.Scene.PollInterval)
		if err != nil {
			return fmt.Errorf("parse scene.poll_interval in %q: %w", path, err)
		}
		cfg.Scene.PollInterval = v
	}
	if fc.Scene.DelayAfterT1 != "" {
		v, err := time.ParseDuration(fc.Scene.DelayAfterT1)
		if err != nil {
			return fmt.Errorf("parse scene.delay_after_t1 in %q: %w", path, err)
		}
		cfg.Scene.DelayAfterT1 = v
	}
	if fc.Scene.BatchSessions != nil {
		cfg.Scene.BatchSessions = *fc.Scene.BatchSessions
	}
	if fc.Scene.MaxAtomsPerBatch != nil {
		cfg.Scene.MaxAtomsPerBatch = *fc.Scene.MaxAtomsPerBatch
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
	if v := firstEnv("MYPAST_USERNAME", "USERNAME"); v != "" {
		cfg.Auth.Username = v
	}
	if v := firstEnv("MYPAST_PASSWORD", "PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("MYPAST_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("MYPAST_LLM_API_BASE"); v != "" {
		cfg.LLM.APIBase = v
	}
	if v := os.Getenv("MYPAST_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("MYPAST_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("MYPAST_LLM_MAX_RETRIES"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_LLM_MAX_RETRIES: %w", err)
		}
		cfg.LLM.MaxRetries = parsed
	}
	if v := os.Getenv("MYPAST_LLM_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_LLM_TIMEOUT: %w", err)
		}
		cfg.LLM.Timeout = d
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
	if v := os.Getenv("MYPAST_SUMMARIZER_MAX_TURNS_PER_MERGE"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_SUMMARIZER_MAX_TURNS_PER_MERGE: %w", err)
		}
		cfg.Summarizer.MaxTurnsPerMerge = parsed
	}
	if v := os.Getenv("MYPAST_SUMMARIZER_MAX_CHARS_PER_MERGE"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_SUMMARIZER_MAX_CHARS_PER_MERGE: %w", err)
		}
		cfg.Summarizer.MaxCharsPerMerge = parsed
	}
	if v := os.Getenv("MYPAST_EXTRACTION_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Extraction.Enabled = true
		case "0", "false", "no", "off":
			cfg.Extraction.Enabled = false
		default:
			return fmt.Errorf("parse MYPAST_EXTRACTION_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("MYPAST_EXTRACTION_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_POLL_INTERVAL: %w", err)
		}
		cfg.Extraction.PollInterval = d
	}
	if v := os.Getenv("MYPAST_EXTRACTION_EVERY_N"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_EVERY_N: %w", err)
		}
		cfg.Extraction.EveryN = parsed
	}
	if v := os.Getenv("MYPAST_EXTRACTION_IDLE_SECONDS"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_IDLE_SECONDS: %w", err)
		}
		cfg.Extraction.IdleSeconds = d
	}
	if v := os.Getenv("MYPAST_EXTRACTION_WARMUP"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Extraction.Warmup = true
		case "0", "false", "no", "off":
			cfg.Extraction.Warmup = false
		default:
			return fmt.Errorf("parse MYPAST_EXTRACTION_WARMUP: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("MYPAST_EXTRACTION_BATCH_SESSIONS"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_BATCH_SESSIONS: %w", err)
		}
		cfg.Extraction.BatchSessions = parsed
	}
	if v := os.Getenv("MYPAST_EXTRACTION_MAX_TURNS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_MAX_TURNS_PER_BATCH: %w", err)
		}
		cfg.Extraction.MaxTurnsPerBatch = parsed
	}
	if v := os.Getenv("MYPAST_EXTRACTION_MAX_CHARS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_EXTRACTION_MAX_CHARS_PER_BATCH: %w", err)
		}
		cfg.Extraction.MaxCharsPerBatch = parsed
	}
	if v := os.Getenv("MYPAST_SCENE_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Scene.Enabled = true
		case "0", "false", "no", "off":
			cfg.Scene.Enabled = false
		default:
			return fmt.Errorf("parse MYPAST_SCENE_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("MYPAST_SCENE_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_SCENE_POLL_INTERVAL: %w", err)
		}
		cfg.Scene.PollInterval = d
	}
	if v := os.Getenv("MYPAST_SCENE_DELAY_AFTER_T1"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse MYPAST_SCENE_DELAY_AFTER_T1: %w", err)
		}
		cfg.Scene.DelayAfterT1 = d
	}
	if v := os.Getenv("MYPAST_SCENE_BATCH_SESSIONS"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_SCENE_BATCH_SESSIONS: %w", err)
		}
		cfg.Scene.BatchSessions = parsed
	}
	if v := os.Getenv("MYPAST_SCENE_MAX_ATOMS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse MYPAST_SCENE_MAX_ATOMS_PER_BATCH: %w", err)
		}
		cfg.Scene.MaxAtomsPerBatch = parsed
	}
	return nil
}
