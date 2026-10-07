package domain

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

// archiveOf builds a gzipped tar of regular files, in order.
func archiveOf(t *testing.T, files ...[2]string) []byte {
	t.Helper()

	var buffer bytes.Buffer

	zipped := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(zipped)

	for _, file := range files {
		header := &tar.Header{Name: file[0], Mode: 0o755, Size: int64(len(file[1])), Typeflag: tar.TypeReg}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}

		if _, err := writer.Write([]byte(file[1])); err != nil {
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}

func sumOf(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

func TestArchiveName(t *testing.T) {
	got := ArchiveName(version.Version{Minor: 28}, "darwin", "arm64")
	if want := "codefall_0.28.0_darwin_arm64.tar.gz"; got != want {
		t.Errorf("ArchiveName = %q, want %q", got, want)
	}
}

func TestVerify(t *testing.T) {
	archive := []byte("archive bytes")
	name := "codefall_0.28.0_darwin_arm64.tar.gz"
	checksums := []byte(sumOf([]byte("other")) + "  codefall_0.28.0_linux_amd64.tar.gz\n" +
		sumOf(archive) + "  " + name + "\n")

	if err := Verify(archive, checksums, name); err != nil {
		t.Errorf("Verify of a matching archive: %v", err)
	}

	if err := Verify([]byte("tampered"), checksums, name); !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("Verify of a tampered archive: %v, want ErrChecksumMismatch", err)
	}

	if err := Verify(archive, checksums, "codefall_0.28.0_windows_amd64.tar.gz"); !errors.Is(err, ErrChecksumMissing) {
		t.Errorf("Verify of an archive checksums.txt does not list: %v, want ErrChecksumMissing", err)
	}
}

func TestExtractBinary(t *testing.T) {
	archive := archiveOf(t,
		[2]string{"CHANGELOG.md", "changes"},
		[2]string{"README.md", "readme"},
		[2]string{"codefall", "the binary"},
	)

	binary, err := ExtractBinary(archive)
	if err != nil {
		t.Fatalf("ExtractBinary: %v", err)
	}

	if string(binary) != "the binary" {
		t.Errorf("ExtractBinary = %q", binary)
	}
}

func TestExtractBinaryWithoutOne(t *testing.T) {
	archive := archiveOf(t, [2]string{"README.md", "readme"}, [2]string{"docs/codefall", "not at the root"})

	if _, err := ExtractBinary(archive); !errors.Is(err, ErrNoBinary) {
		t.Errorf("ExtractBinary error = %v, want ErrNoBinary", err)
	}
}

func TestExtractBinaryFromNotAnArchive(t *testing.T) {
	if _, err := ExtractBinary([]byte("not gzip")); err == nil {
		t.Error("ExtractBinary of bytes that are not gzip returned no error")
	}
}
