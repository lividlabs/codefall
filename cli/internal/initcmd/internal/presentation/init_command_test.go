package presentation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

type fakeInitialize struct {
	report     domain.Report
	err        error
	exists     bool
	existsErr  error
	suggestion mo.Option[string]
	root       mo.Option[string]
	installed  mo.Option[application.Installation]
	harnesses  mo.Option[[]string]
	testDir    mo.Option[string]
	formers    []string

	got       application.Request
	ran       bool
	suggested bool
}

func (f *fakeInitialize) Run(
	_ context.Context, request application.Request, observer application.Observer,
) (domain.Report, error) {
	f.got, f.ran = request, true

	for _, result := range f.report.Results() {
		observer.StepStarted(result.Step)
		observer.StepFinished(result)
	}

	return f.report, f.err
}

func (f *fakeInitialize) SettingsExist(string) (bool, error) {
	return f.exists, f.existsErr
}

// Installed on the fake reports nothing applied before unless a test says otherwise — to the gate
// that reads first install, so the upgrade confirmation is skipped and the survey is reached only
// through the settings-exists logic.
func (f *fakeInitialize) Installed(string) (mo.Option[application.Installation], error) {
	return f.installed, nil
}

// ChosenHarnesses reports what a settled project's settings record. None is the answer for a project
// with no settings, and for one whose settings predate the field.
func (f *fakeInitialize) ChosenHarnesses(string) (mo.Option[[]string], error) {
	return f.harnesses, nil
}

// DeclaredTestDir reports the testing root a settled project's settings declare. None is the answer
// for a project with no settings, and for one whose settings predate the block.
func (f *fakeInitialize) DeclaredTestDir(string) (mo.Option[string], error) {
	return f.testDir, nil
}

// FormerHarnessNames reports the old harness spellings a settled project's files carry. None is the
// answer unless a test says otherwise.
func (f *fakeInitialize) FormerHarnessNames(string) ([]string, error) {
	return f.formers, nil
}

func (f *fakeInitialize) SuggestIssuesRepo(context.Context, string) mo.Option[string] {
	f.suggested = true

	return f.suggestion
}

func (f *fakeInitialize) RepositoryRoot(context.Context, string) mo.Option[string] {
	return f.root
}

func newFakeInitialize() *fakeInitialize {
	return &fakeInitialize{
		report:     domain.NewReport(domain.SettingsStep.Done("wrote .codefall/settings.json (tracker: beads)")),
		suggestion: mo.None[string](),
		root:       mo.None[string](),
		installed:  mo.None[application.Installation](),
		harnesses:  mo.None[[]string](),
		testDir:    mo.None[string](),
	}
}

// run executes the command against a fake use case and returns everything it wrote.
//
// stdin is forced to answer "not a terminal", which is the path a script, CI, and an agent take —
// and the only one a test can take, because a form needs a terminal (ADR-002).
//
// CLICOLOR_FORCE and TTY_FORCE are cleared so a developer's forced-colour environment cannot leak
// ANSI into the buffer: colorprofile honours both regardless of whether the writer is a terminal.
func run(t *testing.T, initialize InitializeUseCase, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")

	interactive(t, false)

	var out bytes.Buffer

	cmd := NewInitCommand(initialize)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	// Fang sets both on the root in production, because it renders errors and usage itself.
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()

	return out.String(), err
}

// interactive decides what the command believes about stdin for the length of one test.
func interactive(t *testing.T, yes bool) {
	t.Helper()

	previous := stdinIsTerminal
	stdinIsTerminal = func() bool { return yes }

	t.Cleanup(func() { stdinIsTerminal = previous })
}

