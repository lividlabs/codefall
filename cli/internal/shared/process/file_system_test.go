package process

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemDirExists(t *testing.T) {
	root := t.TempDir()

	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := NewFileSystem()

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"a directory", nested, true},
		{"a file is not a directory", file, false},
		{"a missing path is not an error", filepath.Join(root, "nope"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := files.DirExists(tc.path)
			if err != nil {
				t.Fatalf("DirExists: %v", err)
			}

			if got != tc.want {
				t.Errorf("DirExists(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestFileSystemExists(t *testing.T) {
	root := t.TempDir()

	file := filepath.Join(root, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	dangling := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "nope"), dangling); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	files := NewFileSystem()

	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"a directory", root, true},
		{"a file", file, true},
		{"a link whose target is missing is still there", dangling, true},
		{"a missing path is not an error", filepath.Join(root, "nope"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := files.Exists(tc.path)
			if err != nil {
				t.Fatalf("Exists: %v", err)
			}

			if got != tc.want {
				t.Errorf("Exists(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestFileSystemReadFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "project.json")

	if err := os.WriteFile(path, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	files := NewFileSystem()

	data, err := files.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != `{"version":1}` {
		t.Errorf("ReadFile = %q, want %q", data, `{"version":1}`)
	}

	// Both use cases branch on this exact sentinel to decide whether a file is there at all, so it
	// is the behaviour worth pinning.
	if _, err := files.ReadFile(filepath.Join(root, "missing.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file = %v, want an fs.ErrNotExist", err)
	}
}

// The permissions are the reason this module exists rather than the use cases calling os themselves,
// so they are asserted here. The umask can only take bits away, and the modes below are what a
// normal umask of 022 leaves alone.
func TestFileSystemMkdirAllCreatesATraversableDirectory(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", ".codefall")

	files := NewFileSystem()

	if err := files.MkdirAll(nested); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if !info.IsDir() {
		t.Fatalf("MkdirAll made a %v, want a directory", info.Mode())
	}

	if got := info.Mode().Perm(); got != dirMode {
		t.Errorf("mode = %v, want %v", got, dirMode)
	}

	// Running init twice must not fail on the directory it made the first time.
	if err := files.MkdirAll(nested); err != nil {
		t.Errorf("MkdirAll on an existing directory = %v, want nil", err)
	}
}

func TestFileSystemWriteFileCreatesAndReplaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), "project.json")

	files := NewFileSystem()

	if err := files.WriteFile(path, []byte("first\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if got := info.Mode().Perm(); got != fileMode {
		t.Errorf("mode = %v, want %v", got, fileMode)
	}

	// --force rewrites the file, so the second write must leave nothing of the first behind.
	if err := files.WriteFile(path, []byte("second\n")); err != nil {
		t.Fatalf("WriteFile again: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(data) != "second\n" {
		t.Errorf("contents = %q, want %q", data, "second\n")
	}
}

func TestFileSystemWriteFileReportsAMissingDirectory(t *testing.T) {
	files := NewFileSystem()

	if err := files.WriteFile(filepath.Join(t.TempDir(), "nope", "project.json"), nil); err == nil {
		t.Error("WriteFile into a missing directory = nil, want an error")
	}
}

// A script has to be runnable where it lands, and that is all this says. A project that tightened a
// copied script to its own user keeps that decision: init adds the execute bit wherever the file is
// already readable rather than replacing the mode, so a rerun does not widen 0700 back to 0755 on
// every upgrade.
func TestMakeExecutableAddsTheExecuteBitWithoutWideningTheMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode fs.FileMode
		want fs.FileMode
	}{
		{name: "a file as WriteFile leaves it", mode: 0o644, want: 0o755},
		{name: "a file the project tightened", mode: 0o600, want: 0o700},
		{name: "a file that is already runnable", mode: 0o755, want: 0o755},
		{name: "a group-readable file", mode: 0o640, want: 0o750},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "codefall-script.sh")
			if err := os.WriteFile(path, []byte("#!/bin/bash\n"), tc.mode); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}

			if err := NewFileSystem().MakeExecutable(path); err != nil {
				t.Fatalf("MakeExecutable: %v", err)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}

			if got := info.Mode().Perm(); got != tc.want {
				t.Errorf("mode = %v, want %v", got, tc.want)
			}
		})
	}
}
