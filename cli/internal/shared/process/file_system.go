package process

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// The permissions codefall creates things with. A repository's configuration is not a secret and is
// read by the tools codefall drives, so a directory is traversable and a file is world-readable —
// the modes git itself uses for a checkout. A use case does not choose them: it asks for a directory
// and a file, and this is where the operating system's terms are decided.
const (
	dirMode  fs.FileMode = 0o755
	fileMode fs.FileMode = 0o644
)

// How a readable file becomes a runnable one: the read bits of all three classes, and the distance
// from a read bit to the execute bit beside it.
const readBits fs.FileMode = 0o444

const readToExecute = 2

// FileSystem reads and writes the working directory through the operating system. It is the union of
// what the commands need; each one's gateway exposes only its own half.
type FileSystem struct{}

// NewFileSystem builds the real file system.
func NewFileSystem() *FileSystem {
	return &FileSystem{}
}

// DirExists reports whether path is a directory. A path that is not there is not an error — that is
// the answer the caller asked for.
func (*FileSystem) DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return info.IsDir(), nil
}

// Exists reports whether anything is at path — a directory, a file, or a link, which is not followed.
// A path that is not there is not an error — that is the answer the caller asked for.
func (*FileSystem) Exists(path string) (bool, error) {
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ReadFile returns a file's bytes. os.ReadFile's *PathError already satisfies
// errors.Is(err, fs.ErrNotExist), which is what the use cases branch on.
func (*FileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// MkdirAll creates path and every parent it needs. An existing directory is success, which is what
// makes init safe to run again.
func (*FileSystem) MkdirAll(path string) error {
	return os.MkdirAll(path, dirMode)
}

// WriteFile writes data to path, replacing whatever was there. The mode applies only to a file this
// call creates; the permissions of one that already exists are left alone, as os.WriteFile has it.
func (*FileSystem) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, fileMode)
}

// MakeExecutable marks a copied script runnable. WriteFile's mode is deliberately not executable —
// most of what codefall writes is configuration — so the step that installs a script says so here,
// rather than the use case choosing modes it was promised not to think about.
//
// The bit is added to the mode the file already has, not written over it: WriteFile leaves an
// existing file's permissions alone, so a project that tightened a copied script to its own user
// would otherwise have it widened back on every rerun. Execute is added wherever read is already
// allowed, which is the rule a mode of 0700 and a mode of 0644 both come out of correctly. A file
// that is already runnable everywhere it is readable is left untouched, so a directory that refuses
// chmod does not fail a rerun that had nothing to change.
func (*FileSystem) MakeExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	executable := mode | (mode&readBits)>>readToExecute

	if executable == mode {
		return nil
	}

	return os.Chmod(path, executable)
}

// Remove deletes one file, or one empty directory. A path that is not there is success: the caller
// asked for it to be gone, and it is.
func (*FileSystem) Remove(path string) error {
	err := os.Remove(path)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}

// DirIsEmpty reports whether path is a directory holding nothing. A path that is not there, or that
// is a file, is not an empty directory, and neither is an error — that is the answer the caller
// asked for.
func (*FileSystem) DirIsEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		var pathErr *fs.PathError
		if errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.ENOTDIR) {
			return false, nil
		}

		return false, err
	}

	return len(entries) == 0, nil
}
