// Package infrastructure implements initcmd's gateways against the real process: the file system it
// writes into and the external tools codefall depends on. Both are thin adapters over
// `internal/shared/process` — what they add is initcmd's own terms, because a gateway interface
// belongs to the use case that declared it. The permissions a write uses are the shared module's
// decision, not this one's and not the use case's.
package infrastructure

import (
	"github.com/lividlabs/codefall/cli/internal/shared/process"
)

// OSFileSystem reads and writes the working directory through the operating system.
type OSFileSystem struct {
	files *process.FileSystem
}

// NewOSFileSystem builds the real file system gateway.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{files: process.NewFileSystem()}
}

// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist), which is
// what the use case branches on.
func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return f.files.ReadFile(path)
}

// MkdirAll creates path and every parent it needs, and does nothing when it already exists, which is
// what makes init safe to run again.
func (f *OSFileSystem) MkdirAll(path string) error {
	return f.files.MkdirAll(path)
}

// WriteFile writes data to path, replacing whatever was there.
func (f *OSFileSystem) WriteFile(path string, data []byte) error {
	return f.files.WriteFile(path, data)
}

// Remove deletes one file or one empty directory. A path that is not there is success, which is what
// the cleanup step wants: it asked for the path to be gone.
func (f *OSFileSystem) Remove(path string) error {
	return f.files.Remove(path)
}

// DirIsEmpty reports whether path is a directory holding nothing, which is what decides whether an
// emptied parent goes with the file the cleanup step removed from it.
func (f *OSFileSystem) DirIsEmpty(path string) (bool, error) {
	return f.files.DirIsEmpty(path)
}
