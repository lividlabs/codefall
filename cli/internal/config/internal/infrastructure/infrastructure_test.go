package infrastructure

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/application"
)

var _ application.FileSystem = (*OSFileSystem)(nil)

// What the shared module does is pinned by its own tests. What is left here is this adapter's whole
// job: that it delegates, and that a missing file is the sentinel the use case branches on.
func TestOSFileSystemDelegates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.json")
	files := NewOSFileSystem()

	if _, err := files.ReadFile(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file = %v, want an fs.ErrNotExist", err)
	}

	if err := files.WriteFile(path, []byte(`{"version":1}`)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := files.ReadFile(path)
	if err != nil || string(data) != `{"version":1}` {
		t.Errorf("ReadFile = %q, %v, want what was written", data, err)
	}
}
