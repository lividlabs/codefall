package application

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// formerSettingsFull is the settings file under the name an earlier release wrote it.
var formerSettingsFull = filepath.Join(codefallFull, formerSettingsFile)

// settledUnderTheFormerName is a project an earlier release set up: valid settings, still declaring
// a testing root so the testing step leaves them alone, and still under the old name.
func settledUnderTheFormerName() *fakeFileSystem {
	files := newFakeFileSystem()
	files.files[formerSettingsFull] = []byte(`{
  "$schema": "` + settings.FormerSchemaIDs[1] + `",
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "test": { "dir": "testing" }
}
`)

	return files
}

// A run over a project whose settings are still under the old name moves them to the name every
// reader uses now and says so. The bytes are the file's own apart from the $schema URL, which the
// move lets the same step repoint because the URL the old release wrote is a former one, and the
// old name is gone once the run has finished.
func TestRunMovesSettingsUnderTheFormerName(t *testing.T) {
	files := settledUnderTheFormerName()

	report := runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude))

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	if want := "moved .codefall/settings.json to .codefall/project.json; " +
		"pointed $schema in .codefall/project.json at the current schema"; result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	if _, there := files.files[formerSettingsFull]; there {
		t.Errorf("%s is still there, want the run to have removed it", formerSettingsName)
	}

	if !slices.Contains(files.removed, formerSettingsFull) {
		t.Errorf("removed = %q, want it to name %s", files.removed, formerSettingsName)
	}

	want := `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "test": { "dir": "testing" }
}
`
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("project.json =\n%s\nwant\n%s", got, want)
	}
}

// A move the run cannot finish is reported as the step's failure: a project left holding both
// files is told what refused, and the next run is the one that says the old file is the person's
// to remove.
func TestTheMoveNamesTheFileItCannotRemove(t *testing.T) {
	files := settledUnderTheFormerName()
	files.errs["remove "+formerSettingsFull] = errors.New("permission denied")

	_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(
		t.Context(), requestFor(harness.Claude), nil)
	if err == nil || !strings.Contains(err.Error(),
		"remove "+formerSettingsName+": permission denied") {
		t.Errorf("Run error = %v, want it to name the file the move could not remove", err)
	}
}

// A project that has both files keeps project.json — the file this binary reads — and the run says
// the old one was left where it is, for the person to remove. Nothing is moved and nothing is
// edited: which of the two files goes is not codefall's to decide.
func TestRunKeepsProjectJsonWhenBothFilesAreThere(t *testing.T) {
	body := `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "test": { "dir": "testing" }
}
`
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(body)
	files.files[formerSettingsFull] = []byte(body)

	report := runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude))

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	if want := "left .codefall/settings.json where it is, for you to remove"; result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	if got := string(files.files[settingsFull]); got != body {
		t.Errorf("project.json =\n%s\nwant it untouched\n%s", got, body)
	}

	if got := string(files.files[formerSettingsFull]); got != body {
		t.Errorf(".codefall/settings.json =\n%s\nwant it left where it was\n%s", got, body)
	}

	if len(files.removed) != 0 {
		t.Errorf("removed = %q, want the run to have removed nothing", files.removed)
	}
}

// A first run writes .codefall/project.json and nothing under the old name: no step writes that
// name, so a project never ends up holding both files because of init.
func TestRunWritesNoSettingsUnderTheFormerName(t *testing.T) {
	files := newFakeFileSystem()

	runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude))

	if _, there := files.files[formerSettingsFull]; there {
		t.Errorf("the run wrote %s, want it to write only %s", formerSettingsName, settingsName)
	}

	if _, there := files.files[settingsFull]; !there {
		t.Errorf("the run wrote no %s, want the settings there", settingsName)
	}
}

// The reads a run makes before its steps — the harnesses, the testing root, and the old harness
// spellings — reach a project that still has only the old name, so the run that moves the file is
// built from the settings it is about to move. No other command reaches them: init has refused the
// project already.
func TestTheReadsBeforeTheStepsReachTheFormerName(t *testing.T) {
	files := newFakeFileSystem()
	files.files[formerSettingsFull] = []byte(
		`{"harnesses": ["claude-code"], "test": {"dir": "e2e"}}`)

	initialize := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges())

	harnesses, err := initialize.ChosenHarnesses(workingDir)
	if err != nil {
		t.Fatalf("ChosenHarnesses: %v", err)
	}

	// A former spelling is reported under the name the harness has now, as it is from the new name.
	if names, _ := harnesses.Get(); !slices.Equal(names, []string{harness.Claude}) {
		t.Errorf("ChosenHarnesses = %v, want [claude]", names)
	}

	declared, err := initialize.DeclaredTestDir(workingDir)
	if err != nil {
		t.Fatalf("DeclaredTestDir: %v", err)
	}

	if got := declared.OrEmpty(); got != "e2e" {
		t.Errorf("DeclaredTestDir = %q, want e2e", got)
	}

	formers, err := initialize.FormerHarnessNames(workingDir)
	if err != nil {
		t.Fatalf("FormerHarnessNames: %v", err)
	}

	if want := []string{"claude-code"}; !slices.Equal(formers, want) {
		t.Errorf("FormerHarnessNames = %q, want %q", formers, want)
	}
}
