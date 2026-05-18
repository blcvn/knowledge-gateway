// Package ui provides the embedded frontend assets for the VNP Memory Console.
//
// The ui_dist/ directory is populated at build time by copying ui/dist/ into
// apps/memory/ui_dist/. Use the Makefile target `make ui-embed` or `make memory-full`
// to generate it automatically before building the Go binary.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed all:ui_dist
var distDir embed.FS

// DistFS returns a filesystem rooted at the ui_dist directory,
// ready to be served by net/http.FileServer.
func DistFS() (fs.FS, error) {
	return fs.Sub(distDir, "ui_dist")
}
