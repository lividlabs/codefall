package application

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// extensionRequest is a run that has nothing to do but install the extension: the settings are already
// there, so the first step skips and what the test watches is the second.
func extensionRequest() Request {
	return Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}}
}

// settled is a file system whose .codefall/settings.json is already written, with whatever
// .claude/settings.json the test wants beside it.
func settled(claudeSettings string) *fakeFileSystem {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte("{}\n")

	if claudeSettings != "" {
		files.files[claudeFull] = []byte(claudeSettings)
	}

	return files
}

// The step copies twice: the skills into the harness's own skills directory, which for Claude Code
// is .claude/, and the files a path reaches into .codefall/, once whatever harnesses the run is for.
func TestExtensionStepCopiesIntoTheHarnessSkillsDirectory(t *testing.T) {
	fetcher := newFakeExtensionSource()

	report, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(t.Context(), extensionRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := report.Results()[1]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "installed codefall's skills into .claude/ and its shared files into .codefall/"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if len(fetcher.calls) != 2 {
		t.Fatalf("fetcher calls = %+v, want one fetch per destination", fetcher.calls)
	}

	if got, want := fetcher.calls[0].dir, filepath.Join(workingDir, ".claude"); got != want {
		t.Errorf("the skills fetch went to %q, want %q", got, want)
	}

	if got, want := fetcher.calls[1].dir, filepath.Join(workingDir, ".codefall"); got != want {
		t.Errorf("the shared fetch went to %q, want %q", got, want)
	}

	// The per-harness definitions are the hook step's, read from the embedded tree rather than
	// copied, and the maintainer documents belong to this repository. Both are left behind by the
	// step naming the three subtrees it installs: a step that asked for the tree whole would write
	// every harness's definition into every project's skills directory and record them in the
	// manifest, which is the stray-definition problem the unified hooks change set out to remove.
	if got, want := fetcher.calls[0].sources, []string{"skills"}; !slices.Equal(got, want) {
		t.Errorf("the skills fetch asked for %q, want %q", got, want)
	}

	if got, want := fetcher.calls[1].sources, []string{"hooks/shared", "shared"}; !slices.Equal(got, want) {
		t.Errorf("the shared fetch asked for %q, want %q", got, want)
	}

	if got, want := fetcher.calls[0].exclude, []string{"skills/AGENTS.md", "NOTES.md"}; !slices.Equal(got, want) {
		t.Errorf("the skills fetch excluded %q, want the maintainer documents %q", got, want)
	}
}

// Nothing a project has no reader for is installed: the rules for changing a skill, each skill's
// lineage note, and the three documents that sit outside skills/ — the extension's own README.md and
// AGENTS.md, and docs/ROADMAP.md, which are left behind by never being named as a subtree to copy.
func TestExtensionStepInstallsNoMaintainerDocument(t *testing.T) {
	fetcher := newFakeExtensionSource()

	if _, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(
		t.Context(), extensionRequest(), nil,
	); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, call := range fetcher.calls {
		for _, source := range call.sources {
			for _, document := range []string{"README.md", "AGENTS.md", "docs"} {
				if source == document {
					t.Errorf("the fetch into %s asked for %q, which no project reads", call.dir, source)
				}
			}
		}
	}
}

// A claude harness that cannot be read runs no extension step: the report commands stay empty.
// Its companion tests for enforcement about manifest write aside — done behavior.
func TestExtensionStepStopsTheRunWhenTheCopyFails(t *testing.T) {
	fetcher := newFakeExtensionSource()
	fetcher.err = errors.New("disk full")

	_, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(t.Context(), extensionRequest(), nil)
	if err == nil ||
		!strings.HasPrefix(err.Error(), domain.ExtensionStep.ID+": ") ||
		!strings.Contains(err.Error(), "install the embedded extension: disk full") {
		t.Errorf("Run error = %v, want it to name the extension step and the copy that failed", err)
	}
}

// The copy into .codefall/ stops the run the same way the copy into a skills directory does: a
// project whose skills are installed and whose shared scripts are not has a guard pointing at
// nothing and three verbs that cannot run their preflight.
func TestExtensionStepStopsTheRunWhenTheSharedCopyFails(t *testing.T) {
	fetcher := newFakeExtensionSource()
	fetcher.err = errors.New("disk full")
	fetcher.failOn = "shared"

	_, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(t.Context(), extensionRequest(), nil)
	if err == nil ||
		!strings.HasPrefix(err.Error(), domain.ExtensionStep.ID+": ") ||
		!strings.Contains(err.Error(), "install codefall's shared files: disk full") {
		t.Errorf("Run error = %v, want it to name the extension step and the copy that failed", err)
	}
}

// The mechanism table is a guard: presentation refuses an unknown harness before the use case sees
// it, so this is a path a run should never take.
func TestExtensionStepRefusesAHarnessItDoesNotKnow(t *testing.T) {
	request := extensionRequest()
	request.Harnesses = []string{"aider"}

	_, err := NewInitialize(settled(""), toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil)
	if err == nil || !strings.Contains(err.Error(), `harness "aider" has no extension mechanism`) {
		t.Errorf("Run error = %v, want it to say the harness has no extension mechanism", err)
	}
}
