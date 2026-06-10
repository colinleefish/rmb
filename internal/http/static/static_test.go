package static

import (
	"io/fs"
	"testing"
)

// TestEmbeddedUI guards the go:embed contract: the Next.js static export must be
// present in the binary, including the underscore-prefixed _next/ asset dir that
// a plain `//go:embed web/*` would silently drop.
func TestEmbeddedUI(t *testing.T) {
	web, err := fs.Sub(Web, "web")
	if err != nil {
		t.Fatalf("fs.Sub web: %v", err)
	}
	for _, name := range []string{"index.html", "_next", "sessions/index.html"} {
		if _, err := fs.Stat(web, name); err != nil {
			t.Errorf("embedded UI missing %q: %v", name, err)
		}
	}
}
