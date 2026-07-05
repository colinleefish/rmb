package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvValueFromDotenvConf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_URL", "")

	conf := filepath.Join(home, ".rmb.conf")
	if err := os.WriteFile(conf, []byte("RMB_URL=https://example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := EnvValue("RMB_URL"); got != "https://example.test" {
		t.Fatalf("EnvValue(RMB_URL) = %q, want https://example.test", got)
	}
}

func TestEnvValueFromYAMLConf(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_URL", "")
	t.Setenv("RMB_USERNAME", "")

	dir := filepath.Join(home, ".rmb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(dir, "config.yaml")
	body := "client:\n  url: https://yaml.test\nauth:\n  username: cli-user\n  password: secret\n"
	if err := os.WriteFile(yamlPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := EnvValue("RMB_URL"); got != "https://yaml.test" {
		t.Fatalf("EnvValue(RMB_URL) = %q, want https://yaml.test", got)
	}
	if got := EnvValue("RMB_USERNAME"); got != "cli-user" {
		t.Fatalf("EnvValue(RMB_USERNAME) = %q, want cli-user", got)
	}
}

func TestDotenvConfPrecedesYAML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_URL", "")

	if err := os.WriteFile(filepath.Join(home, ".rmb.conf"), []byte("RMB_URL=https://dotenv.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".rmb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("client:\n  url: https://yaml.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := EnvValue("RMB_URL"); got != "https://dotenv.test" {
		t.Fatalf("EnvValue(RMB_URL) = %q, want https://dotenv.test", got)
	}
}

func TestLoadFromYAMLServerConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_DB_URL", "")

	dir := filepath.Join(home, ".rmb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "db:\n  url: postgres://rmb@127.0.0.1:5432/rmb_db?sslmode=disable\nserver:\n  addr: \"127.0.0.1:9090\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DB.URL != "postgres://rmb@127.0.0.1:5432/rmb_db?sslmode=disable" {
		t.Fatalf("DB.URL = %q", cfg.DB.URL)
	}
	if cfg.Server.Addr != "127.0.0.1:9090" {
		t.Fatalf("Server.Addr = %q", cfg.Server.Addr)
	}
}

func TestEnvValueIgnoresRelativeRMBConfig(t *testing.T) {
	home := t.TempDir()
	wd := t.TempDir()
	t.Chdir(wd)
	t.Setenv("HOME", home)
	t.Setenv("RMB_URL", "")
	t.Setenv("RMB_CONFIG", ".env")

	dir := filepath.Join(home, ".rmb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "client:\n  url: https://home.test\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	// Relative RMB_CONFIG must not shadow home config for client recall.
	if got := EnvValue("RMB_URL"); got != "https://home.test" {
		t.Fatalf("EnvValue(RMB_URL) = %q, want https://home.test", got)
	}
}

func TestEnvValueHonorsAbsoluteRMBConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("RMB_URL", "")

	override := filepath.Join(t.TempDir(), "override.conf")
	if err := os.WriteFile(override, []byte("RMB_URL=https://override.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RMB_CONFIG", override)

	dir := filepath.Join(home, ".rmb")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("client:\n  url: https://home.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := EnvValue("RMB_URL"); got != "https://override.test" {
		t.Fatalf("EnvValue(RMB_URL) = %q, want https://override.test", got)
	}
}

func TestLoadDotEnvOverridesEmptyEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("RMB_URL=https://from-file.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RMB_URL", "")

	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("RMB_URL"); got != "https://from-file.test" {
		t.Fatalf("RMB_URL = %q, want https://from-file.test", got)
	}
}
