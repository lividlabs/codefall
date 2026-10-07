// Package domain holds what update knows about a codefall binary: how it was installed, the receipt
// the install script writes beside it, and the release files it downloads to replace it. It does no
// IO (ADR-015).
package domain

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

// The commands update prints for an install it leaves alone. Each is the command a person types, so
// the presentation prints them as they are.
const (
	// InstallScriptCommand installs the latest release natively and writes the receipt.
	InstallScriptCommand = "curl -fsSL https://install.codefall.dev/sh | sh"
	// MiseTool is the name mise installs codefall under.
	MiseTool = "packslip:github.com/lividlabs/codefall"
	// GoInstallCommand builds the latest release from source.
	GoInstallCommand = "go install github.com/lividlabs/codefall/cli/cmd/codefall@latest"
)

// Method is how the running binary was installed.
type Method int

const (
	// MethodUnknown is a binary none of the checks recognise: a release copied by hand, or one a
	// script installed before the receipt existed.
	MethodUnknown Method = iota
	// MethodNative is a binary the install script installed, which the receipt names.
	MethodNative
	// MethodMise is a binary in one of mise's per-version install directories.
	MethodMise
	// MethodSource is a binary built by `go install` or from a checkout, which carries no release
	// version.
	MethodSource
)

// String is how the method reads in a report.
func (m Method) String() string {
	switch m {
	case MethodNative:
		return "install script"
	case MethodMise:
		return "mise"
	case MethodSource:
		return "go install or a checkout build"
	default:
		return "unknown"
	}
}

// Evidence is what the checks read. Every path in it has its symbolic links resolved, so two paths to
// the same file compare equal.
type Evidence struct {
	// Executable is the running binary.
	Executable string
	// ReceiptPath is the binary the receipt names, and None when there is no receipt.
	ReceiptPath mo.Option[string]
	// MiseDataDir is MISE_DATA_DIR, and None when it is not set.
	MiseDataDir mo.Option[string]
	// Release is the running binary's release version, and None for a development build.
	Release mo.Option[version.Version]
}

// miseInstalls is the segment every path under mise's default data directory carries.
const miseInstalls = "/mise/installs/"

// Detect decides how the running binary was installed, checking in the order ADR-015 fixes: the
// receipt, mise's install directories, the release version, and otherwise unknown.
func Detect(e Evidence) Method {
	if receipt, ok := e.ReceiptPath.Get(); ok && filepath.Clean(receipt) == filepath.Clean(e.Executable) {
		return MethodNative
	}

	if underMise(e.Executable, e.MiseDataDir) {
		return MethodMise
	}

	if e.Release.IsAbsent() {
		return MethodSource
	}

	return MethodUnknown
}

// underMise reports whether path is in one of mise's install directories: under MISE_DATA_DIR's
// installs/ when that is set, and otherwise anywhere with a /mise/installs/ segment.
func underMise(path string, dataDir mo.Option[string]) bool {
	if dir, ok := dataDir.Get(); ok {
		installs := filepath.Join(dir, "installs") + string(filepath.Separator)

		return strings.HasPrefix(filepath.Clean(path), installs)
	}

	return strings.Contains(filepath.ToSlash(path), miseInstalls)
}

// Installation is the running binary and how it was installed.
type Installation struct {
	Method Method
	// Path is the running binary, with symbolic links resolved.
	Path string
	// Release is its release version, and None for a development build.
	Release mo.Option[version.Version]
	// Build is the version it reports, which for a development build names the commit.
	Build string
}

// ErrNotWritable is a native binary in a directory update cannot write to. Update never asks for
// elevated privileges; the person reinstalls with the install script instead.
var ErrNotWritable = errors.New("cannot write the binary's directory")

// ErrNotRelease is a requested version that is not a release version.
var ErrNotRelease = errors.New("not a release version")

// Outcome is what a native update did.
type Outcome struct {
	// Path is the binary update replaced, or would have.
	Path string
	// From is the release that was there, and None for a development build.
	From mo.Option[version.Version]
	// To is the release now there.
	To version.Version
	// Changed is false when the binary was already at To.
	Changed bool
}
