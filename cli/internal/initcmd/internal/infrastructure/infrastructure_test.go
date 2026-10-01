package infrastructure

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
)

var (
	_ application.FileSystem    = (*OSFileSystem)(nil)
	_ application.CommandRunner = (*ExecCommandRunner)(nil)
)

// What the shared module does is pinned by its own tests. What is left here is this adapter's whole
// job: that it delegates, and that a process result arrives in initcmd's own terms.

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

// A non-zero exit is a normal result carried across whole, which is what the steps read.
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
	nested := filepath.Join(root, "a", ".codefall")

	files := NewOSFileSystem()

	if err := files.MkdirAll(nested); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	path := filepath.Join(nested, "settings.json")
	if err := files.WriteFile(path, []byte(`{"version":1}`)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	data, err := files.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != `{"version":1}` {
		t.Errorf("ReadFile = %q, want %q", data, `{"version":1}`)
	}

	// The use case branches on this exact sentinel to decide whether settings are already there, so
	// it is the behaviour worth pinning here too.
	if _, err := files.ReadFile(filepath.Join(root, "missing.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file = %v, want an fs.ErrNotExist", err)
	}

	if _, err := os.Stat(nested); err != nil {
		t.Errorf("stat of the directory MkdirAll made: %v", err)
	}
}
