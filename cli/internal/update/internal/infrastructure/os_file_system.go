package infrastructure

import (
	"io/fs"
	"os"
	"path/filepath"
)

// dirMode is the permissions of a directory Replace creates, the ones mkdir -p gives it.
const dirMode fs.FileMode = 0o755

// OSFileSystem reads and replaces files through the operating system.
type OSFileSystem struct{}

// NewOSFileSystem builds the real file system gateway.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// ReadFile returns a file's bytes.
func (*OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Replace writes data to a temporary file in path's directory and renames it over path. The rename
// is atomic within one directory, so an interrupted update leaves the old file in place, and a binary
// that is running keeps running from the file it was started from.
func (*OSFileSystem) Replace(path string, data []byte, mode fs.FileMode) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}

	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = os.Remove(temp.Name())
		}
	}()

	_, err = temp.Write(data)
	if err == nil {
		err = temp.Chmod(mode)
	}

	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		return err
	}

	return os.Rename(temp.Name(), path)
}
