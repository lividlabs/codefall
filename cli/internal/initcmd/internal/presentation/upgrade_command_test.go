package presentation

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Upgrade is every run after the first, and the first is the one that writes the manifest: a project
// without one has nothing to bring current, and the command says which command it is for.
func TestUpgradeCommandRefusesAProjectWithNoManifest(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})

	_, err := runUpgradeCommand(t, initialize)
	if err == nil || !strings.Contains(err.Error(), "run codefall init first") {
		t.Fatalf("Execute error = %v, want it to name codefall init", err)
	}

	if initialize.ran {
		t.Error("the use case ran, want the command to stop before it")
	}
}

// Nothing is surveyed on an upgrade: the answers were given at init. A run with a manifest, settled
// harnesses, and a declared root, and no record of a version, does its work and prints what to run
// next, the same closing line as init.
func TestUpgradeCommandPrintsALineForEachFinishedStepAndWhatToRunNext(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.report = domain.NewReport(
		domain.SettingsStep.Skipped(".codefall/settings.json already exists"),
		domain.ExtensionStep.Done("installed the codefall extension for claude"),
	)

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "- .codefall/settings.json already exists\n" +
		"✓ installed the codefall extension for claude\n" +
		nextStep + "\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}

	if initialize.suggested {
		t.Error("asked gh for a repository, want nothing asked on an upgrade")
	}
}

func TestUpgradeCommandWrapsAUseCaseError(t *testing.T) {
	failure := errors.New("extension: no space left on device")

	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.report = domain.NewReport()
	initialize.err = failure

	_, err := runUpgradeCommand(t, initialize)
	if !errors.Is(err, failure) {
		t.Fatalf("Execute error = %v, want it to wrap %v", err, failure)
	}

	if !strings.HasPrefix(err.Error(), "upgrade: ") {
		t.Errorf("error = %q, want it to say which command failed", err)
	}
}

// A flag that init refuses is refused here in the same words, and nothing runs.
func TestUpgradeCommandRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "a harness codefall cannot set up yet",
			args: []string{"--harness", "cursor"},
			want: `harness "cursor" is not supported yet (supported: agy, claude, codex, muse, opencode)`,
		},
		{
			name: "a location that is neither here nor root",
			args: []string{"--location", "elsewhere"},
			want: `the --location flag must be here or root, not "elsewhere"`,
		},
		{
			name: "an argument",
			args: []string{"somewhere"},
			want: "unknown command",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialize := newFakeInitialize()
			initialize.exists = true
			initialize.manifest = true

			out, err := runUpgradeCommand(t, initialize, tc.args...)
			if err == nil {
				t.Fatalf("Execute = nil error, want one\n%s", out)
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}

			if initialize.ran {
				t.Error("the use case ran, want the command to stop at the flag")
			}
		})
	}
}

// The flag takes a former spelling as well, and the run is asked for the harness under the name it has
// now: a script written before the rename keeps working, and nothing downstream sees the old name.
func TestUpgradeCommandReadsAFormerHarnessNameAsTheCurrentOne(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true

	if _, err := runUpgradeCommand(t, initialize, "--harness", "claude-code,antigravity"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []string{harness.Claude, harness.Agy}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
	}
}

