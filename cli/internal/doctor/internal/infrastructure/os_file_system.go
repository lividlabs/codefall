// Package infrastructure implements doctor's gateways against the real process: the file system and
// the external tools codefall depends on. Both are thin adapters over `internal/shared/process` —
// what they add is doctor's own terms, because a gateway interface belongs to the use case that
// declared it.
package infrastructure

import (
	"github.com/lividlabs/codefall/cli/internal/shared/process"
)

// OSFileSystem reads the working directory through the operating system. Doctor never writes, so its
// gateway exposes only the half of the shared file system that reads.
type OSFileSystem struct {
	files *process.FileSystem
}

// NewOSFileSystem builds the real file system gateway.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{files: process.NewFileSystem()}
}

// DirExists reports whether path is a directory. A path that is not there is not an error — that is
// the answer the caller asked for.
func (f *OSFileSystem) DirExists(path string) (bool, error) {
	return f.files.DirExists(path)
}

// Exists reports whether anything is at path. A path that is not there is not an error either.
func (f *OSFileSystem) Exists(path string) (bool, error) {
	return f.files.Exists(path)
}

// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist), which is
// what the use case branches on.
func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return f.files.ReadFile(path)
}