func TestInitCommandPassesTheFlagsToTheUseCase(t *testing.T) {
	initialize := newFakeInitialize()

	if _, err := run(t, initialize,
		"--tracker", "github",
		"--issues-repo", "lividlabs/codefall-cli",
		"--issues-project", "3",
		"--harness", "claude",
		"--test-dir", "e2e",
		"--force",
	); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	want := application.Request{
		Dir:           dir,
		Tracker:       settings.TrackerGitHub,
		IssuesRepo:    mo.Some("lividlabs/codefall-cli"),
		IssuesProject: mo.Some(3),
		Harnesses:     []string{harness.Claude},
		TestDir:       "e2e",
		CLIVersion:    buildinfo.Version(),
		Force:         true,
	}

	if !reflect.DeepEqual(initialize.got, want) {
		t.Errorf("request = %+v, want %+v", initialize.got, want)
	}

	// Every answer was on the command line, so there was nothing to ask gh either.
	if initialize.suggested {
		t.Error("asked gh for a repository, want it not asked when --issues-repo was given")
	}
}

func TestInitCommandLeavesTheOptionalValuesAbsent(t *testing.T) {
	initialize := newFakeInitialize()

	if _, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude,
		"--test-dir", settings.DefaultTestDir); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []string{harness.Claude}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
	}

	if initialize.got.IssuesRepo.IsPresent() || initialize.got.IssuesProject.IsPresent() {
		t.Errorf("request = %+v, want the GitHub values absent", initialize.got)
	}

	if initialize.got.Force {
		t.Error("Force = true, want false without the flag")
	}

	// Beads needs no repository, so gh is not asked about one.
	if initialize.suggested {
		t.Error("asked gh for a repository, want it not asked when the tracker is beads")
	}
}

func TestInitCommandRejects(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "an unknown tracker",
			args: []string{"--tracker", "jira"},
			want: `unknown tracker "jira" (known trackers: beads, github)`,
		},
		{
			name: "a harness codefall cannot set up yet",
			args: []string{"--tracker", "beads", "--harness", "cursor"},
			want: `harness "cursor" is not supported yet (supported: agy, claude, codex, muse, opencode)`,
		},
		{
			name: "a repository that is not owner/name",
			args: []string{"--tracker", "github", "--issues-repo", "codefall-cli"},
			want: "owner/name",
		},
		{
			name: "a project number below one",
			args: []string{"--tracker", "github", "--issues-repo", "owner/name", "--issues-project", "0"},
			want: "the --issues-project flag must be a positive integer, not 0",
		},
		{
			name: "a repository on a tracker that has no use for one",
			args: []string{"--harness", harness.Claude, "--tracker", "beads", "--test-dir", "testing",
				"--issues-repo", "owner/name"},
			want: "the --issues-repo flag is only used with --tracker github",
		},
		{
			name: "a project number on a tracker that has no use for one",
			args: []string{"--harness", harness.Claude, "--tracker", "beads", "--test-dir", "testing",
				"--issues-project", "3"},
			want: "the --issues-project flag is only used with --tracker github",
		},
		{
			name: "a location that is neither here nor root",
			args: []string{"--tracker", "beads", "--location", "elsewhere"},
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

			out, err := run(t, initialize, tc.args...)
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

// Without a terminal the command never waits for an answer that cannot arrive: it names the flag
// that would have supplied it (ADR-002).
func TestInitCommandWithoutATerminalNamesTheMissingFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			// The harnesses are asked for first, so a run that names none hears about that before
			// anything else: nothing the run installs can be placed without them.
			name: "no harness",
			args: nil,
			want: "missing --harness (stdin is not a terminal)",
		},
		{
			name: "no tracker",
			args: []string{"--harness", harness.Claude},
			want: "missing --tracker (stdin is not a terminal)",
		},
		{
			name: "github without a repository",
			args: []string{"--harness", harness.Claude, "--tracker", "github"},
			want: "missing --issues-repo (stdin is not a terminal)",
		},
		{
			// The testing root has a default, and the default is what the survey offers rather than
			// what a script gets: where a project's cases live is a decision the project makes.
			name: "no testing root",
			args: []string{"--harness", harness.Claude, "--tracker", "beads"},
			want: "missing --test-dir (stdin is not a terminal)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialize := newFakeInitialize()

			out, err := run(t, initialize, tc.args...)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("Execute error = %v, want %q\n%s", err, tc.want, out)
			}

			if initialize.ran {
				t.Error("the use case ran, want the command to stop before it")
			}
		})
	}
}

