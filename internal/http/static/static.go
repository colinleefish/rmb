package static

import "embed"

// all: is required so Next.js assets under web/_next/ (underscore-prefixed,
// which go:embed skips by default) are included in the binary.
//go:embed all:web
var Web embed.FS
