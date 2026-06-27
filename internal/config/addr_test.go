package config

import "testing"

func TestAddrRequiresAuth(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{":8080", true},
		{"0.0.0.0:8080", true},
		{"127.0.0.1:8080", false},
		{"localhost:8080", false},
		{"[::1]:8080", false},
		{"192.168.1.5:8080", true},
	}
	for _, tc := range tests {
		if got := addrRequiresAuth(tc.addr); got != tc.want {
			t.Fatalf("addrRequiresAuth(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestLoadRejectsPublicBindWithoutAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_ADDR", ":9090")
	t.Setenv("RMB_USERNAME", "")
	t.Setenv("RMB_PASSWORD", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected error binding to :9090 without auth")
	}
}

func TestLoadAllowsLocalhostWithoutAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RMB_CONFIG", "")
	t.Setenv("RMB_ADDR", "127.0.0.1:8080")
	t.Setenv("RMB_USERNAME", "")
	t.Setenv("RMB_PASSWORD", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() = %v", err)
	}
}