// A repository gh already knows is an answer, not a question: without a terminal it stands in for
// the flag rather than failing the run.
func TestInitCommandUsesTheRepositoryGHSuggests(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.suggestion = mo.Some("lividlabs/codefall-cli")

	if _, err := run(t, initialize, "--harness", harness.Claude, "--tracker", "github",
		"--test-dir", settings.DefaultTestDir); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got, ok := initialize.got.IssuesRepo.Get(); !ok || got != "lividlabs/codefall-cli" {
		t.Errorf("IssuesRepo = %v, want Some(%q)", initialize.got.IssuesRepo, "lividlabs/codefall-cli")
	}
}

// Settings that are already there and are not being rewritten are not worth asking about: the step
// will skip them whatever the answers are, so the command does not fail on a missing flag either.
func TestInitCommandAsksNothingWhenSettingsAlreadyExist(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.report = domain.NewReport(
		domain.SettingsStep.Skipped(".codefall/settings.json already exists (use --force to rewrite it)"))

	out, err := run(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if initialize.got.Tracker != "" {
		t.Errorf("Tracker = %q, want it left empty", initialize.got.Tracker)
	}

	if initialize.suggested {
		t.Error("asked gh for a repository, want nothing asked when the settings are already there")
	}

	want := "- .codefall/settings.json already exists (use --force to rewrite it)\n" + nextStep + "\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// --force is what turns the questions back on, because the file it would skip is going to be
// rewritten.
func TestInitCommandStillAsksWhenForcingOverExistingSettings(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true

	// The harnesses are asked for first, so that is the flag a forced run with no answers is told
	// about — the same question order the survey puts them in.
	out, err := run(t, initialize, "--force")
	if err == nil || err.Error() != "missing --harness (stdin is not a terminal)" {
		t.Fatalf("Execute error = %v, want the missing flag\n%s", err, out)
	}
}

func TestInitCommandPrintsALineForEachFinishedStepAndWhatToRunNext(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.report = domain.NewReport(
		domain.SettingsStep.Done("wrote .codefall/settings.json (tracker: beads)"),
		domain.ExtensionStep.Skipped(
			"the codefall marketplace is declared and codefall@codefall enabled in .claude/settings.json"),
	)

	out, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude,
		"--test-dir", settings.DefaultTestDir)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "✓ wrote .codefall/settings.json (tracker: beads)\n" +
		"- the codefall marketplace is declared and codefall@codefall enabled in .claude/settings.json\n" +
		nextStep + "\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

func TestInitCommandWrapsAUseCaseError(t *testing.T) {
	failure := errors.New("settings: no space left on device")

	initialize := newFakeInitialize()
	initialize.report = domain.NewReport()
	initialize.err = failure

	out, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude,
		"--test-dir", settings.DefaultTestDir)
	if !errors.Is(err, failure) {
		t.Fatalf("Execute error = %v, want it to wrap %v", err, failure)
	}

	if !strings.HasPrefix(err.Error(), "init: ") {
		t.Errorf("error = %q, want it to say which command failed", err)
	}

	if out != "" {
		t.Errorf("output = %q, want nothing printed", out)
	}
}

func TestInitCommandReportsSettingsItCannotRead(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.existsErr = errors.New("permission denied")

	if _, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude,
		"--test-dir", settings.DefaultTestDir); err == nil ||
		!strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Execute error = %v, want it to carry the read failure", err)
	}
}

