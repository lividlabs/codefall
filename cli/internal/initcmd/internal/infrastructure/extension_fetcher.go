package infrastructure

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"sort"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/process"
)

// EmbeddedExtensionFetcher serves the extension tree out of the embedded extensions FS, and answers
// for the skill names the tree used to use.
type EmbeddedExtensionFetcher struct {
	src     fs.FS
	renames map[string]string
	files   *process.FileSystem
}

// NewEmbeddedExtensionFetcher builds the gateway over the tree the binary was compiled with and the
// table of former skill directory names kept beside it. Both are injected so tests can drive an
// in-memory tree and a table of their own instead of the real ones.
func NewEmbeddedExtensionFetcher(src fs.FS, renames map[string]string) *EmbeddedExtensionFetcher {
	return &EmbeddedExtensionFetcher{src: src, renames: renames, files: process.NewFileSystem()}
}

// RenamedSkill returns the name a skill directory has now when former is a name it used to have.
func (f *EmbeddedExtensionFetcher) RenamedSkill(former string) mo.Option[string] {
	current, known := f.renames[former]
	if !known {
		return mo.None[string]()
	}

	return mo.Some(current)
}

// Fetch mirrors the named subtrees of the embedded tree onto destDir, except the excluded paths, and
// returns the relative paths it wrote (a manifest-of-one-copy) sorted so two sequential runs produce
// the same record. Each file keeps the path it has in the tree, so a caller that asks for
// "hooks/shared" gets it back at destDir/hooks/shared.
func (f *EmbeddedExtensionFetcher) Fetch(
	ctx context.Context, destDir string, sources, exclude []string,
) ([]string, error) {
	var installed []string

	for _, source := range sources {
		written, err := f.mirror(ctx, destDir, source, exclude)
		installed = append(installed, written...)

		if err != nil {
			sort.Strings(installed)
			return installed, err
		}
	}

	sort.Strings(installed)
	return installed, nil
}

// mirror copies one subtree. A source the tree does not hold is an error rather than nothing copied:
// the caller named a subtree this binary was meant to ship.
func (f *EmbeddedExtensionFetcher) mirror(
	ctx context.Context, destDir, source string, exclude []string,
) ([]string, error) {
	var installed []string

	err := fs.WalkDir(f.src, source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		if excluded(path, exclude) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(f.src, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		target := filepath.Join(destDir, path)

		if err := f.files.MkdirAll(filepath.Dir(target)); err != nil {
			return fmt.Errorf("create %s: %w", filepath.Dir(target), err)
		}

		if err := f.files.WriteFile(target, data); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}

		// The hooks the definitions register are invoked by path, so a copied script has to be
		// runnable where it lands: the embedded tree carries no modes to copy.
		if strings.HasSuffix(path, ".sh") {
			if err := f.files.MakeExecutable(target); err != nil {
				return fmt.Errorf("make %s executable: %w", target, err)
			}
		}

		installed = append(installed, path)
		return nil
	})

	return installed, err
}

// excluded reports whether a path in the tree is one the install leaves behind. An entry holding a
// slash names a path, and everything under it; an entry that is a bare file name matches that file
// wherever it sits, which is what a rule about a file beside every skill needs.
func excluded(path string, exclude []string) bool {
	name := path
	if at := strings.LastIndex(path, "/"); at >= 0 {
		name = path[at+1:]
	}

	for _, entry := range exclude {
		if strings.Contains(entry, "/") {
			if path == entry || strings.HasPrefix(path, entry+"/") {
				return true
			}

			continue
		}

		if name == entry {
			return true
		}
	}

	return false
}

// Read returns one file from the embedded tree: what the hook step consumes per harness.
func (f *EmbeddedExtensionFetcher) Read(path string) ([]byte, error) {
	return fs.ReadFile(f.src, path)
}
