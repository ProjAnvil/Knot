//go:build !embedfrontend

package embedded

import (
	"errors"
	"io/fs"
)

// GetFrontendFS returns an error when frontend is not embedded
func GetFrontendFS() (fs.FS, error) {
	return nil, errors.New("frontend not embedded")
}

// HasFrontend returns false when frontend is not embedded
func HasFrontend() bool {
	return false
}
