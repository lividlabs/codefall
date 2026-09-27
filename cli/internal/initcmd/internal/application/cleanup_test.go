package application

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// upgrading is a settled project with a finished run on record: the manifest, and every file it
// lists present on disk, so an emptied directory can be told from one that still holds something.
func upgrading(manifest string, present ...string) *fakeFileSystem {
	files := settled("")
	files.files[manifestFull] = []byte(manifest)

	for _, file := range present {
		files.files[filepath.Join(workingDir, file)] = []byte("…")
	}

	return files
}

// cleanupResult is the cleanup step's result, wherever the run put it.
func cleanupResult(t *testing.T, report domain.Report) domain.StepResult {
	t.Helper()

	for _, result := range report.Results() {
		if result.Step.ID == domain.CleanupStep.ID {
			return result
		}
	}

	t.Fatalf("Results() = %+v, want a cleanup step", report.Results())

	return domain.StepResult{}
}

// A file the previous run recorded and this run did not write is removed, and the directory it
// leaves empty goes with it. A whole skill directory that this run wrote nothing into is reported
// once, as no longer shipped; a shared file is reported on its own.
func TestCleanupRemovesWhatThePreviousInstallWroteAndThisOneDidNot(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"claude": {"version": "v0.18.0", "files": [
    ".claude/skills/design/SKILL.md", ".claude/skills/old/SKILL.md", ".claude/skills/old/reference/notes.md"]}},
  "shared": {"version": "v0.18.0", "files": [
    ".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh", ".codefall/shared/retired.md"]}
}`,
		".claude/skills/design/SKILL.md", ".claude/skills/old/SKILL.md", ".claude/skills/old/reference/notes.md",
		".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh", ".codefall/shared/retired.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude)))

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "removed .claude/skills/old/ (no longer shipped) and .codefall/shared/retired.md (no longer shipped)"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	// The two files, then the directories they emptied, up to but never including the skills
	// directory itself; the shared file's directory still holds preflight.sh and stays.
	wantRemoved := []string{
		filepath.Join(workingDir, ".claude/skills/old/SKILL.md"),
		filepath.Join(workingDir, ".claude/skills/old/reference/notes.md"),
		filepath.Join(workingDir, ".claude/skills/old/reference"),
		filepath.Join(workingDir, ".claude/skills/old"),
		filepath.Join(workingDir, ".codefall/shared/retired.md"),
	}
	if !slices.Equal(files.removed, wantRemoved) {
		t.Errorf("removed %q, want %q", files.removed, wantRemoved)
	}

	if _, still := files.files[filepath.Join(workingDir, ".claude/skills/design/SKILL.md")]; !still {
		t.Error("the skill this run wrote is gone, want it left alone")
	}
}

// A skill directory the tree renamed is reported as the rename, so a reader knows the verb is still
// there under its new name rather than gone.
func TestCleanupNamesARenamedSkillAsARename(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"codex": {"version": "v0.18.0", "files": [".agents/skills/design/SKILL.md", ".agents/skills/graft/SKILL.md"]}},
  "shared": {"version": "v0.18.0", "files": [".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh"]}
}`, ".agents/skills/design/SKILL.md", ".agents/skills/graft/SKILL.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Codex)))

	if want := "removed .agents/skills/graft/ (renamed to codefall-graft)"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// A file dropped from a skill this run still ships is one file, named on its own: the skill survives.
func TestCleanupNamesAFileDroppedFromASurvivingSkill(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"claude": {"version": "v0.18.0", "files": [".claude/skills/design/SKILL.md", ".claude/skills/design/reference/old.md"]}},
  "shared": {"version": "v0.18.0", "files": [".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh"]}
}`, ".claude/skills/design/SKILL.md", ".claude/skills/design/reference/old.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude)))

	if want := "removed .claude/skills/design/reference/old.md (no longer shipped)"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	// reference/ emptied and went; design/ still holds SKILL.md and stays.
	wantRemoved := []string{
		filepath.Join(workingDir, ".claude/skills/design/reference/old.md"),
		filepath.Join(workingDir, ".claude/skills/design/reference"),
	}
	if !slices.Equal(files.removed, wantRemoved) {
		t.Errorf("removed %q, want %q", files.removed, wantRemoved)
	}
}

// A record with nothing to compare removes nothing and says which record was empty: an entry written
// before file lists existed, and a .codefall/ entry no run has recorded yet.
func TestCleanupRemovesNothingWhenTheRecordListsNoFiles(t *testing.T) {
	files := upgrading(`{"harnesses": {"claude": {"version": "v0.13.0"}}}`, ".claude/skills/old/SKILL.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude)))

	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	want := "nothing to remove; the record for claude lists no files; no earlier install is recorded for " +
		".codefall/, so nothing was compared there"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if len(files.removed) != 0 {
		t.Errorf("removed %q, want nothing", files.removed)
	}
}

// Only the harnesses this run installs for are compared. A harness the settings have dropped keeps
// what codefall wrote for it, which doctor reports; and a harness with no record is a first install
// for it, with nothing to remove.
func TestCleanupComparesOnlyTheHarnessesThisRunInstallsFor(t *testing.T) {
	files := upgrading(`{
  "harnesses": {
    "claude": {"version": "v0.18.0", "files": [".claude/skills/design/SKILL.md", ".claude/skills/old/SKILL.md"]},
    "codex": {"version": "v0.18.0", "files": [".agents/skills/design/SKILL.md", ".agents/skills/old/SKILL.md"]}},
  "shared": {"version": "v0.18.0", "files": [".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh"]}
}`, ".claude/skills/design/SKILL.md", ".claude/skills/old/SKILL.md", ".agents/skills/design/SKILL.md", ".agents/skills/old/SKILL.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Codex, harness.Muse)))

	want := "removed .agents/skills/old/ (no longer shipped); no earlier install is recorded for muse, so nothing was compared there"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if _, still := files.files[filepath.Join(workingDir, ".claude/skills/old/SKILL.md")]; !still {
		t.Error("a file recorded for a harness this run is not for was removed, want it left alone")
	}
}

// A manifest is a file a person can edit. A path that climbs out of the install directory, or that is
// not under one at all, is not something codefall wrote whatever the record says, and is left alone.
func TestCleanupRemovesNothingOutsideTheInstallDirectories(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"claude": {"version": "v0.18.0", "files": [
    ".claude/skills/design/SKILL.md", "../secrets.txt", "/etc/hosts", "src/main.go", ".agents/skills/x/SKILL.md"]}},
  "shared": {"version": "v0.18.0", "files": [".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh", ".codefall/../README.md"]}
}`, ".claude/skills/design/SKILL.md", "src/main.go", "README.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude)))

	if result.Outcome != domain.OutcomeSkipped || result.Detail != "nothing to remove" {
		t.Errorf("result = %+v, want nothing removed", result)
	}

	if len(files.removed) != 0 {
		t.Errorf("removed %q, want nothing", files.removed)
	}
}

// A first run has no record to compare and performs no cleanup: the report holds the seven steps it
// always held.
func TestAFirstRunHasNoCleanupStep(t *testing.T) {
	report := runFor(t, settled(""), newFakeExtensionSource(), requestFor(harness.Claude))

	for _, result := range report.Results() {
		if result.Step.ID == domain.CleanupStep.ID {
			t.Fatalf("Results() = %+v, want no cleanup step on a first run", report.Results())
		}
	}
}
