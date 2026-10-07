package domain

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

var (
	// ErrChecksumMissing is a checksums.txt with no line for the archive.
	ErrChecksumMissing = errors.New("checksums.txt has no entry for the archive")
	// ErrChecksumMismatch is an archive whose SHA-256 is not the one checksums.txt records.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrNoBinary is an archive with no codefall binary at its root.
	ErrNoBinary = errors.New("archive has no codefall binary")
)

// binaryName is the binary's name inside a release archive, at the archive's root.
const binaryName = "codefall"

// maxBinary bounds what extraction reads, so a corrupt or hostile archive cannot fill memory. A
// release binary is about 10 MB.
const maxBinary = 256 << 20

// ArchiveName is a release archive's file name for one platform, as GoReleaser names it.
func ArchiveName(v version.Version, goos, goarch string) string {
	return fmt.Sprintf("codefall_%s_%s_%s.tar.gz", v, goos, goarch)
}

// Verify checks an archive against a release's checksums.txt, whose lines are a SHA-256 in hex, two
// spaces, and a file name.
func Verify(archive []byte, checksums []byte, name string) error {
	want, ok := checksumFor(checksums, name)
	if !ok {
		return fmt.Errorf("%w %s", ErrChecksumMissing, name)
	}

	sum := sha256.Sum256(archive)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), want) {
		return fmt.Errorf("%w for %s", ErrChecksumMismatch, name)
	}

	return nil
}

func checksumFor(checksums []byte, name string) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(checksums))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == name {
			return fields[0], true
		}
	}

	return "", false
}

// ExtractBinary returns the codefall binary from a release archive, a gzipped tar with the binary at
// its root beside the README, the licence, and the changelog.
func ExtractBinary(archive []byte) ([]byte, error) {
	unzipped, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}

	reader := tar.NewReader(unzipped)

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil, ErrNoBinary
		}

		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg || strings.TrimPrefix(header.Name, "./") != binaryName {
			continue
		}

		binary, err := io.ReadAll(io.LimitReader(reader, maxBinary+1))
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}

		if len(binary) > maxBinary {
			return nil, fmt.Errorf("reading archive: %s is larger than %d bytes", binaryName, maxBinary)
		}

		return binary, nil
	}
}
