// Package webui serves the embedded Vue single-page application.
package webui

import "embed"

// dist is produced by `make frontend-build` (Vite writes into backend/webui/dist).
// The `all:` prefix makes go:embed include dotfiles such as the .gitkeep
// placeholder that keeps the package buildable before the first frontend build.
//
//go:embed all:dist
var embedded embed.FS
