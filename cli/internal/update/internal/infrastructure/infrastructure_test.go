package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/shared/version"
)

func TestHTTPReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			_, _ = w.Write([]byte("0.29.0\n"))
		case "/0.29.0/checksums.txt":
			_, _ = w.Write([]byte("sums"))
		case "/0.1.0/checksums.txt":
			// What the bucket answers for a file it does not hold.
			w.WriteHeader(http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	releases := NewHTTPReleases(server.URL + "/")

	latest, err := releases.Latest(context.Background())
	if err != nil || latest != "0.29.0" {
		t.Errorf("Latest = %q, %v", latest, err)
	}

	body, err := releases.Fetch(context.Background(), version.Version{Minor: 29}, "checksums.txt")
	if err != nil || string(body) != "sums" {
		t.Errorf("Fetch = %q, %v", body, err)
	}

	if _, err := releases.Fetch(context.Background(), version.Version{Minor: 1}, "checksums.txt"); err == nil ||
		!strings.Contains(err.Error(), "is not published") {
		t.Errorf("Fetch of a file the bucket refuses: %v, want it reported as not published", err)
	}

	if _, err := releases.Fetch(context.Background(), version.Version{Minor: 29}, "missing"); err == nil ||
		!strings.Contains(err.Error(), "is not published") {
		t.Errorf("Fetch of a missing file: %v, want it reported as not published", err)
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin", "codefall")
	files := NewOSFileSystem()

	if err := files.Replace(path, []byte("one"), 0o755); err != nil {
		t.Fatalf("Replace into a new directory: %v", err)
	}

	if err := files.Replace(path, []byte("two"), 0o755); err != nil {
		t.Fatalf("Replace over a file: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil || string(data) != "two" {
		t.Errorf("file = %q, %v", data, err)
	}

	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, %v", info.Mode().Perm(), err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Errorf("directory holds %d entries, want only the file: %v", len(entries), err)
	}
}

func TestReplaceInADirectoryItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere")
	}

	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := NewOSFileSystem().Replace(filepath.Join(dir, "codefall"), []byte("new"), 0o755)
	if !os.IsPermission(err) {
		t.Errorf("Replace error = %v, want a permission error", err)
	}
}
