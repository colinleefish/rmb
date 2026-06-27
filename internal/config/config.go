package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultDBURL = "postgres://admin@127.0.0.1:5432/rmb_db?sslmode=disable"
	defaultAddr  = ":8080"
)

type Config struct {
	DB           DBConfig
	Server       ServerConfig
	Auth         AuthConfig
	LLM          LLMConfig
	Extraction   ExtractionConfig
	Scene        SceneConfig
	Memory       MemoryConfig
	Embed        EmbedConfig
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
	Provider        string
	APIBase         string
	APIKey          string
	Model           string
	MaxRetries      int
	Timeout         time.Duration
	ThinkingStyle   string // "" | thinking_type | enable_thinking | reasoning_effort
	ThinkingEnabled bool
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

type MemoryConfig struct {
	Enabled          bool
	PollInterval     time.Duration
	MaxAtomsPerBatch int
}

type EmbedConfig struct {
	Enabled      bool
	APIBase      string
	APIKey       string
	Model        string
	Dimensions   int
	PollInterval time.Duration
	BatchSize    int
	Timeout      time.Duration
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
		Provider      string `toml:"provider"`
		APIBase       string `toml:"api_base"`
		APIKey        string `toml:"api_key"`
		Model         string `toml:"model"`
		MaxRetries    *int   `toml:"max_retries"`
		Timeout       string `toml:"timeout"`
		ThinkingStyle string `toml:"thinking_style"`
		Thinking      string `toml:"thinking"`
	} `toml:"llm"`
	Auth struct {
		Username string `toml:"username"`
		Password string `toml:"password"`
	} `toml:"auth"`
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
	Memory struct {
		Enabled          *bool  `toml:"enabled"`
		PollInterval     string `toml:"poll_interval"`
		MaxAtomsPerBatch *int   `toml:"max_atoms_per_batch"`
	} `toml:"memory"`
	Embed struct {
		Enabled      *bool  `toml:"enabled"`
		APIBase      string `toml:"api_base"`
		APIKey       string `toml:"api_key"`
		Model        string `toml:"model"`
		Dimensions   *int   `toml:"dimensions"`
		PollInterval string `toml:"poll_interval"`
		BatchSize    *int   `toml:"batch_size"`
		Timeout      string `toml:"timeout"`
	} `toml:"embed"`
	Client struct {
		URL string `toml:"url" yaml:"url"`
	} `toml:"client" yaml:"client"`
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
			Provider:        "openai",
			MaxRetries:      2,
			Timeout:         30 * time.Second,
			ThinkingStyle:   "",
			ThinkingEnabled: true,
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
		Memory: MemoryConfig{
			Enabled:          true,
			PollInterval:     5 * time.Minute,
			MaxAtomsPerBatch: 60,
		},
		Embed: EmbedConfig{
			Enabled:      true,
			APIBase:      "https://open.bigmodel.cn/api/paas/v4",
			Model:        "embedding-3",
			Dimensions:   1024,
			PollInterval: 30 * time.Second,
			BatchSize:    32,
			Timeout:      60 * time.Second,
		},
	}

	configPath, explicitPath := resolveConfigPath()
	if configPath != "" {
		if isDotenvConfig(configPath) {
			if err := loadDotEnv(configPath); err != nil {
				return Config{}, err
			}
		} else if err := loadStructuredFileConfig(&cfg, configPath, explicitPath); err != nil {
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

	switch cfg.LLM.ThinkingStyle {
	case "", "thinking_type", "enable_thinking", "reasoning_effort":
	default:
		return Config{}, fmt.Errorf("llm thinking_style %q invalid (want thinking_type|enable_thinking|reasoning_effort)", cfg.LLM.ThinkingStyle)
	}

	return cfg, nil
}

// parseThinking maps an enabled/disabled string to a bool.
func parseThinking(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled", "enable":
		return true, nil
	case "0", "false", "no", "off", "disabled", "disable":
		return false, nil
	default:
		return false, fmt.Errorf("invalid thinking value %q (want enabled|disabled)", v)
	}
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

func applyFileConfig(cfg *Config, path string, fc fileConfig) error {
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
	if fc.LLM.ThinkingStyle != "" {
		cfg.LLM.ThinkingStyle = fc.LLM.ThinkingStyle
	}
	if fc.LLM.Thinking != "" {
		enabled, err := parseThinking(fc.LLM.Thinking)
		if err != nil {
			return fmt.Errorf("parse llm.thinking in %q: %w", path, err)
		}
		cfg.LLM.ThinkingEnabled = enabled
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
	if fc.Memory.Enabled != nil {
		cfg.Memory.Enabled = *fc.Memory.Enabled
	}
	if fc.Memory.PollInterval != "" {
		v, err := time.ParseDuration(fc.Memory.PollInterval)
		if err != nil {
			return fmt.Errorf("parse memory.poll_interval in %q: %w", path, err)
		}
		cfg.Memory.PollInterval = v
	}
	if fc.Memory.MaxAtomsPerBatch != nil {
		cfg.Memory.MaxAtomsPerBatch = *fc.Memory.MaxAtomsPerBatch
	}
	if fc.Embed.Enabled != nil {
		cfg.Embed.Enabled = *fc.Embed.Enabled
	}
	if fc.Embed.APIBase != "" {
		cfg.Embed.APIBase = fc.Embed.APIBase
	}
	if fc.Embed.APIKey != "" {
		cfg.Embed.APIKey = fc.Embed.APIKey
	}
	if fc.Embed.Model != "" {
		cfg.Embed.Model = fc.Embed.Model
	}
	if fc.Embed.Dimensions != nil {
		cfg.Embed.Dimensions = *fc.Embed.Dimensions
	}
	if fc.Embed.PollInterval != "" {
		v, err := time.ParseDuration(fc.Embed.PollInterval)
		if err != nil {
			return fmt.Errorf("parse embed.poll_interval in %q: %w", path, err)
		}
		cfg.Embed.PollInterval = v
	}
	if fc.Embed.BatchSize != nil {
		cfg.Embed.BatchSize = *fc.Embed.BatchSize
	}
	if fc.Embed.Timeout != "" {
		v, err := time.ParseDuration(fc.Embed.Timeout)
		if err != nil {
			return fmt.Errorf("parse embed.timeout in %q: %w", path, err)
		}
		cfg.Embed.Timeout = v
	}
	if fc.Client.URL != "" {
		if err := os.Setenv("RMB_URL", fc.Client.URL); err != nil {
			return fmt.Errorf("set RMB_URL from %q: %w", path, err)
		}
	}

	return nil
}

func applyEnvOverrides(cfg *Config) error {
	if v := os.Getenv("RMB_DB_URL"); v != "" {
		cfg.DB.URL = v
	}
	if v := os.Getenv("RMB_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("RMB_SHUTDOWN_TIMEOUT"); v != "" {
		timeout, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.Server.ShutdownTimeout = timeout
	}
	if v := firstEnv("RMB_USERNAME", "USERNAME"); v != "" {
		cfg.Auth.Username = v
	}
	if v := firstEnv("RMB_PASSWORD", "PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("RMB_LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("RMB_LLM_API_BASE"); v != "" {
		cfg.LLM.APIBase = v
	}
	if v := os.Getenv("RMB_LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("RMB_LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("RMB_LLM_MAX_RETRIES"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_LLM_MAX_RETRIES: %w", err)
		}
		cfg.LLM.MaxRetries = parsed
	}
	if v := os.Getenv("RMB_LLM_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_LLM_TIMEOUT: %w", err)
		}
		cfg.LLM.Timeout = d
	}
	if v := os.Getenv("RMB_LLM_THINKING_STYLE"); v != "" {
		cfg.LLM.ThinkingStyle = strings.TrimSpace(v)
	}
	if v := os.Getenv("RMB_LLM_THINKING"); v != "" {
		enabled, err := parseThinking(v)
		if err != nil {
			return fmt.Errorf("parse RMB_LLM_THINKING: %w", err)
		}
		cfg.LLM.ThinkingEnabled = enabled
	}
	if v := os.Getenv("RMB_EXTRACTION_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Extraction.Enabled = true
		case "0", "false", "no", "off":
			cfg.Extraction.Enabled = false
		default:
			return fmt.Errorf("parse RMB_EXTRACTION_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("RMB_EXTRACTION_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_POLL_INTERVAL: %w", err)
		}
		cfg.Extraction.PollInterval = d
	}
	if v := os.Getenv("RMB_EXTRACTION_EVERY_N"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_EVERY_N: %w", err)
		}
		cfg.Extraction.EveryN = parsed
	}
	if v := os.Getenv("RMB_EXTRACTION_IDLE_SECONDS"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_IDLE_SECONDS: %w", err)
		}
		cfg.Extraction.IdleSeconds = d
	}
	if v := os.Getenv("RMB_EXTRACTION_WARMUP"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Extraction.Warmup = true
		case "0", "false", "no", "off":
			cfg.Extraction.Warmup = false
		default:
			return fmt.Errorf("parse RMB_EXTRACTION_WARMUP: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("RMB_EXTRACTION_BATCH_SESSIONS"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_BATCH_SESSIONS: %w", err)
		}
		cfg.Extraction.BatchSessions = parsed
	}
	if v := os.Getenv("RMB_EXTRACTION_MAX_TURNS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_MAX_TURNS_PER_BATCH: %w", err)
		}
		cfg.Extraction.MaxTurnsPerBatch = parsed
	}
	if v := os.Getenv("RMB_EXTRACTION_MAX_CHARS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EXTRACTION_MAX_CHARS_PER_BATCH: %w", err)
		}
		cfg.Extraction.MaxCharsPerBatch = parsed
	}
	if v := os.Getenv("RMB_SCENE_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Scene.Enabled = true
		case "0", "false", "no", "off":
			cfg.Scene.Enabled = false
		default:
			return fmt.Errorf("parse RMB_SCENE_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("RMB_SCENE_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_SCENE_POLL_INTERVAL: %w", err)
		}
		cfg.Scene.PollInterval = d
	}
	if v := os.Getenv("RMB_SCENE_DELAY_AFTER_T1"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_SCENE_DELAY_AFTER_T1: %w", err)
		}
		cfg.Scene.DelayAfterT1 = d
	}
	if v := os.Getenv("RMB_SCENE_BATCH_SESSIONS"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_SCENE_BATCH_SESSIONS: %w", err)
		}
		cfg.Scene.BatchSessions = parsed
	}
	if v := os.Getenv("RMB_SCENE_MAX_ATOMS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_SCENE_MAX_ATOMS_PER_BATCH: %w", err)
		}
		cfg.Scene.MaxAtomsPerBatch = parsed
	}
	if v := os.Getenv("RMB_MEMORY_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Memory.Enabled = true
		case "0", "false", "no", "off":
			cfg.Memory.Enabled = false
		default:
			return fmt.Errorf("parse RMB_MEMORY_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("RMB_MEMORY_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_MEMORY_POLL_INTERVAL: %w", err)
		}
		cfg.Memory.PollInterval = d
	}
	if v := os.Getenv("RMB_MEMORY_MAX_ATOMS_PER_BATCH"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_MEMORY_MAX_ATOMS_PER_BATCH: %w", err)
		}
		cfg.Memory.MaxAtomsPerBatch = parsed
	}
	if v := os.Getenv("RMB_EMBED_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			cfg.Embed.Enabled = true
		case "0", "false", "no", "off":
			cfg.Embed.Enabled = false
		default:
			return fmt.Errorf("parse RMB_EMBED_ENABLED: invalid boolean %q", v)
		}
	}
	if v := os.Getenv("RMB_EMBED_API_BASE"); v != "" {
		cfg.Embed.APIBase = v
	}
	if v := os.Getenv("RMB_EMBED_API_KEY"); v != "" {
		cfg.Embed.APIKey = v
	}
	if v := os.Getenv("RMB_EMBED_MODEL"); v != "" {
		cfg.Embed.Model = v
	}
	if v := os.Getenv("RMB_EMBED_DIMENSIONS"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EMBED_DIMENSIONS: %w", err)
		}
		cfg.Embed.Dimensions = parsed
	}
	if v := os.Getenv("RMB_EMBED_POLL_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_EMBED_POLL_INTERVAL: %w", err)
		}
		cfg.Embed.PollInterval = d
	}
	if v := os.Getenv("RMB_EMBED_BATCH_SIZE"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse RMB_EMBED_BATCH_SIZE: %w", err)
		}
		cfg.Embed.BatchSize = parsed
	}
	if v := os.Getenv("RMB_EMBED_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parse RMB_EMBED_TIMEOUT: %w", err)
		}
		cfg.Embed.Timeout = d
	}
	return nil
}
