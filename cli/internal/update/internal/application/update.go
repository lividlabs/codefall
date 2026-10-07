// Package application holds update's use case: find out how the running binary was installed, and
// replace it with a release only when the install script put it there (ADR-015). It owns the gateway
// interfaces it needs and depends on nothing but the standard library, mo, update's own domain, and
// the pure shared modules (ADR-003).
package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
	"github.com/lividlabs/codefall/cli/internal/update/internal/domain"
)

// binaryMode is a replaced binary's permissions, the ones the install script gives it.
const binaryMode fs.FileMode = 0o755

// receiptMode is the receipt's permissions.
const receiptMode fs.FileMode = 0o644

// Host is the process update runs in (one gateway role).
type Host interface {
	// Executable is the running binary's path, with symbolic links resolved.
	Executable() (string, error)
	// BuildVersion is the version the binary reports: a release version, or a development build's.
	BuildVersion() string
	// Platform is the operating system and architecture the binary was built for, as Go spells them.
	Platform() (goos, goarch string)
	// LookupEnv is an environment variable, and None when it is not set.
	LookupEnv(name string) mo.Option[string]
	// HomeDir is the user's home directory.
	HomeDir() (string, error)
	// Resolve is path with its symbolic links resolved. A path that is not there is an error that
	// satisfies errors.Is(err, fs.ErrNotExist).
	Resolve(path string) (string, error)
}

// FileSystem reads the receipt and replaces files (one gateway role).
type FileSystem interface {
	// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist).
	ReadFile(path string) ([]byte, error)
	// Replace writes data to a temporary file beside path and renames it over path, creating path's
	// directory first, so a reader sees the old file or the new one and never part of either. A
	// directory that cannot be written satisfies errors.Is(err, fs.ErrPermission).
	Replace(path string, data []byte, mode fs.FileMode) error
}

// Releases reads the files install.codefall.dev publishes (one gateway role).
type Releases interface {
	// Latest is the text of the `latest` file, which names the newest release.
	Latest(ctx context.Context) (string, error)
	// Fetch is one file from a release's folder.
	Fetch(ctx context.Context, release version.Version, name string) ([]byte, error)
}

// Update is the use case behind `codefall update`.
type Update struct {
	host     Host
	files    FileSystem
	releases Releases
}

// NewUpdate builds the use case over its three gateways.
func NewUpdate(host Host, files FileSystem, releases Releases) *Update {
	return &Update{host: host, files: files, releases: releases}
}

// Inspect finds out how the running binary was installed. It reads the receipt, the environment, and
// the binary's own version, and changes nothing. A receipt that is missing or unreadable is not an
// error: the other checks still say what they can.
func (u *Update) Inspect() (domain.Installation, error) {
	executable, err := u.host.Executable()
	if err != nil {
		return domain.Installation{}, fmt.Errorf("finding the running binary: %w", err)
	}

	build := u.host.BuildVersion()
	release := mo.TupleToOption(version.Parse(build))

	evidence := domain.Evidence{
		Executable:  executable,
		ReceiptPath: u.receiptBinary(),
		MiseDataDir: u.resolvedEnv("MISE_DATA_DIR"),
		Release:     release,
	}

	return domain.Installation{
		Method:  domain.Detect(evidence),
		Path:    executable,
		Release: release,
		Build:   build,
	}, nil
}

// Latest is the newest release.
func (u *Update) Latest(ctx context.Context) (version.Version, error) {
	text, err := u.releases.Latest(ctx)
	if err != nil {
		return version.Version{}, fmt.Errorf("reading the latest release: %w", err)
	}

	latest, ok := version.Parse(text)
	if !ok {
		return version.Version{}, fmt.Errorf("reading the latest release: %q is %w", text, domain.ErrNotRelease)
	}

	return latest, nil
}