// The marks are the only styled part of a line, and they degrade to plain text when nothing is
// reading them as a terminal.
func TestStepLine(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result domain.StepResult
		want   string
	}{
		{
			name:   "a step that did its work",
			result: domain.SettingsStep.Done("wrote .codefall/settings.json (tracker: beads)"),
			want:   "✓ wrote .codefall/settings.json (tracker: beads)",
		},
		{
			name:   "a step that found its work done",
			result: domain.SettingsStep.Skipped(".codefall/settings.json already exists"),
			want:   "- .codefall/settings.json already exists",
		},
		{
			name:   "an outcome from nowhere",
			result: domain.StepResult{Outcome: domain.Outcome(42), Detail: "nothing to say"},
			want:   "? nothing to say",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripANSI(stepLine(tc.result)); got != tc.want {
				t.Errorf("stepLine() = %q, want %q", got, tc.want)
			}

			if want := outcomeStyle(tc.result.Outcome).Render(glyph(tc.result.Outcome)); !strings.HasPrefix(
				stepLine(tc.result), want+" ") {
				t.Errorf("stepLine() = %q, want it to open with the styled mark %q", stepLine(tc.result), want)
			}
		})
	}
}

// Dimming is invisible to the command tests, because the colorprofile writer strips it before the
// buffer sees it. Assert it against the styled string.
func TestTheClosingLineIsFaint(t *testing.T) {
	const faintAttr = "\x1b[2"

	line := ui.Style(ui.ToneFaint).Render(nextStep)
	if !strings.Contains(line, faintAttr) || !strings.Contains(line, nextStep) {
		t.Errorf("the closing line = %q, want %q dimmed", line, nextStep)
	}
}

func TestValidateProject(t *testing.T) {
	for _, value := range []string{"", "   ", "1", "42"} {
		if err := validateProject(value); err != nil {
			t.Errorf("validateProject(%q) = %v, want nil", value, err)
		}
	}

	for _, value := range []string{"0", "-1", "three", "1.5"} {
		if err := validateProject(value); err == nil {
			t.Errorf("validateProject(%q) = nil, want an error", value)
		}
	}
}

// Huh v2 has no disabled option, so the two trackers that are only announced are refused here.
func TestAvailableTracker(t *testing.T) {
	for _, tracker := range settings.Trackers() {
		if err := availableTracker(tracker); err != nil {
			t.Errorf("availableTracker(%q) = %v, want nil", tracker, err)
		}
	}

	for _, tc := range []struct{ tracker, want string }{
		{trackerJira, "codefall cannot use Jira yet"},
		{trackerLinear, "codefall cannot use Linear yet"},
		{"nonsense", "unknown tracker"},
	} {
		err := availableTracker(tc.tracker)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("availableTracker(%q) = %v, want it to mention %q", tc.tracker, err, tc.want)
		}
	}
}

// The spinner itself belongs to the shared UI module and is tested there. What is left here is the
// half init owns: that a step starting renames the label, and a step finishing becomes a line.
func TestProgressObserverReportsEachStep(t *testing.T) {
	progress := &recordingProgress{}
	observer := progressObserver{progress: progress}

	observer.StepStarted(domain.SettingsStep)

	result := domain.SettingsStep.Done("wrote .codefall/settings.json (tracker: beads)")
	observer.StepFinished(result)

	if want := domain.SettingsStep.Title + "…"; len(progress.labels) != 1 || progress.labels[0] != want {
		t.Errorf("labels = %q, want %q", progress.labels, []string{want})
	}

	if want := stepLine(result); len(progress.lines) != 1 || progress.lines[0] != want {
		t.Errorf("lines = %q, want %q", progress.lines, []string{want})
	}
}

type recordingProgress struct {
	labels []string
	lines  []string
}

func (p *recordingProgress) Label(label string) { p.labels = append(p.labels, label) }

func (p *recordingProgress) Line(line string) { p.lines = append(p.lines, line) }

// stripANSI removes the escape sequences lipgloss renders, which the colorprofile writer would
// normally downsample away on its way to a non-terminal.
func stripANSI(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}

			continue
		}

		b.WriteByte(s[i])
	}

	return b.String()
}

