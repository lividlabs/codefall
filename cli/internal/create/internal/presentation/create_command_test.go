package presentation

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/create/internal/application"
	"github.com/lividlabs/codefall/cli/internal/create/internal/domain"
)

type fakeCreate struct {
	scaffoldErr error
	pushErr     error

	got      application.Request
	scaffold bool
	pushed   string
}

// Scaffold makes the directory for real, because the command moves into it before running init.
func (f *fakeCreate) Scaffold(
	_ context.Context, request application.Request, observer application.Observer,
) (domain.Report, error) {
	f.got, f.scaffold = request, true

	if f.scaffoldErr != nil {
		return domain.Report{}, f.scaffoldErr
	}

	if err := os.MkdirAll(request.Dir, 0o755); err != nil {
		return domain.Report{}, err
	}

	result := domain.DirectoryStep.Done("created " + request.Dir)
	observer.StepStarted(result.Step)
	observer.StepFinished(result)

	return domain.NewReport(result), nil
}

func (f *fakeCreate) Push(_ context.Context, dir string) (domain.StepResult, error) {
	f.pushed = dir

	return domain.PushStep.Done("pushed to origin"), f.pushErr
}

// fakeInit stands in for init's command: two of its real flag names, one create adopts and one it
// leaves out, and a body that records where it ran and what it was given.
type fakeInit struct {
	command *cobra.Command
	tracker string
	err     error

	ran        bool
	dir        string
	sawTracker string
	changed    bool
}

func newFakeInit() *fakeInit {
	f := &fakeInit{}

	f.command = &cobra.Command{
		Use: "init",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f.ran = true
			f.dir, _ = os.Getwd()
			f.sawTracker = f.tracker
			f.changed = cmd.Flags().Changed("tracker")

			return f.err
		},
	}
	f.command.Flags().StringVar(&f.tracker, "tracker", "", "issue tracker")
	f.command.Flags().StringVar(new(string), "location", "", "where to install")

	return f
}

// run executes the command in a fresh working directory and returns everything it wrote, with stdin
// answering "not a terminal" — the path a script takes, and the only one a test can.
func run(t *testing.T, create CreateUseCase, init *fakeInit, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")
	t.Chdir(t.TempDir())

	previous := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }

	t.Cleanup(func() { stdinIsTerminal = previous })

	var out bytes.Buffer

	cmd := NewCreateCommand(create, init.command)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()

	return out.String(), err
}

// sameDir compares two paths after resolving links, because macOS's temporary directories are
// reached through /var and reported through /private/var.
func sameDir(t *testing.T, a, b string) bool {
	t.Helper()

	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)

	return errA == nil && errB == nil && ra == rb
}

func TestCreatePassesTheFlagsAndRunsInitInTheNewDirectory(t *testing.T) {
	create, init := &fakeCreate{}, newFakeInit()

	// An absolute target, because the command moves into the directory before it returns and the
	// working directory is no longer where it started by the time the test could read it.
	dir := filepath.Join(t.TempDir(), "app")

	out, err := run(t, create, init,
		dir, "--description", " A tool for things. ", "--remote", "git@github.com:o/app.git",
		"--tracker", "beads")
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	want := application.Request{
		Dir:         dir,
		Title:       "app",
		Description: mo.Some("A tool for things."),
		Remote:      mo.Some("git@github.com:o/app.git"),
	}
	if create.got != want {
		t.Errorf("request = %+v, want %+v", create.got, want)
	}

	if !init.ran || !sameDir(t, init.dir, dir) {
		t.Errorf("init ran = %v in %q, want it run in %q", init.ran, init.dir, dir)
	}

	// The adopted flag is init's own, so init sees both the value and that it was set.
	if init.sawTracker != "beads" || !init.changed {
		t.Errorf("init saw tracker %q (changed %v), want %q set", init.sawTracker, init.changed, "beads")
	}

	// A remote with no --push and nobody to ask is not pushed.
	if create.pushed != "" {
		t.Errorf("pushed %q, want no push", create.pushed)
	}

	for _, line := range []string{"✓ created " + dir, "Project created in " + dir} {
		if !strings.Contains(out, line) {
			t.Errorf("output does not contain %q:\n%s", line, out)
		}
	}
}

func TestCreateLeavesOutInitFlagsThatMeanNothingInANewDirectory(t *testing.T) {
	cmd := NewCreateCommand(&fakeCreate{}, newFakeInit().command)

	if cmd.Flags().Lookup("tracker") == nil {
		t.Error("create has no --tracker, want init's flag adopted")
	}

	if cmd.Flags().Lookup("location") != nil {
		t.Error("create has --location, want it left out")
	}
}

func TestCreateWithoutAnswersOrATerminalAsksNothing(t *testing.T) {
	create, init := &fakeCreate{}, newFakeInit()

	if out, err := run(t, create, init, "app"); err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if create.got.Description.IsPresent() || create.got.Remote.IsPresent() {
		t.Errorf("request = %+v, want no description and no remote", create.got)
	}
}

func TestCreatePushRequiresARemote(t *testing.T) {
	create, init := &fakeCreate{}, newFakeInit()

	_, err := run(t, create, init, "app", "--push")
	if err == nil || !strings.Contains(err.Error(), "--push") {
		t.Fatalf("error = %v, want one naming --push", err)
	}

	if create.scaffold || init.ran {
		t.Error("scaffolded or ran init, want nothing done")
	}
}

func TestCreatePushesWhenAsked(t *testing.T) {
	create, init := &fakeCreate{}, newFakeInit()

	out, err := run(t, create, init, "app", "--remote", "git@github.com:o/app.git", "--push")
	if err != nil {
		t.Fatalf("Execute: %v\n%s", err, out)
	}

	if create.pushed != create.got.Dir {
		t.Errorf("pushed %q, want %q", create.pushed, create.got.Dir)
	}

	if !strings.Contains(out, "✓ pushed to origin") {
		t.Errorf("output does not report the push:\n%s", out)
	}
}

func TestCreateStopsWhenScaffoldFails(t *testing.T) {
	create, init := &fakeCreate{scaffoldErr: errors.New("git is not on PATH")}, newFakeInit()

	_, err := run(t, create, init, "app")
	if err == nil || !strings.Contains(err.Error(), "git is not on PATH") {
		t.Fatalf("error = %v, want the scaffold's", err)
	}

	if init.ran {
		t.Error("ran init, want it not run after a failed scaffold")
	}
}

func TestCreateSaysHowToFinishWhenInitFails(t *testing.T) {
	create, init := &fakeCreate{}, newFakeInit()
	init.err = errors.New("missing --tracker (stdin is not a terminal)")

	_, err := run(t, create, init, "app", "--remote", "git@github.com:o/app.git", "--push")
	if err == nil {
		t.Fatal("Execute: want an error, got nil")
	}

	for _, want := range []string{"missing --tracker", "run codefall init there to finish"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}

	if create.pushed != "" {
		t.Error("pushed after init failed, want no push")
	}
}
