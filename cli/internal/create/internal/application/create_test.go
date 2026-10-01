package application

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/create/internal/domain"
)

const projectDir = "/work/app"

// The commands a run gives git, keyed the way the fake runner keys them.
const (
	gitInit      = "git init"
	gitRemoteAdd = "git remote add origin git@github.com:lividlabs/app.git"
	gitAdd       = "git add -- README.md .gitignore"
	gitCommit    = "git commit --message Initial commit"
	gitPush      = "git push --set-upstream origin HEAD"
)

// --- fakes -------------------------------------------------------------------------------------

type fakeFileSystem struct {
	existing map[string]bool
	files    map[string]string
	errs     map[string]error
	made     []string
}

func newFakeFileSystem() *fakeFileSystem {
	return &fakeFileSystem{existing: map[string]bool{}, files: map[string]string{}, errs: map[string]error{}}
}

func (f *fakeFileSystem) Exists(path string) (bool, error) {
	return f.existing[path], f.errs["exists "+path]
}

func (f *fakeFileSystem) MkdirAll(path string) error {
	if err, ok := f.errs["mkdir "+path]; ok {
		return err
	}

	f.made = append(f.made, path)

	return nil
}

func (f *fakeFileSystem) WriteFile(path string, data []byte) error {
	if err, ok := f.errs["write "+path]; ok {
		return err
	}

	f.files[path] = string(data)

	return nil
}

type fakeCommandRunner struct {
	paths map[string]string
	runs  map[string]CommandResult
	errs  map[string]error
	calls []string
	dirs  []string
}

func newFakeCommandRunner() *fakeCommandRunner {
	return &fakeCommandRunner{
		paths: map[string]string{gitCommand: "/usr/bin/git"},
		runs:  map[string]CommandResult{},
		errs:  map[string]error{},
	}
}

func (r *fakeCommandRunner) LookPath(name string) mo.Option[string] {
	if path, ok := r.paths[name]; ok {
		return mo.Some(path)
	}

	return mo.None[string]()
}

func (r *fakeCommandRunner) Run(_ context.Context, dir, name string, args ...string) (CommandResult, error) {
	command := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, command)
	r.dirs = append(r.dirs, dir)

	if err, ok := r.errs[command]; ok {
		return CommandResult{}, err
	}

	return r.runs[command], nil
}

type recordingObserver struct {
	started  []string
	finished []domain.StepResult
}

func (o *recordingObserver) StepStarted(step domain.Step) { o.started = append(o.started, step.ID) }

func (o *recordingObserver) StepFinished(result domain.StepResult) {
	o.finished = append(o.finished, result)
}

func request() Request {
	return Request{
		Dir:         projectDir,
		Title:       "app",
		Description: mo.Some("A tool for things."),
		Remote:      mo.Some("git@github.com:lividlabs/app.git"),
	}
}

// --- tests -------------------------------------------------------------------------------------

func TestScaffoldRunsEveryStepInTheProjectDirectory(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()
	observer := &recordingObserver{}

	report, err := NewCreate(files, runner).Scaffold(context.Background(), request(), observer)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	if want := []string{projectDir}; !slices.Equal(files.made, want) {
		t.Errorf("made = %v, want %v", files.made, want)
	}

	// The remote is added before the commit and the files are staged by name, so nothing else in
	// the directory is swept into the first commit.
	if want := []string{gitInit, gitRemoteAdd, gitAdd, gitCommit}; !slices.Equal(runner.calls, want) {
		t.Errorf("git calls = %v, want %v", runner.calls, want)
	}

	for _, dir := range runner.dirs {
		if dir != projectDir {
			t.Errorf("git ran in %q, want %q", dir, projectDir)
		}
	}

	if got, want := files.files[filepath.Join(projectDir, "README.md")], "# app\n\nA tool for things.\n"; got != want {
		t.Errorf("README.md = %q, want %q", got, want)
	}

	if got := files.files[filepath.Join(projectDir, ".gitignore")]; got != domain.GitIgnore {
		t.Errorf(".gitignore = %q, want the stack-agnostic default", got)
	}

	wantSteps := []string{"directory", "repository", "remote", "files", "commit"}
	if !slices.Equal(observer.started, wantSteps) {
		t.Errorf("started = %v, want %v", observer.started, wantSteps)
	}

	if got := len(report.Results()); got != len(wantSteps) {
		t.Errorf("report has %d results, want %d", got, len(wantSteps))
	}
}

func TestScaffoldWithoutARemoteSkipsThatStep(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()

	req := request()
	req.Remote = mo.None[string]()

	report, err := NewCreate(files, runner).Scaffold(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	if slices.ContainsFunc(runner.calls, func(c string) bool { return strings.HasPrefix(c, "git remote") }) {
		t.Errorf("git calls = %v, want no remote added", runner.calls)
	}

	remote := report.Results()[2]
	if remote.Step != domain.RemoteStep || remote.Outcome != domain.OutcomeSkipped {
		t.Errorf("third result = %+v, want the remote step skipped", remote)
	}
}

func TestScaffoldRefusesBeforeDoingAnything(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()
	files.existing[projectDir] = true
	delete(runner.paths, gitCommand)

	_, err := NewCreate(files, runner).Scaffold(context.Background(), request(), nil)
	if err == nil {
		t.Fatal("Scaffold: want an error, got nil")
	}

	// Both problems at once, so clearing them takes one try.
	for _, want := range []string{"git is not on PATH", projectDir + " already exists"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}

	if len(files.made) > 0 || len(runner.calls) > 0 {
		t.Errorf("made %v and ran %v, want nothing done", files.made, runner.calls)
	}
}

func TestScaffoldNamesTheStepAndGitsReason(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()
	runner.runs[gitCommit] = CommandResult{
		ExitCode: 128,
		Stderr:   "Author identity unknown\n\n*** Please tell me who you are.\n\nfatal: unable to auto-detect email address\n",
	}

	_, err := NewCreate(files, runner).Scaffold(context.Background(), request(), nil)
	if err == nil {
		t.Fatal("Scaffold: want an error, got nil")
	}

	want := "commit: git commit --message Initial commit exited 128: fatal: unable to auto-detect email address"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestScaffoldReportsAGitThatCannotStart(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()
	cause := errors.New("exec format error")
	runner.errs[gitInit] = cause

	_, err := NewCreate(files, runner).Scaffold(context.Background(), request(), nil)
	if !errors.Is(err, cause) {
		t.Errorf("error = %v, want it to wrap %v", err, cause)
	}
}

func TestPushSetsTheUpstream(t *testing.T) {
	files, runner := newFakeFileSystem(), newFakeCommandRunner()

	result, err := NewCreate(files, runner).Push(context.Background(), projectDir)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}

	if want := []string{gitPush}; !slices.Equal(runner.calls, want) {
		t.Errorf("git calls = %v, want %v", runner.calls, want)
	}

	if result.Step != domain.PushStep || result.Outcome != domain.OutcomeDone {
		t.Errorf("result = %+v, want the push step done", result)
	}
}
