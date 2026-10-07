package application

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
	"github.com/lividlabs/codefall/cli/internal/update/internal/domain"
)

const (
	home        = "/home/someone"
	binary      = "/home/someone/.local/bin/codefall"
	receiptFile = "/home/someone/.local/state/codefall/install.json"
	newBinary   = "codefall 0.29.0"
)

var (
	v28 = version.Version{Minor: 28}
	v29 = version.Version{Minor: 29}
)

// fakeHost is a process whose binary, version, and environment the test chooses. Resolve follows the
// links it is given and reports anything in missing as not there.
type fakeHost struct {
	executable string
	build      string
	env        map[string]string
	links      map[string]string
	missing    map[string]bool
}

func (h *fakeHost) Executable() (string, error)     { return h.executable, nil }
func (h *fakeHost) BuildVersion() string            { return h.build }
func (h *fakeHost) Platform() (goos, goarch string) { return "linux", "amd64" }
func (h *fakeHost) HomeDir() (string, error)        { return home, nil }
func (h *fakeHost) LookupEnv(name string) mo.Option[string] {
	return mo.TupleToOption(h.env[name], h.env[name] != "")
}

func (h *fakeHost) Resolve(path string) (string, error) {
	if h.missing[path] {
		return "", fs.ErrNotExist
	}

	if target, ok := h.links[path]; ok {
		return target, nil
	}

	return path, nil
}

// fakeFiles is a file system in memory. A path in denied refuses writes with a permission error.
type fakeFiles struct {
	contents map[string][]byte
	modes    map[string]fs.FileMode
	denied   map[string]bool
}

func newFakeFiles() *fakeFiles {
	return &fakeFiles{contents: map[string][]byte{}, modes: map[string]fs.FileMode{}, denied: map[string]bool{}}
}

func (f *fakeFiles) ReadFile(path string) ([]byte, error) {
	data, ok := f.contents[path]
	if !ok {
		return nil, fs.ErrNotExist
	}

	return data, nil
}

func (f *fakeFiles) Replace(path string, data []byte, mode fs.FileMode) error {
	if f.denied[path] {
		return &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
	}

	f.contents[path] = data
	f.modes[path] = mode

	return nil
}

// fakeReleases serves a latest file and release files from memory, and counts what it fetched.
type fakeReleases struct {
	latest  string
	files   map[string][]byte
	fetched []string
}

func (r *fakeReleases) Latest(context.Context) (string, error) { return r.latest, nil }

func (r *fakeReleases) Fetch(_ context.Context, release version.Version, name string) ([]byte, error) {
	key := release.String() + "/" + name
	r.fetched = append(r.fetched, key)

	data, ok := r.files[key]
	if !ok {
		return nil, fmt.Errorf("GET %s: 404 Not Found", key)
	}

	return data, nil
}

// releaseOf publishes one release for linux/amd64 with a matching checksums.txt.
func releaseOf(t *testing.T, v version.Version, contents string) map[string][]byte {
	t.Helper()

	var buffer bytes.Buffer

	zipped := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(zipped)

	if err := writer.WriteHeader(&tar.Header{
		Name: "codefall", Mode: 0o755, Size: int64(len(contents)), Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := writer.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}

	archive := buffer.Bytes()
	name := domain.ArchiveName(v, "linux", "amd64")
	sum := sha256.Sum256(archive)

	return map[string][]byte{
		v.String() + "/" + name:       archive,
		v.String() + "/checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + name + "\n"),
	}
}

// nativeFixture is a script install of 0.28.0 with its receipt, and 0.29.0 published as latest.
func nativeFixture(t *testing.T) (*fakeHost, *fakeFiles, *fakeReleases) {
	t.Helper()

	host := &fakeHost{executable: binary, build: "0.28.0"}
	files := newFakeFiles()
	files.contents[receiptFile] = domain.Receipt{Path: binary, Version: "0.28.0"}.Encode()
	releases := &fakeReleases{latest: "0.29.0\n", files: releaseOf(t, v29, newBinary)}

	return host, files, releases
}

func TestInspectNative(t *testing.T) {
	host, files, releases := nativeFixture(t)

	installation, err := NewUpdate(host, files, releases).Inspect()
	if err != nil {
		t.Fatal(err)
	}

	if installation.Method != domain.MethodNative || installation.Path != binary ||
		installation.Release != mo.Some(v28) || installation.Build != "0.28.0" {
		t.Errorf("Inspect = %+v", installation)
	}
}

func TestInspectResolvesTheReceiptsPath(t *testing.T) {
	host, files, releases := nativeFixture(t)
	files.contents[receiptFile] = domain.Receipt{Path: "/home/someone/bin/codefall", Version: "0.28.0"}.Encode()
	host.links = map[string]string{"/home/someone/bin/codefall": binary}

	installation, _ := NewUpdate(host, files, releases).Inspect()
	if installation.Method != domain.MethodNative {
		t.Errorf("Method = %s, want native", installation.Method)
	}
}

func TestInspectWithoutAReceiptStillChecks(t *testing.T) {
	for _, tc := range []struct {
		name       string
		executable string
		build      string
		want       domain.Method
	}{
		{"mise", "/home/someone/.local/share/mise/installs/codefall/0.28.0/codefall", "0.28.0", domain.MethodMise},
		{"source", "/home/someone/go/bin/codefall", "0.28.0-dev (abc1234)", domain.MethodSource},
		{"unknown", binary, "0.28.0", domain.MethodUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{executable: tc.executable, build: tc.build}

			installation, err := NewUpdate(host, newFakeFiles(), &fakeReleases{}).Inspect()
			if err != nil {
				t.Fatal(err)
			}

			if installation.Method != tc.want {
				t.Errorf("Method = %s, want %s", installation.Method, tc.want)
			}
		})
	}
}

