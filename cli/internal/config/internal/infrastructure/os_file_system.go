// Package infrastructure implements config's gateway against the real process: the file system it
// reads the settings, the user file, and .gitignore from and writes them back to. It is a thin
// adapter over `internal/shared/process`, whose permissions a write uses.
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

// WriteFile writes data to path, replacing whatever was there.
func (f *OSFileSystem) WriteFile(path string, data []byte) error {
	return f.files.WriteFile(path, data)
}