// The manifest records the version each harness was installed at, and the gate compares them one
// harness at a time: a project set up for one harness has had nothing done for the next one, whatever
// version installed it. Comparing the version alone reported "already up to date" and left the second
// harness with no skills, no hooks, and no way in.
func TestUpgradeCommandIsANoOpOnlyForTheHarnessItInstalled(t *testing.T) {
	for _, tc := range []struct {
		name      string
		installed application.Installation
		args      []string
		wantRun   bool
	}{
		{
			name:      "the same harness at the same version",
			installed: application.Installation{Versions: map[string]string{harness.Claude: buildinfo.Version()}},
			wantRun:   false,
		},
		{
			name:      "another harness at the same version",
			installed: application.Installation{Versions: map[string]string{harness.Claude: buildinfo.Version()}},
			args:      []string{"--harness", harness.Agy},
			wantRun:   true,
		},
		{
			// A record naming several harnesses is read per harness, so the one this run is for is
			// what decides — not whichever install happened to finish last.
			name: "one of several recorded harnesses, at the same version",
			installed: application.Installation{Versions: map[string]string{
				harness.Claude: buildinfo.Version(), harness.Codex: buildinfo.Version()}},
			args:    []string{"--harness", harness.Codex},
			wantRun: false,
		},
		{
			name:      "the same harness at an older version",
			installed: application.Installation{Versions: map[string]string{harness.Claude: "v0.1.0"}},
			args:      []string{"--yes"},
			wantRun:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialize := newFakeInitialize()
			initialize.exists = true
			initialize.manifest = true
			initialize.harnesses = mo.Some([]string{harness.Claude})
			initialize.testDir = mo.Some(settings.DefaultTestDir)
			initialize.installed = mo.Some(tc.installed)

			out, err := runUpgradeCommand(t, initialize, tc.args...)
			if err != nil {
				t.Fatalf("Execute: %v\n%s", err, out)
			}

			if initialize.ran != tc.wantRun {
				t.Errorf("ran = %v, want %v\n%s", initialize.ran, tc.wantRun, out)
			}

			if !tc.wantRun && !strings.Contains(out, "already up to date") {
				t.Errorf("output = %q, want it to report the install is current", out)
			}
		})
	}
}

// A rerun installs for what the project already chose, because its settings record them. The flag is
// needed the first time, and afterwards only to add a harness.
func TestUpgradeCommandTakesTheHarnessesFromTheSettingsOnARerun(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Codex, harness.Muse})

	if _, err := runUpgradeCommand(t, initialize); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []string{harness.Codex, harness.Muse}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
	}
}

// Settings that record no harnesses are a file written before the field existed. There is nothing to
// install for and nothing to read it from, so the run says which flag would answer it rather than
// choosing a harness on the project's behalf.
func TestUpgradeCommandRefusesSettingsThatRecordNoHarnesses(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true

	_, err := runUpgradeCommand(t, initialize)
	if err == nil || !strings.Contains(err.Error(), "record no harnesses") {
		t.Fatalf("Execute error = %v, want it to say the settings record none", err)
	}

	if initialize.ran {
		t.Error("the use case ran, want the command to stop before it")
	}
}

// A rerun works with the testing root the project declared, so the flag is needed the first time and
// never again — and the run has no way to move a root a project's cases already sit at.
func TestUpgradeCommandTakesTheTestingRootFromTheSettingsOnARerun(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some("packages/web/e2e")

	if _, err := runUpgradeCommand(t, initialize); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if initialize.got.TestDir != "packages/web/e2e" {
		t.Errorf("TestDir = %q, want the root the settings declare", initialize.got.TestDir)
	}
}

// A project settled before the block existed declares none, and the run still has the tree to make:
// nothing is asked, because a rerun surveys for nothing, and the use case takes the format's default.
func TestUpgradeCommandLeavesTheTestingRootToTheUseCaseOnARerunThatDeclaresNone(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})

	if _, err := runUpgradeCommand(t, initialize); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !initialize.ran {
		t.Fatal("the use case did not run, want a rerun to make the tree")
	}

	if initialize.got.TestDir != "" {
		t.Errorf("TestDir = %q, want it left empty for the use case's default", initialize.got.TestDir)
	}
}

// An install that is current in every other way is still work when the project has never declared a
// testing root: doctor's remedy for that is this command, so this command has to do something.
func TestUpgradeCommandIsNotANoOpWhileTheTestingRootIsUndeclared(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if !initialize.ran {
		t.Errorf("ran = false, want the run to declare the root\n%s", out)
	}
}

// An install that is current in every other way is still work when its files record a harness under
// the spelling it had before it was named for its binary: doctor's remedy for that is this command,
// and the run is what rewrites it.
func TestUpgradeCommandIsNotANoOpWhileAFormerHarnessNameIsRecorded(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})
	initialize.formers = []string{"claude-code"}

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if !initialize.ran {
		t.Errorf("ran = false, want the run to rewrite the old name\n%s", out)
	}
}