// Below the repository root the run installs where it is told to. Without a terminal and without the
// flag there is nobody to ask, so the command names the flag and says why it is needed.
func TestInitCommandBelowTheRepositoryRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	for _, tc := range []struct {
		name    string
		args    []string
		wantDir string
		wantErr string
	}{
		{name: "here", args: []string{"--location", "here"}, wantDir: cwd},
		{name: "the root", args: []string{"--location", "root"}, wantDir: "/repo"},
		{name: "no answer", wantErr: "missing --location (stdin is not a terminal; " + cwd +
			" is below the repository root at /repo)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialize := newFakeInitialize()
			initialize.root = mo.Some("/repo")

			args := append([]string{"--tracker", "beads", "--harness", harness.Claude,
				"--test-dir", settings.DefaultTestDir}, tc.args...)

			out, err := run(t, initialize, args...)

			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("Execute error = %v, want %q\n%s", err, tc.wantErr, out)
				}

				if initialize.ran {
					t.Error("the use case ran, want the command to stop before it")
				}

				return
			}

			if err != nil {
				t.Fatalf("Execute: %v\n%s", err, out)
			}

			if initialize.got.Dir != tc.wantDir {
				t.Errorf("Dir = %q, want %q", initialize.got.Dir, tc.wantDir)
			}
		})
	}
}

// At the root there is no choice to make, so a run neither asks nor needs the flag.
func TestInitCommandAtTheRepositoryRootNeedsNoLocation(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	initialize := newFakeInitialize()

	if _, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude,
		"--test-dir", settings.DefaultTestDir); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if initialize.got.Dir != cwd {
		t.Errorf("Dir = %q, want the working directory %q", initialize.got.Dir, cwd)
	}
}

// The manifest records the version each harness was installed at, and the gate compares them one
// harness at a time: a project set up for one harness has had nothing done for the next one, whatever
// version installed it. Comparing the version alone reported "already up to date" and left the second
// harness with no skills, no hooks, and no way in short of --force.
func TestInitCommandIsANoOpOnlyForTheHarnessItInstalled(t *testing.T) {
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
			initialize.harnesses = mo.Some([]string{harness.Claude})
			initialize.testDir = mo.Some(settings.DefaultTestDir)
			initialize.installed = mo.Some(tc.installed)

			out, err := run(t, initialize, tc.args...)
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

// Several harnesses arrive either way a repeatable flag can be written, so a scripted run is not
// forced into one shape of the same answer.
func TestInitCommandTakesSeveralHarnesses(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "the flag repeated",
			args: []string{"--harness", harness.Claude, "--harness", harness.Codex},
		},
		{
			name: "one flag, the names separated by commas",
			args: []string{"--harness", harness.Claude + "," + harness.Codex},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initialize := newFakeInitialize()

			args := append([]string{"--tracker", "beads", "--test-dir", settings.DefaultTestDir}, tc.args...)

			if _, err := run(t, initialize, args...); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if want := []string{harness.Claude, harness.Codex}; !slices.Equal(initialize.got.Harnesses, want) {
				t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
			}
		})
	}
}

// A rerun installs for what the project already chose, because its settings record them. The flag is
// needed the first time, and afterwards only to add a harness.
func TestInitCommandTakesTheHarnessesFromTheSettingsOnARerun(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Codex, harness.Muse})

	if _, err := run(t, initialize); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []string{harness.Codex, harness.Muse}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
	}
}

// Settings that record no harnesses are a file written before the field existed. There is nothing to
// install for and nothing to read it from, so the run says which flag would answer it rather than
// choosing a harness on the project's behalf.
func TestInitCommandRefusesSettingsThatRecordNoHarnesses(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true

	_, err := run(t, initialize)
	if err == nil || !strings.Contains(err.Error(), "record no harnesses") {
		t.Fatalf("Execute error = %v, want it to say the settings record none", err)
	}

	if initialize.ran {
		t.Error("the use case ran, want the command to stop before it")
	}
}

