// Package buildinfo holds linker-injected build metadata.
package buildinfo

// Commit is the git short SHA at build time (ldflags -X).
var Commit = "unknown"