// Run replaces a native binary with the requested release, or the latest when none is requested,
// and records the new version in the receipt. It refuses any other installation; the caller inspects
// first and says what to run instead.
func (u *Update) Run(
	ctx context.Context, installation domain.Installation, requested mo.Option[string],
) (domain.Outcome, error) {
	if installation.Method != domain.MethodNative {
		return domain.Outcome{}, fmt.Errorf("update replaces only a binary the install script installed; this one: %s",
			installation.Method)
	}

	target, err := u.target(ctx, requested)
	if err != nil {
		return domain.Outcome{}, err
	}

	outcome := domain.Outcome{Path: installation.Path, From: installation.Release, To: target}

	if current, ok := installation.Release.Get(); ok && version.Compare(current, target) == 0 {
		return outcome, nil
	}

	binary, err := u.download(ctx, target)
	if err != nil {
		return domain.Outcome{}, err
	}

	if err := u.files.Replace(installation.Path, binary, binaryMode); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return domain.Outcome{}, fmt.Errorf("%w %s", domain.ErrNotWritable, filepath.Dir(installation.Path))
		}

		return domain.Outcome{}, fmt.Errorf("replacing %s: %w", installation.Path, err)
	}

	outcome.Changed = true

	receiptPath, err := u.receiptPath()
	if err == nil {
		receipt := domain.Receipt{Path: installation.Path, Version: target.String()}
		err = u.files.Replace(receiptPath, receipt.Encode(), receiptMode)
	}

	if err != nil {
		return outcome, fmt.Errorf("updated %s to %s, but could not record it in the install receipt: %w",
			installation.Path, target, err)
	}

	return outcome, nil
}

// target is the requested release, or the latest when none was requested.
func (u *Update) target(ctx context.Context, requested mo.Option[string]) (version.Version, error) {
	text, ok := requested.Get()
	if !ok {
		return u.Latest(ctx)
	}

	target, ok := version.Parse(text)
	if !ok {
		return version.Version{}, fmt.Errorf("%q is %w", text, domain.ErrNotRelease)
	}

	return target, nil
}

// download fetches a release's archive for this platform, checks it against the release's
// checksums.txt, and returns the binary inside it.
func (u *Update) download(ctx context.Context, target version.Version) ([]byte, error) {
	goos, goarch := u.host.Platform()
	name := domain.ArchiveName(target, goos, goarch)

	archive, err := u.releases.Fetch(ctx, target, name)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", name, err)
	}

	checksums, err := u.releases.Fetch(ctx, target, "checksums.txt")
	if err != nil {
		return nil, fmt.Errorf("downloading the checksums for %s: %w", target, err)
	}

	if err := domain.Verify(archive, checksums, name); err != nil {
		return nil, err
	}

	return domain.ExtractBinary(archive)
}

// receiptBinary is the binary the receipt names, resolved, and None when there is no receipt, it
// cannot be read, or the binary it names is gone.
func (u *Update) receiptBinary() mo.Option[string] {
	path, err := u.receiptPath()
	if err != nil {
		return mo.None[string]()
	}

	data, err := u.files.ReadFile(path)
	if err != nil {
		return mo.None[string]()
	}

	receipt, err := domain.ParseReceipt(data)
	if err != nil {
		return mo.None[string]()
	}

	resolved, err := u.host.Resolve(receipt.Path)
	if err != nil {
		return mo.None[string]()
	}

	return mo.Some(resolved)
}

func (u *Update) receiptPath() (string, error) {
	home, err := u.host.HomeDir()
	if err != nil {
		return "", fmt.Errorf("finding the home directory: %w", err)
	}

	return domain.ReceiptPath(u.host.LookupEnv("XDG_STATE_HOME"), home), nil
}

// resolvedEnv is an environment variable naming a directory, with its symbolic links resolved so it
// compares with the resolved executable. A directory that cannot be resolved is used as it is.
func (u *Update) resolvedEnv(name string) mo.Option[string] {
	value, ok := u.host.LookupEnv(name).Get()
	if !ok || value == "" {
		return mo.None[string]()
	}

	if resolved, err := u.host.Resolve(value); err == nil {
		return mo.Some(resolved)
	}

	return mo.Some(value)
}