// A rerun works with the testing root the project declared, so the flag is needed the first time and
// never again — and the run has no way to move a root a project's cases already sit at.
func TestInitCommandTakesTheTestingRootFromTheSettingsOnARerun(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some("packages/web/e2e")

	if _, err := run(t, initialize); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if initialize.got.TestDir != "packages/web/e2e" {
		t.Errorf("TestDir = %q, want the root the settings declare", initialize.got.TestDir)
	}
}

// A project settled before the block existed declares none, and the run still has the tree to make:
// nothing is asked, because a rerun surveys for nothing, and the use case takes the format's default.
func TestInitCommandLeavesTheTestingRootToTheUseCaseOnARerunThatDeclaresNone(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})

	if _, err := run(t, initialize); err != nil {
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
func TestInitCommandIsNotANoOpWhileTheTestingRootIsUndeclared(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})

	out, err := run(t, initialize)
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
func TestInitCommandIsNotANoOpWhileAFormerHarnessNameIsRecorded(t *testing.T) {
	initialize := newFakeInitialize()
	initialize.exists = true
	initialize.harnesses = mo.Some([]string{harness.Claude})
	initialize.testDir = mo.Some(settings.DefaultTestDir)
	initialize.installed = mo.Some(application.Installation{
		Versions: map[string]string{harness.Claude: buildinfo.Version()}})
	initialize.formers = []string{"claude-code"}

	out, err := run(t, initialize)
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if !initialize.ran {
		t.Errorf("ran = false, want the run to rewrite the old name\n%s", out)
	}
}

// The flag takes a former spelling as well, and the run is asked for the harness under the name it has
// now: a script written before the rename keeps working, and nothing downstream sees the old name.
func TestInitCommandReadsAFormerHarnessNameAsTheCurrentOne(t *testing.T) {
	initialize := newFakeInitialize()

	args := []string{"--tracker", "beads", "--test-dir", settings.DefaultTestDir,
		"--harness", "claude-code,antigravity"}
	if _, err := run(t, initialize, args...); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []string{harness.Claude, harness.Agy}; !slices.Equal(initialize.got.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", initialize.got.Harnesses, want)
	}
}

// A root that is not a relative path inside the project is refused where the person typed it, in
// terms of the flag rather than of the settings field it would have filled.
func TestInitCommandRejectsATestingRootOutsideTheProject(t *testing.T) {
	for _, dir := range []string{"/srv/testing", "../testing"} {
		initialize := newFakeInitialize()

		_, err := run(t, initialize, "--tracker", "beads", "--harness", harness.Claude, "--test-dir", dir)
		if err == nil || !strings.Contains(err.Error(), "relative path inside the project") {
			t.Errorf("Execute error for %q = %v, want it to refuse the path", dir, err)
		}

		if initialize.ran {
			t.Errorf("the use case ran for %q, want the command to stop at the flag", dir)
		}
	}
}

// Every harness the survey offers is written the way its makers write it. A harness added to the
// shared module and not labelled here fails this test rather than reaching a person as a bare
// identifier.
func TestEveryHarnessHasASurveyLabel(t *testing.T) {
	for _, name := range harness.All() {
		if label, ok := harnessLabels[name]; !ok || label == "" {
			t.Errorf("harnessLabels[%q] = %q, want a label", name, label)
		}
	}

	if got := harnessLabel("cursor"); got != "cursor" {
		t.Errorf("harnessLabel of an unlabelled harness = %q, want %q", got, "cursor")
	}
}

// Huh accepts an empty multi-select, so the field's own validation is what makes the question
// unskippable — and leaving it unanswered is the one thing the survey must not allow, because there
// is nothing to install for.
func TestAtLeastOneHarness(t *testing.T) {
	if err := atLeastOneHarness(nil); err == nil {
		t.Error("atLeastOneHarness(nil) = nil, want an error")
	}

	if err := atLeastOneHarness([]string{harness.Claude}); err != nil {
		t.Errorf("atLeastOneHarness of one harness = %v, want nil", err)
	}
}
