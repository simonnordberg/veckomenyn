package web

import "embed"

// DistFS contains the built frontend (Vite output) embedded at compile time.
// When running without a built frontend, only .gitkeep is present and the
// server falls back to a placeholder page.
//
//go:embed all:dist
var DistFS embed.FS
