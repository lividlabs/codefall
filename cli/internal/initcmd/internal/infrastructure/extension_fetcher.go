package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"sort"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall/cli/internal/shared/process"
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
// returns the relative paths it installed (a manifest-of-one-copy) and the ones it had to write, both
// sorted so two sequential runs produce the same record. Each file keeps the path it has in the
// tree, so a caller that asks for "hooks/shared" gets it back at destDir/hooks/shared.
func (f *EmbeddedExtensionFetcher) Fetch(
	ctx context.Context, destDir string, sources, exclude []string,
) (application.Fetched, error) {
	var fetched application.Fetched

	for _, source := range sources {
		if err := f.mirror(ctx, destDir, source, exclude, &fetched); err != nil {
			sort.Strings(fetched.Files)
			sort.Strings(fetched.Changed)

			return fetched, err
		}
	}

	sort.Strings(fetched.Files)
	sort.Strings(fetched.Changed)

	return fetched, nil
}

// mirror copies one subtree into fetched. A source the tree does not hold is an error rather than
// nothing copied: the caller named a subtree this binary was meant to ship.
//
// A file already holding the tree's bytes is not written again, so a copy over an unchanged install
// reports no change. A file that is missing, holds other bytes, or cannot be read is written; one
// that cannot be read then fails at the write, with the reason.
func (f *EmbeddedExtensionFetcher) mirror(
	ctx context.Context, destDir, source string, exclude []string, fetched *application.Fetched,
) error {
	return fs.WalkDir(f.src, source, func(path string, d fs.DirEntry, err error) error {
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

		if existing, err := f.files.ReadFile(target); err != nil || !bytes.Equal(existing, data) {
			if err := f.files.MkdirAll(filepath.Dir(target)); err != nil {
				return fmt.Errorf("create %s: %w", filepath.Dir(target), err)
			}

			if err := f.files.WriteFile(target, data); err != nil {
				return fmt.Errorf("write %s: %w", target, err)
			}

			fetched.Changed = append(fetched.Changed, path)
		}

		// The hooks the definitions register are invoked by path, so a copied script has to be
		// runnable where it lands: the embedded tree carries no modes to copy. A script whose bytes
		// already matched still gets the bit, in case a person took it away.
		if strings.HasSuffix(path, ".sh") {
			if err := f.files.MakeExecutable(target); err != nil {
				return fmt.Errorf("make %s executable: %w", target, err)
			}
		}

		fetched.Files = append(fetched.Files, path)
		return nil
	})
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
