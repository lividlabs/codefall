package infrastructure

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/application"
)

var (
	_ application.FileSystem    = (*OSFileSystem)(nil)
	_ application.CommandRunner = (*ExecCommandRunner)(nil)
)

// What the shared module does is pinned by its own tests. What is left here is this adapter's whole
// job: that it delegates, and that a process result arrives in doctor's own terms.

func TestExecCommandRunnerDelegates(t *testing.T) {
	runner := NewExecCommandRunner()

	if path, ok := runner.LookPath("sh").Get(); !ok || path == "" {
		t.Errorf(`LookPath("sh") = %v, want a path`, runner.LookPath("sh"))
	}

	if got := runner.LookPath("no-such-binary-codefall"); got.IsPresent() {
		t.Errorf("LookPath of a missing binary = %v, want None", got)
	}

	if _, err := runner.Run(t.Context(), t.TempDir(), "no-such-binary-codefall"); err == nil {
		t.Error("Run of a missing binary = nil error, want an error")
	}
}

// A non-zero exit is a normal result carried across whole, which is what the checks read.
func TestExecCommandRunnerConvertsTheResult(t *testing.T) {
	runner := NewExecCommandRunner()

	got, err := runner.Run(t.Context(), t.TempDir(), "sh", "-c", "echo out; echo err 1>&2; exit 3")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := application.CommandResult{Stdout: "out\n", Stderr: "err\n", ExitCode: 3}
	if got != want {
		t.Errorf("Run = %+v, want %+v", got, want)
	}
}

func TestOSFileSystemDelegates(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.json")

	if err := os.WriteFile(path, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := NewOSFileSystem()

	data, err := files.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != `{"version":1}` {
		t.Errorf("ReadFile = %q, want %q", data, `{"version":1}`)
	}

	// The use case branches on this exact sentinel, so it is the behaviour worth pinning here too.
	if _, err := files.ReadFile(filepath.Join(root, "missing.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file = %v, want an fs.ErrNotExist", err)
	}

	exists, err := files.DirExists(root)
	if err != nil || !exists {
		t.Errorf("DirExists(%q) = %v, %v, want true, nil", root, exists, err)
	}

	exists, err = files.Exists(path)
	if err != nil || !exists {
		t.Errorf("Exists(%q) = %v, %v, want true, nil", path, exists, err)
	}

	exists, err = files.Exists(filepath.Join(root, "missing.sh"))
	if err != nil || exists {
		t.Errorf("Exists of a missing file = %v, %v, want false, nil", exists, err)
	}
}
