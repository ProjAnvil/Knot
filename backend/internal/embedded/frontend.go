//go:build embedfrontend

package embedded

import (
	"embed"
	"io/fs"
)

// FrontendFS embeds the frontend dist directory
// This will be populated during build time with //go:embed directive
//
//go:embed all:frontend_dist
var FrontendFS embed.FS

// GetFrontendFS returns the embedded filesystem with the dist directory stripped
func GetFrontendFS() (fs.FS, error) {
	return fs.Sub(FrontendFS, "frontend_dist")
}

// HasFrontend checks if frontend files are embedded
func HasFrontend() bool {
	entries, err := FrontendFS.ReadDir("frontend_dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
