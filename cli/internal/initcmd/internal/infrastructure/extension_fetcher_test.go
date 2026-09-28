package infrastructure

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"
	"time"
)

// The walk mirrors the whole source tree. The promise "everything that is in the source, nothing
// else" is checked with the names read from the destination, never from a second literal list.
func TestEmbeddedExtensionFetcherCopiesTheTree(t *testing.T) {
	src := fstest.MapFS{
		"hooks/shared/codefall-block-merge-to-main.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"skills/x/SKILL.md":                            &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
		"README.md":                                    &fstest.MapFile{Data: []byte("# extension\n")},
		"docs/ROADMAP.md":                              &fstest.MapFile{Data: []byte("road")},
	}
	fetcher := NewEmbeddedExtensionFetcher(src, nil)
	dest := t.TempDir()

	installed, err := fetcher.Fetch(context.Background(), dest, []string{"."}, nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got := readAllFiles(t, dest)
	if len(got) != len(src) {
		t.Fatalf("dest holds %d files, want all %d source files copied (%q)", len(got), len(src), got)
	}

	if len(installed.Files) != len(got) {
		t.Fatalf("installed = %d paths, want one per copied file (%v)", len(installed.Files), installed.Files)
	}
}

// The per-harness hook definitions are the hook step's business, so the walk skips them when it is
// told to; the shared scripts every harness runs still copy.
func TestEmbeddedExtensionFetcherSkipsTheExcludedPrefixes(t *testing.T) {
	src := fstest.MapFS{
		"hooks/claude/hooks.json":    &fstest.MapFile{Data: []byte(`{"hooks":{}}`)},
		"hooks/opencode/codefall.js": &fstest.MapFile{Data: []byte("// plugin\n")},
		"hooks/shared/codefall-block-merge-to-main.sh": &fstest.MapFile{
			Data: []byte("#!/bin/bash\n"),
		},
		"skills/x/SKILL.md": &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
	}
	fetcher := NewEmbeddedExtensionFetcher(src, nil)
	dest := t.TempDir()

	installed, err := fetcher.Fetch(context.Background(), dest,
		[]string{"hooks", "skills"}, []string{"hooks/claude", "hooks/opencode"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got := readAllFiles(t, dest)
	want := []string{
		"hooks/shared/codefall-block-merge-to-main.sh",
		"skills/x/SKILL.md",
	}

	if len(got) != len(want) {
		t.Errorf("dest holds %q, want %q", got, want)
	}

	if len(installed.Files) != len(want) {
		t.Errorf("installed = %q, want the same %d paths", installed.Files, len(want))
	}

	// The hook definitions invoke the script by path, so it has to land runnable.
	info, err := os.Stat(filepath.Join(dest, "hooks/shared/codefall-block-merge-to-main.sh"))
	if err != nil {
		t.Fatalf("stat the copied script: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("copied script mode = %o, want an executable bit set", info.Mode().Perm())
	}
}

// The install copies three named subtrees rather than the tree whole, and each file keeps the path
// it has in the tree — which is what puts hooks/shared/ and shared/ under .codefall/ and leaves the
// hook definitions, the maintainer documents, and everything else the tree carries behind.
func TestEmbeddedExtensionFetcherCopiesOnlyTheNamedSubtrees(t *testing.T) {
	src := fstest.MapFS{
		"hooks/claude/hooks.json":                      &fstest.MapFile{Data: []byte(`{"hooks":{}}`)},
		"hooks/shared/codefall-block-merge-to-main.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"shared/preflight.sh":                          &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"skills/x/SKILL.md":                            &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
		"README.md":                                    &fstest.MapFile{Data: []byte("# extension\n")},
		"AGENTS.md":                                    &fstest.MapFile{Data: []byte("# rules\n")},
		"docs/ROADMAP.md":                              &fstest.MapFile{Data: []byte("road")},
	}
	dest := t.TempDir()

	installed, err := NewEmbeddedExtensionFetcher(src, nil).
		Fetch(context.Background(), dest, []string{"hooks/shared", "shared"}, nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	want := []string{"hooks/shared/codefall-block-merge-to-main.sh", "shared/preflight.sh"}

	if !slices.Equal(installed.Files, want) {
		t.Errorf("installed = %q, want %q", installed.Files, want)
	}

	if got := readAllFiles(t, dest); len(got) != len(want) {
		t.Errorf("dest holds %q, want only %q", got, want)
	}
}

// The rule that leaves each skill's NOTES.md behind cannot name a path: there is one beside every
// skill. An exclusion with no slash in it is a file name, matched wherever it sits.
func TestEmbeddedExtensionFetcherExcludesAFileNameWhereverItSits(t *testing.T) {
	src := fstest.MapFS{
		"skills/AGENTS.md":        &fstest.MapFile{Data: []byte("# rules\n")},
		"skills/x/SKILL.md":       &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
		"skills/x/NOTES.md":       &fstest.MapFile{Data: []byte("lineage")},
		"skills/y/SKILL.md":       &fstest.MapFile{Data: []byte("---\nname: y\n---\n")},
		"skills/y/NOTES.md":       &fstest.MapFile{Data: []byte("lineage")},
		"skills/y/reference/a.md": &fstest.MapFile{Data: []byte("read me")},
	}
	dest := t.TempDir()

	installed, err := NewEmbeddedExtensionFetcher(src, nil).Fetch(context.Background(), dest,
		[]string{"skills"}, []string{"skills/AGENTS.md", "NOTES.md"})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	want := []string{"skills/x/SKILL.md", "skills/y/SKILL.md", "skills/y/reference/a.md"}

	if !slices.Equal(installed.Files, want) {
		t.Errorf("installed = %q, want %q", installed.Files, want)
	}

	if got := readAllFiles(t, dest); len(got) != len(want) {
		t.Errorf("dest holds %q, want only %q", got, want)
	}
}

// A copy reports the files it had to write: every file on a first copy, none on a second over an
// unchanged tree, and on a later one only the file that was deleted and the file that was edited.
// A file already holding the tree's bytes is not written again, so its modification time stays.
func TestEmbeddedExtensionFetcherReportsOnlyWhatDiffered(t *testing.T) {
	src := fstest.MapFS{
		"shared/preflight.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"skills/x/SKILL.md":   &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
		"skills/y/SKILL.md":   &fstest.MapFile{Data: []byte("---\nname: y\n---\n")},
	}
	fetcher := NewEmbeddedExtensionFetcher(src, nil)
	dest := t.TempDir()
	all := []string{"shared/preflight.sh", "skills/x/SKILL.md", "skills/y/SKILL.md"}

	first, err := fetcher.Fetch(context.Background(), dest, []string{"shared", "skills"}, nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if !slices.Equal(first.Changed, all) {
		t.Errorf("first copy changed %q, want every file %q", first.Changed, all)
	}

	untouched := filepath.Join(dest, "skills/y/SKILL.md")
	past := time.Now().Add(-time.Hour).Truncate(time.Second)

	if err := os.Chtimes(untouched, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	second, err := fetcher.Fetch(context.Background(), dest, []string{"shared", "skills"}, nil)
	if err != nil {
		t.Fatalf("Fetch again: %v", err)
	}

	if len(second.Changed) != 0 || !slices.Equal(second.Files, all) {
		t.Errorf("second copy = %+v, want every file installed and none changed", second)
	}

	if err := os.Remove(filepath.Join(dest, "shared/preflight.sh")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dest, "skills/x/SKILL.md"), []byte("edited\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	third, err := fetcher.Fetch(context.Background(), dest, []string{"shared", "skills"}, nil)
	if err != nil {
		t.Fatalf("Fetch a third time: %v", err)
	}

	if want := []string{"shared/preflight.sh", "skills/x/SKILL.md"}; !slices.Equal(third.Changed, want) {
		t.Errorf("third copy changed %q, want %q", third.Changed, want)
	}

	if got, err := os.ReadFile(filepath.Join(dest, "skills/x/SKILL.md")); err != nil || string(got) != "---\nname: x\n---\n" {
		t.Errorf("edited SKILL.md = %q, %v, want the tree's bytes back", got, err)
	}

	info, err := os.Stat(untouched)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if !info.ModTime().Equal(past) {
		t.Errorf("unchanged file modified at %v, want it left at %v", info.ModTime(), past)
	}
}

// readAllFiles returns the relative name of every file under dir; for the copy promise above.
func readAllFiles(t *testing.T, dir string) []string {
	t.Helper()

	var names []string
	if err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, rerr := filepath.Rel(dir, path)
			if rerr != nil {
				return rerr
			}

			names = append(names, rel)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk dest: %v", err)
	}

	return names
}

// A cancelled context ends the walk rather than halting mid-directory.
func TestEmbeddedExtensionFetcherHonoursTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewEmbeddedExtensionFetcher(fstest.MapFS{"one.txt": &fstest.MapFile{}}, nil).
		Fetch(ctx, t.TempDir(), []string{"."}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Fetch err = %v, want context.Canceled", err)
	}
}

// Every script the tree ships is invoked by path — a hook command names the guard, and three skills
// name shared/preflight.sh — so a copied script is made runnable where it lands, whatever the
// embedded tree says about modes, which is nothing. A rerun over a script the project tightened
// leaves that decision alone.
func TestEmbeddedExtensionFetcherMakesCopiedScriptsRunnable(t *testing.T) {
	src := fstest.MapFS{
		"hooks/shared/codefall-block-merge-to-main.sh": &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"shared/preflight.sh":                          &fstest.MapFile{Data: []byte("#!/bin/bash\n")},
		"skills/x/SKILL.md":                            &fstest.MapFile{Data: []byte("---\nname: x\n---\n")},
	}
	fetcher := NewEmbeddedExtensionFetcher(src, nil)
	dest := t.TempDir()

	if _, err := fetcher.Fetch(context.Background(), dest, []string{"."}, nil); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	for _, script := range []string{"hooks/shared/codefall-block-merge-to-main.sh", "shared/preflight.sh"} {
		if mode := modeOf(t, filepath.Join(dest, script)); mode&0o100 == 0 {
			t.Errorf("%s = %v, want its owner able to run it", script, mode)
		}
	}

	if mode := modeOf(t, filepath.Join(dest, "skills/x/SKILL.md")); mode&0o111 != 0 {
		t.Errorf("SKILL.md = %v, want a document left unexecutable", mode)
	}

	tightened := filepath.Join(dest, "hooks/shared/codefall-block-merge-to-main.sh")
	if err := os.Chmod(tightened, 0o700); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	if _, err := fetcher.Fetch(context.Background(), dest, []string{"."}, nil); err != nil {
		t.Fatalf("Fetch again: %v", err)
	}

	if mode := modeOf(t, tightened); mode != 0o700 {
		t.Errorf("mode after a rerun = %v, want the 0700 the project chose", mode)
	}
}

func modeOf(t *testing.T, path string) fs.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat %s: %v", path, err)
	}

	return info.Mode().Perm()
}
