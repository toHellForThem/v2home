// Package buildinfo exposes version metadata injected at build time.
package buildinfo

import "runtime"

// These values are overridden with -ldflags "-X v2home/internal/buildinfo.Version=...".
var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// GoVersion returns the Go toolchain version used to build the binary.
func GoVersion() string {
	return runtime.Version()
}
