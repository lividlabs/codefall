package extensions

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

// The embedded tree holds every file the directory holds. Go's embed skips names beginning with
// `_` or `.` when it is given a directory, which is how `_TEMPLATE.md` shipped in the repo for
// thirty releases and reached no project: a file the directory holds and the binary does not is a
// template nobody receives.
func TestEmbeddedTreeMatchesTheDirectory(t *testing.T) {
	roots := embeddedRoots(t)
	onDisk := regularFiles(t, os.DirFS("."), roots)
	embedded := regularFiles(t, Files(), roots)

	for path := range onDisk {
		if !embedded[path] {
			t.Errorf("%s is in the directory and not in the binary", path)
		}
	}
	for path := range embedded {
		if !onDisk[path] {
			t.Errorf("%s is in the binary and not in the directory", path)
		}
	}
}

// embeddedRoots reads what the go:embed directive in embed.go names, so the test cannot drift from
// the directive.
func embeddedRoots(t *testing.T) []string {
	t.Helper()
	source, err := os.ReadFile("embed.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(source), "\n") {
		if rest, ok := strings.CutPrefix(line, "//go:embed "); ok {
			roots := strings.Fields(rest)
			for i, root := range roots {
				roots[i] = strings.TrimPrefix(root, "all:")
			}
			return roots
		}
	}
	t.Fatal("embed.go holds no go:embed directive")
	return nil
}

// regularFiles returns every regular file under the roots. The directory walk sees `_` and `.`
// names, which is the point.
func regularFiles(t *testing.T, fsys fs.FS, roots []string) map[string]bool {
	t.Helper()
	files := map[string]bool{}
	for _, root := range roots {
		err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsRegular() {
				files[path] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
	return files
}
