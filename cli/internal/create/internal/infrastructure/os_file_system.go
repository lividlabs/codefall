// Package infrastructure implements create's gateways against the real process: the disk it creates
// the project on and the git it drives. Both are thin adapters over `internal/shared/process` — what
// they add is create's own terms, because a gateway interface belongs to the use case that declared
// it.
package infrastructure

import (
	"github.com/lividlabs/codefall/cli/internal/shared/process"
)

// OSFileSystem reads and writes the disk through the operating system.
type OSFileSystem struct {
	files *process.FileSystem
}

// NewOSFileSystem builds the real file system gateway.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{files: process.NewFileSystem()}
}

// Exists reports whether anything is at path.
func (f *OSFileSystem) Exists(path string) (bool, error) {
	return f.files.Exists(path)
}

// MkdirAll creates path and every parent it needs.
func (f *OSFileSystem) MkdirAll(path string) error {
	return f.files.MkdirAll(path)
}

// WriteFile writes data to path, replacing whatever was there.
func (f *OSFileSystem) WriteFile(path string, data []byte) error {
	return f.files.WriteFile(path, data)
}
