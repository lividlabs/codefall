package presentation

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
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
func TestUpgradeCommandIsCurrentOnlyForTheHarnessItInstalled(t *testing.T) {
	for _, tc := range []struct {
		name        string
		installed   application.Installation
		args        []string
		wantCurrent bool
	}{
		{
			name:        "the same harness at the same version",
			installed:   application.Installation{Versions: map[string]string{harness.Claude: buildinfo.Version()}},
			wantCurrent: true,
		},
		{
			name:      "another harness at the same version",
			installed: application.Installation{Versions: map[string]string{harness.Claude: buildinfo.Version()}},
			args:      []string{"--harness", harness.Agy},
		},
		{
			// The flag names a harness the settings do not, so the run has that to record, however
			// current the install for it is.
			name: "a recorded harness the settings do not name",
			installed: application.Installation{Versions: map[string]string{
				harness.Claude: buildinfo.Version(), harness.Codex: buildinfo.Version()}},
			args: []string{"--harness", harness.Codex},
		},
		{
			name:      "the same harness at an older version",
			installed: application.Installation{Versions: map[string]string{harness.Claude: "v0.1.0"}},
			args:      []string{"--yes"},
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

			// Every case runs the use case; what differs is whether it is told the install is current
			// and so skips Beads and prints only what changed.
			if !initialize.ran {
				t.Fatalf("ran = false, want every upgrade to run\n%s", out)
			}

			if initialize.got.Current != tc.wantCurrent {
				t.Errorf("Current = %v, want %v\n%s", initialize.got.Current, tc.wantCurrent, out)
			}
		})
	}
}

// A current install runs the repair steps, and a step that put something back is printed and closes
// with what to run next. The steps that found nothing to do are not printed: the one line that
// matters would be lost among them.
func TestUpgradeCommandReportsWhatARunOverACurrentInstallRepaired(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})
	initialize.report = domain.NewReport(
		domain.SettingsStep.Skipped(".codefall/settings.json already exists"),
		domain.HookStep.Skipped("codefall's hooks are already in .claude/settings.json"),
		domain.IgnoreStep.Done("added .codefall/user.json to .gitignore"),
	)

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	want := "✓ added .codefall/user.json to .gitignore\n" + nextStep + "\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// A current install whose steps all found their work done, the extension copy included, is up to
// date, and that is the whole of what the run says.
func TestUpgradeCommandSaysACurrentInstallWithNothingMissingIsUpToDate(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})
	initialize.report = domain.NewReport(
		domain.SettingsStep.Skipped(".codefall/settings.json already exists"),
		domain.IgnoreStep.Skipped(".gitignore already names .codefall/user.json"),
	)

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if want := "already up to date with " + buildinfo.Version() + "\n"; out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// A rerun installs for what the project already chose, because its settings record them. The flag is
// needed the first time, and afterwards only to add a harness — alongside the recorded ones, never in
// their place.
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

	if _, err := runUpgradeCommand(t, initialize, "--harness", harness.Claude); err != nil {
		t.Fatalf("Execute with --harness: %v", err)
	}

	if want := []string{harness.Codex, harness.Muse, harness.Claude}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want the flag's harness added to the settings', %q", initialize.got.Harnesses, want)
	}
}

// The breaking changes between the recorded version and the binary's are printed before anything
// runs, grouped by release, and --yes carries on past them. A range that could not be determined is
// said in one line, so silence never reads as none.
func TestUpgradeCommandPrintsTheBreakingChangesBeforeItRuns(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{Versions: map[string]string{harness.Claude: "0.16.0"}})
	initialize.breaking = mo.Some([]application.BreakingRelease{
		{Version: "0.17.0", Notes: []string{"**skills:** conceptualize is now envision", "**skills:** the Beads section changed"}},
		{Version: "0.19.0", Notes: []string{"**cli:** harnesses are named for their binaries"}},
	})
	initialize.report = domain.NewReport(domain.SettingsStep.Skipped(".codefall/settings.json already exists"))

	out, err := runUpgradeCommand(t, initialize, "--yes")
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	want := "Breaking changes in 0.17.0\n" +
		"  • **skills:** conceptualize is now envision\n" +
		"  • **skills:** the Beads section changed\n" +
		"Breaking changes in 0.19.0\n" +
		"  • **cli:** harnesses are named for their binaries\n" +
		"- .codefall/settings.json already exists\n" +
		nextStep + "\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}

	if !initialize.ran {
		t.Error("ran = false, want --yes to carry on past the breaking changes")
	}
}

func TestUpgradeCommandSaysWhenTheBreakingChangesCouldNotBeDetermined(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{Versions: map[string]string{harness.Claude: "0.1.0-dev"}})
	initialize.report = domain.NewReport()

	out, err := runUpgradeCommand(t, initialize, "--yes")
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if !strings.Contains(out, "could not be determined") {
		t.Errorf("output = %q, want it to say the range could not be determined", out)
	}

	if strings.Contains(out, "Breaking changes in") {
		t.Errorf("output = %q, want no release listed", out)
	}
}

// A run over a current install prints nothing about breaking changes: there is no version
// moving for them to be about.
func TestUpgradeCommandPrintsNoBreakingChangesOnACurrentInstall(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.manifest = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{Versions: map[string]string{harness.Claude: buildinfo.Version()}})
	initialize.breaking = mo.Some([]application.BreakingRelease{{Version: "9.9.9", Notes: []string{"never printed"}}})

	out, err := runUpgradeCommand(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if strings.Contains(out, "never printed") || strings.Contains(out, "could not be determined") {
		t.Errorf("output = %q, want nothing about breaking changes on a current install", out)
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
func TestUpgradeCommandIsAFullRunWhileTheTestingRootIsUndeclared(t *testing.T) {
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

	if !initialize.ran || initialize.got.Current {
		t.Errorf("ran = %v, Current = %v, want a full run to declare the root\n%s",
			initialize.ran, initialize.got.Current, out)
	}
}

// An install that is current in every other way is still work when its files record a harness under
// the spelling it had before it was named for its binary: doctor's remedy for that is this command,
// and the run is what rewrites it.
func TestUpgradeCommandIsAFullRunWhileAFormerHarnessNameIsRecorded(t *testing.T) {
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

	if !initialize.ran || initialize.got.Current {
		t.Errorf("ran = %v, Current = %v, want a full run to rewrite the old name\n%s",
			initialize.ran, initialize.got.Current, out)
	}
}
