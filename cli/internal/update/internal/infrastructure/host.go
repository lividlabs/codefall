// Package infrastructure implements update's gateways against the real process: the running binary
// and its environment, the file system, and install.codefall.dev over HTTPS.
package infrastructure

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/buildinfo"
)

// OSHost is the process update runs in.
type OSHost struct{}

// NewOSHost builds the real host gateway.
func NewOSHost() *OSHost {
	return &OSHost{}
}

// Executable is the running binary, with symbolic links resolved. macOS can report the path the
// binary was started by, which may be a link; Linux reports the file itself.
func (*OSHost) Executable() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(path)
}

// BuildVersion is the version the binary reports.
func (*OSHost) BuildVersion() string {
	return buildinfo.Version()
}

// Platform is the platform the binary was built for, which is the one its replacement is built for.
func (*OSHost) Platform() (goos, goarch string) {
	return runtime.GOOS, runtime.GOARCH
}

// LookupEnv is an environment variable, and None when it is not set.
func (*OSHost) LookupEnv(name string) mo.Option[string] {
	return mo.TupleToOption(os.LookupEnv(name))
}

// HomeDir is the user's home directory.
func (*OSHost) HomeDir() (string, error) {
	return os.UserHomeDir()
}

// Resolve is path with its symbolic links resolved.
func (*OSHost) Resolve(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