func TestInspectIgnoresAReceiptWhoseBinaryIsGone(t *testing.T) {
	host, files, releases := nativeFixture(t)
	host.missing = map[string]bool{binary: true}

	installation, _ := NewUpdate(host, files, releases).Inspect()
	if installation.Method != domain.MethodUnknown {
		t.Errorf("Method = %s, want unknown", installation.Method)
	}
}

func TestInspectReadsTheReceiptUnderXDGStateHome(t *testing.T) {
	host, files, releases := nativeFixture(t)
	host.env = map[string]string{"XDG_STATE_HOME": "/state"}
	files.contents["/state/codefall/install.json"] = files.contents[receiptFile]
	delete(files.contents, receiptFile)

	installation, _ := NewUpdate(host, files, releases).Inspect()
	if installation.Method != domain.MethodNative {
		t.Errorf("Method = %s, want native", installation.Method)
	}
}

func TestRunReplacesTheBinaryAndTheReceipt(t *testing.T) {
	host, files, releases := nativeFixture(t)
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	outcome, err := update.Run(context.Background(), installation, mo.None[string]())
	if err != nil {
		t.Fatal(err)
	}

	if !outcome.Changed || outcome.From != mo.Some(v28) || outcome.To != v29 || outcome.Path != binary {
		t.Errorf("Run = %+v", outcome)
	}

	if string(files.contents[binary]) != newBinary || files.modes[binary] != 0o755 {
		t.Errorf("binary = %q mode %v", files.contents[binary], files.modes[binary])
	}

	receipt, err := domain.ParseReceipt(files.contents[receiptFile])
	if err != nil || receipt != (domain.Receipt{Path: binary, Version: "0.29.0"}) {
		t.Errorf("receipt = %+v, %v", receipt, err)
	}
}

func TestRunInstallsARequestedVersion(t *testing.T) {
	host, files, releases := nativeFixture(t)
	host.build = "0.29.0"
	releases.files = releaseOf(t, v28, "codefall 0.28.0")
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	outcome, err := update.Run(context.Background(), installation, mo.Some("v0.28.0"))
	if err != nil {
		t.Fatal(err)
	}

	if !outcome.Changed || outcome.To != v28 || string(files.contents[binary]) != "codefall 0.28.0" {
		t.Errorf("Run = %+v, binary %q", outcome, files.contents[binary])
	}
}

func TestRunAlreadyUpToDate(t *testing.T) {
	host, files, releases := nativeFixture(t)
	host.build = "0.29.0"
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	outcome, err := update.Run(context.Background(), installation, mo.None[string]())
	if err != nil {
		t.Fatal(err)
	}

	if outcome.Changed || len(releases.fetched) != 0 {
		t.Errorf("Run = %+v, fetched %v", outcome, releases.fetched)
	}
}

func TestRunRefusesATamperedArchive(t *testing.T) {
	host, files, releases := nativeFixture(t)
	name := "0.29.0/" + domain.ArchiveName(v29, "linux", "amd64")
	releases.files[name] = append(releases.files[name], 0)
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	_, err := update.Run(context.Background(), installation, mo.None[string]())
	if !errors.Is(err, domain.ErrChecksumMismatch) {
		t.Errorf("Run error = %v, want ErrChecksumMismatch", err)
	}

	if _, replaced := files.contents[binary]; replaced {
		t.Error("a tampered archive replaced the binary")
	}
}

func TestRunReportsADirectoryItCannotWrite(t *testing.T) {
	host, files, releases := nativeFixture(t)
	files.denied[binary] = true
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	_, err := update.Run(context.Background(), installation, mo.None[string]())
	if !errors.Is(err, domain.ErrNotWritable) {
		t.Errorf("Run error = %v, want ErrNotWritable", err)
	}
}

func TestRunReportsAReceiptItCannotWrite(t *testing.T) {
	host, files, releases := nativeFixture(t)
	files.denied[receiptFile] = true
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	outcome, err := update.Run(context.Background(), installation, mo.None[string]())
	if err == nil || !outcome.Changed {
		t.Errorf("Run = %+v, %v; want the binary replaced and an error", outcome, err)
	}
}

func TestRunRefusesWhatTheScriptDidNotInstall(t *testing.T) {
	host, files, releases := nativeFixture(t)

	for _, method := range []domain.Method{domain.MethodMise, domain.MethodSource, domain.MethodUnknown} {
		installation := domain.Installation{Method: method, Path: binary, Release: mo.Some(v28)}

		if _, err := NewUpdate(host, files, releases).Run(context.Background(), installation, mo.None[string]()); err == nil {
			t.Errorf("Run of a %s install returned no error", method)
		}
	}

	if len(releases.fetched) != 0 {
		t.Errorf("fetched %v for installs update leaves alone", releases.fetched)
	}
}

func TestRunRejectsARequestThatIsNotARelease(t *testing.T) {
	host, files, releases := nativeFixture(t)
	update := NewUpdate(host, files, releases)
	installation, _ := update.Inspect()

	if _, err := update.Run(context.Background(), installation, mo.Some("latest")); !errors.Is(err, domain.ErrNotRelease) {
		t.Errorf("Run error = %v, want ErrNotRelease", err)
	}
}
