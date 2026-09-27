package application

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

const workingDir = "/work"

var (
	codefallFull      = filepath.Join(workingDir, ".codefall")
	settingsFull      = filepath.Join(codefallFull, "settings.json")
	claudeSettingsDir = filepath.Join(workingDir, ".claude")
	claudeFull        = filepath.Join(claudeSettingsDir, "settings.json")
	// The tree the testing step makes at the root a request that declares none takes by default.
	testingFull   = filepath.Join(workingDir, settings.DefaultTestDir)
	testCasesFull = filepath.Join(testingFull, "test-cases")
)

// The commands a run gives its tools, keyed the way the fake runner keys them.
const (
	pluginInstall = "claude extension install codefall@codefall --scope project -y"
	beadsInit     = "bd init --non-interactive --skip-agents"
	beadsAuditOff = "bd config set audit.enabled false"
	beadsInfo     = "bd info"
	gitWorkTree   = "git rev-parse --is-inside-work-tree"
	gitStaged     = "git diff --cached --quiet"
	gitStatus     = "git status --porcelain -- " +
		".gitignore AGENTS.md CLAUDE.md .claude/settings.json .codex .agents"
)

// --- fakes -------------------------------------------------------------------------------------

type fakeFileSystem struct {
	files map[string][]byte
	errs  map[string]error
	made  []string
}

func newFakeFileSystem() *fakeFileSystem {
	return &fakeFileSystem{
		files: map[string][]byte{},
		errs:  map[string]error{},
	}
}

func (f *fakeFileSystem) ReadFile(path string) ([]byte, error) {
	if err, ok := f.errs[path]; ok {
		return nil, err
	}

	data, ok := f.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return data, nil
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

	f.files[path] = data

	return nil
}

type runCall struct {
	dir     string
	command string
}

type fakeCommandRunner struct {
	paths map[string]string
	runs  map[string]CommandResult
	errs  map[string]error
	calls []runCall
}

func newFakeCommandRunner() *fakeCommandRunner {
	return &fakeCommandRunner{
		paths: map[string]string{},
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

func (r *fakeCommandRunner) Run(
	_ context.Context, dir, name string, args ...string,
) (CommandResult, error) {
	command := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, runCall{dir: dir, command: command})

	if err, ok := r.errs[command]; ok {
		return CommandResult{}, err
	}

	return r.runs[command], nil
}

// toolsInstalled is the runner a run with the claude harness needs: every tool it reaches for
// is on PATH, and every command it is given succeeds, because the zero CommandResult exited 0. That
// makes this a repository that is a git work tree, has nothing staged, and already has Beads — so a
// test that wants bd init to run says so by failing `bd info`.
func toolsInstalled() *fakeCommandRunner {
	runner := newFakeCommandRunner()
	runner.paths["claude"] = "/opt/homebrew/bin/claude"
	runner.paths["bd"] = "/opt/homebrew/bin/bd"
	runner.paths["git"] = "/usr/bin/git"
	// git answers the work-tree question with a word, not an exit code.
	runner.runs[gitWorkTree] = CommandResult{Stdout: "true\n"}

	return runner
}

// uninitialized is toolsInstalled in a repository Beads has never been run in, which is what makes
// the beads step do its work.
func uninitialized() *fakeCommandRunner {
	runner := toolsInstalled()
	runner.runs[beadsInfo] = CommandResult{ExitCode: 1, Stderr: "Error: no beads database found\n"}

	return runner
}

// recordingObserver is what a spinner does in production, without the terminal: it remembers what it
// was told and in which order.
type recordingObserver struct {
	started  []domain.Step
	finished []domain.StepResult
}

func (o *recordingObserver) StepStarted(step domain.Step) {
	o.started = append(o.started, step)
}

func (o *recordingObserver) StepFinished(result domain.StepResult) {
	o.finished = append(o.finished, result)
}

// fetchCall is one Fetch the use case asked for.
type fetchCall struct {
	dir     string
	sources []string
	exclude []string
}

// fakeExtensionSource remembers each Fetch and answers success unless the test gave it an error.
// Reads answer from the hook definitions map hook_test seeds — a test that wants a missing file
// re-seeds its own. It answers with one path per subtree it was asked for, which is what a manifest
// would record, so the extension step's detail string and the manifest test both have something to
// hold.
type fakeExtensionSource struct {
	calls []fetchCall
	err   error
	// failOn narrows err to the fetch of one subtree, so a test can fail the copy into .codefall/
	// while the copy into a skills directory succeeds.
	failOn string
	data   map[string][]byte
}

// newFakeExtensionSource answers with the hook definitions and the documents under agents/: the two
// kinds of file a step reads from the tree one at a time rather than copies. Each test gets its own
// map, so one that replaces a file to break it breaks it for itself alone.
func newFakeExtensionSource() *fakeExtensionSource {
	data := make(map[string][]byte, len(hookDefinitions)+len(agentsDocuments))
	maps.Copy(data, hookDefinitions)
	maps.Copy(data, agentsDocuments)

	return &fakeExtensionSource{data: data}
}

// fetched is the one file the fake answers with for each subtree a caller names.
var fetched = map[string]string{
	"skills":       "skills/design/SKILL.md",
	"hooks/shared": "hooks/shared/codefall-block-merge-to-main.sh",
	"shared":       "shared/preflight.sh",
}

func (f *fakeExtensionSource) Fetch(
	_ context.Context, destDir string, sources, exclude []string,
) ([]string, error) {
	f.calls = append(f.calls, fetchCall{dir: destDir, sources: sources, exclude: exclude})

	written := make([]string, 0, len(sources))
	for _, source := range sources {
		written = append(written, fetched[source])
	}

	if f.err != nil && (f.failOn == "" || slices.Contains(sources, f.failOn)) {
		return written, f.err
	}

	return written, nil
}

func (f *fakeExtensionSource) Read(path string) ([]byte, error) {
	data, ok := f.data[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}
	return data, nil
}

// --- the run -----------------------------------------------------------------------------------

func TestRunWritesSettingsAndReportsWhatItWrote(t *testing.T) {
	files := newFakeFileSystem()
	observer := &recordingObserver{}

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(),
		Request{
			Dir:           workingDir,
			Tracker:       settings.TrackerGitHub,
			IssuesRepo:    mo.Some("lividlabs/codefall-cli"),
			IssuesProject: mo.Some(3),
			Harnesses:     []string{harness.Claude},
		},
		observer,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	if results[0].Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", results[0].Outcome)
	}

	want := "wrote .codefall/settings.json (tracker: github, harnesses: claude, " +
		"issues repo: lividlabs/codefall-cli, issues project: 3)"
	if results[0].Detail != want {
		t.Errorf("detail = %q, want %q", results[0].Detail, want)
	}

	// The settings step makes .codefall/, the hook step makes .claude/, the testing step makes the
	// tree its cases live in, and nothing else makes a directory.
	made := []string{codefallFull, claudeSettingsDir, testingFull, testCasesFull}
	if !slices.Equal(files.made, made) {
		t.Errorf("created %q, want %q", files.made, made)
	}

	// The observer sees every step start and finish, in order, and what it sees on finishing is the
	// result the report carries.
	steps := []domain.Step{
		domain.SettingsStep, domain.ExtensionStep, domain.BeadsStep, domain.HookStep,
		domain.AgentsStep, domain.TestingStep, domain.IgnoreStep,
	}
	if !slices.Equal(observer.started, steps) {
		t.Errorf("started = %+v, want %+v", observer.started, steps)
	}

	if !slices.Equal(observer.finished, results) {
		t.Errorf("finished = %+v, want %+v", observer.finished, results)
	}
}

func TestRunEncodesGitHubSettings(t *testing.T) {
	files := newFakeFileSystem()

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(),
		Request{
			Dir:           workingDir,
			Tracker:       settings.TrackerGitHub,
			IssuesRepo:    mo.Some("owner/name"),
			IssuesProject: mo.Some(3),
			Harnesses:     []string{harness.Claude},
		},
		nil,
	); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": [
    "claude"
  ],
  "github": {
    "issuesRepo": "owner/name",
    "issuesProject": 3
  },
  "review": {
    "postToPullRequest": false
  },
  "test": {
    "dir": "testing"
  }
}
`

	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// An absent project is omitted rather than written as null, and the empty beads block is written
// rather than omitted — the schema requires the block that the tracker selects.
func TestRunEncodesTheOptionalFieldsTheWayTheSchemaExpects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request Request
		want    string
	}{
		{
			name: "github without a project",
			request: Request{
				Dir:        workingDir,
				Tracker:    settings.TrackerGitHub,
				IssuesRepo: mo.Some("owner/name"),
				Harnesses:  []string{harness.Claude},
			},
			want: `  "tracker": "github",
  "harnesses": [
    "claude"
  ],
  "github": {
    "issuesRepo": "owner/name"
  },
  "review": {
    "postToPullRequest": false
  },
  "test": {
    "dir": "testing"
  }
}
`,
		},
		{
			name:    "beads",
			request: Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}},
			want: `  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "beads": {},
  "review": {
    "postToPullRequest": false
  },
  "test": {
    "dir": "testing"
  }
}
`,
		},
		{
			name: "review posting turned on",
			request: Request{
				Dir:                     workingDir,
				Tracker:                 settings.TrackerBeads,
				Harnesses:               []string{harness.Claude},
				ReviewPostToPullRequest: mo.Some(true),
			},
			want: `  "review": {
    "postToPullRequest": true
  },
  "test": {
    "dir": "testing"
  }
}
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem()

			if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), tc.request, nil); err != nil {
				t.Fatalf("Run: %v", err)
			}

			got := string(files.files[settingsFull])
			if !strings.HasSuffix(got, tc.want) {
				t.Errorf("settings.json =\n%s\nwant it to end with\n%s", got, tc.want)
			}

			if strings.Contains(got, "null") {
				t.Errorf("settings.json =\n%s\nwant no null in it", got)
			}
		})
	}
}

func TestRunSkipsSettingsThatAreAlreadyThere(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte("{}\n")

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(),
		Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}},
		nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	results := report.Results()
	if len(results) != 7 || results[0].Outcome != domain.OutcomeSkipped {
		t.Fatalf("Results() = %+v, want the settings step to have skipped", results)
	}

	want := ".codefall/settings.json already exists (use --force to rewrite it)"
	if results[0].Detail != want {
		t.Errorf("detail = %q, want %q", results[0].Detail, want)
	}

	// The settings step wrote nothing, so the file still records no tracker and no harnesses. What
	// it does record is the testing root, which the testing step declares into whatever settings a
	// project has — that is how a project settled before the block existed gets one.
	got := string(files.files[settingsFull])

	if strings.Contains(got, "tracker") {
		t.Errorf("settings.json = %q, want the settings step to have left it alone", got)
	}

	if !strings.Contains(got, `"dir": "testing"`) {
		t.Errorf("settings.json = %q, want the testing step to have declared the root", got)
	}
}

func TestRunRewritesSettingsWithForce(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte("{}\n")

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(),
		Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}, Force: true},
		nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := report.Results()[0].Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	if got := string(files.files[settingsFull]); !strings.Contains(got, `"tracker": "beads"`) {
		t.Errorf("settings.json = %q, want it rewritten", got)
	}
}

// A step that cannot finish ends the run, and the error names the step so the reader knows how far
// init got. The observer is told the step started and never told it finished.
func TestRunStopsOnAStepThatFails(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeFileSystem)
		want  string
	}{
		{
			name:  "the settings could not be built",
			setup: func(*fakeFileSystem) {},
			want:  "needs a repository",
		},
		{
			name: "the directory could not be made",
			setup: func(f *fakeFileSystem) {
				f.errs["mkdir "+codefallFull] = errors.New("read-only file system")
			},
			want: "create .codefall/",
		},
		{
			name: "the file could not be written",
			setup: func(f *fakeFileSystem) {
				f.errs["write "+settingsFull] = errors.New("no space left on device")
			},
			want: "write .codefall/settings.json",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem()
			tc.setup(files)

			observer := &recordingObserver{}

			request := Request{Dir: workingDir, Tracker: settings.TrackerGitHub, Harnesses: []string{harness.Claude}}
			if tc.name != "the settings could not be built" {
				request.IssuesRepo = mo.Some("owner/name")
			}

			report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, observer)
			if err == nil {
				t.Fatalf("Run = %+v, want an error", report)
			}

			if !strings.HasPrefix(err.Error(), domain.SettingsStep.ID+": ") {
				t.Errorf("error = %q, want it to name the step it failed in", err)
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}

			if len(report.Results()) != 0 {
				t.Errorf("Results() = %+v, want none", report.Results())
			}

			if len(observer.started) != 1 || len(observer.finished) != 0 {
				t.Errorf("observer saw %d started and %d finished, want 1 and 0",
					len(observer.started), len(observer.finished))
			}
		})
	}
}

func TestRunReportsAnUnreadableSettingsFile(t *testing.T) {
	files := newFakeFileSystem()
	files.errs[settingsFull] = errors.New("permission denied")

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(),
		Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}},
		nil,
	); err == nil || !strings.Contains(err.Error(), "read .codefall/settings.json") {
		t.Errorf("Run error = %v, want it to say the settings could not be read", err)
	}
}

func TestRunStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewInitialize(newFakeFileSystem(), toolsInstalled(), newFakeExtensionSource()).Run(
		ctx,
		Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}},
		nil,
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run error = %v, want a context.Canceled", err)
	}
}

// --- the questions presentation asks -----------------------------------------------------------

func TestSettingsExist(t *testing.T) {
	files := newFakeFileSystem()

	exists, err := NewInitialize(files, newFakeCommandRunner(), newFakeExtensionSource()).SettingsExist(workingDir)
	if err != nil || exists {
		t.Errorf("SettingsExist of an empty directory = %v, %v, want false, nil", exists, err)
	}

	files.files[settingsFull] = []byte("{}")

	exists, err = NewInitialize(files, newFakeCommandRunner(), newFakeExtensionSource()).SettingsExist(workingDir)
	if err != nil || !exists {
		t.Errorf("SettingsExist with a settings file = %v, %v, want true, nil", exists, err)
	}

	files.errs[settingsFull] = errors.New("permission denied")

	if _, err := NewInitialize(files, newFakeCommandRunner(), newFakeExtensionSource()).SettingsExist(workingDir); err == nil {
		t.Error("SettingsExist of an unreadable file = nil error, want an error")
	}
}

func TestSuggestIssuesRepo(t *testing.T) {
	const (
		ghRepoView   = "gh repo view --json nameWithOwner --jq .nameWithOwner"
		gitRemoteURL = "git remote get-url origin"
	)

	withGH := func() *fakeCommandRunner {
		runner := newFakeCommandRunner()
		runner.paths["gh"] = "/opt/homebrew/bin/gh"

		return runner
	}

	withGit := func(runner *fakeCommandRunner, url string) *fakeCommandRunner {
		runner.paths["git"] = "/usr/bin/git"
		runner.runs[gitRemoteURL] = CommandResult{Stdout: url}

		return runner
	}

	// gh answering means the remote is never consulted, because gh knows the repository as GitHub
	// knows it now and the remote only knows what it was cloned from.
	t.Run("the repository gh names, trimmed", func(t *testing.T) {
		runner := withGit(withGH(), "git@github.com:lividlabs/stale.git\n")
		runner.runs[ghRepoView] = CommandResult{Stdout: "lividlabs/codefall-cli\n"}

		got := NewInitialize(newFakeFileSystem(), runner, newFakeExtensionSource()).SuggestIssuesRepo(t.Context(), workingDir)
		if repo, ok := got.Get(); !ok || repo != "lividlabs/codefall-cli" {
			t.Errorf("SuggestIssuesRepo = %v, want Some(%q)", got, "lividlabs/codefall-cli")
		}

		// gh answers about the directory init is running in, not about wherever the process started.
		if len(runner.calls) != 1 || runner.calls[0].dir != workingDir {
			t.Errorf("calls = %+v, want one in %q", runner.calls, workingDir)
		}
	})

	for _, tc := range []struct {
		name   string
		runner func() *fakeCommandRunner
	}{
		{
			name: "gh is not installed",
			runner: func() *fakeCommandRunner {
				return withGit(newFakeCommandRunner(), "git@github.com:lividlabs/codefall-cli.git\n")
			},
		},
		{
			name: "gh is installed but has no answer",
			runner: func() *fakeCommandRunner {
				runner := withGit(withGH(), "https://github.com/lividlabs/codefall-cli.git\n")
				runner.runs[ghRepoView] = CommandResult{ExitCode: 1, Stderr: "not logged in\n"}

				return runner
			},
		},
	} {
		t.Run("the origin remote when "+tc.name, func(t *testing.T) {
			got := NewInitialize(newFakeFileSystem(), tc.runner(), newFakeExtensionSource()).SuggestIssuesRepo(t.Context(), workingDir)
			if repo, ok := got.Get(); !ok || repo != "lividlabs/codefall-cli" {
				t.Errorf("SuggestIssuesRepo = %v, want Some(%q)", got, "lividlabs/codefall-cli")
			}
		})
	}

	for _, tc := range []struct {
		name   string
		runner func() *fakeCommandRunner
	}{
		{
			name:   "neither tool is installed",
			runner: newFakeCommandRunner,
		},
		{
			name: "the directory is not a GitHub repository",
			runner: func() *fakeCommandRunner {
				runner := withGH()
				runner.paths["git"] = "/usr/bin/git"
				runner.runs[ghRepoView] = CommandResult{ExitCode: 1, Stderr: "not a git repository\n"}
				runner.runs[gitRemoteURL] = CommandResult{ExitCode: 128, Stderr: "No such remote\n"}

				return runner
			},
		},
		{
			name: "neither tool could be started",
			runner: func() *fakeCommandRunner {
				runner := withGH()
				runner.paths["git"] = "/usr/bin/git"
				runner.errs[ghRepoView] = errors.New("broken pipe")
				runner.errs[gitRemoteURL] = errors.New("broken pipe")

				return runner
			},
		},
		{
			name: "both tools said nothing",
			runner: func() *fakeCommandRunner {
				runner := withGH()
				runner.runs[ghRepoView] = CommandResult{Stdout: "  \n"}

				return withGit(runner, "\n")
			},
		},
		{
			name: "the remote is not on GitHub",
			runner: func() *fakeCommandRunner {
				return withGit(newFakeCommandRunner(), "git@gitlab.com:lividlabs/codefall-cli.git\n")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := NewInitialize(newFakeFileSystem(), tc.runner(), newFakeExtensionSource()).SuggestIssuesRepo(t.Context(), workingDir)
			if got.IsPresent() {
				t.Errorf("SuggestIssuesRepo = %v, want None", got)
			}
		})
	}
}

// TestRepoFromRemoteURL covers the shapes git writes and the ones it does not, because the whole
// point of the fallback is that it reads what is already in .git/config.
func TestRepoFromRemoteURL(t *testing.T) {
	const repo = "lividlabs/codefall-cli"

	for _, tc := range []struct {
		url  string
		want string
	}{
		{url: "git@github.com:lividlabs/codefall-cli.git\n", want: repo},
		{url: "git@github.com:lividlabs/codefall-cli", want: repo},
		{url: "https://github.com/lividlabs/codefall-cli.git", want: repo},
		{url: "https://github.com/lividlabs/codefall-cli/", want: repo},
		{url: "https://djensen@github.com/lividlabs/codefall-cli.git", want: repo},
		{url: "ssh://git@github.com/lividlabs/codefall-cli.git", want: repo},
		{url: "ssh://git@github.com:443/lividlabs/codefall-cli.git", want: repo},
		{url: "git://github.com/lividlabs/codefall-cli.git", want: repo},
		{url: "https://GitHub.com/lividlabs/codefall-cli.git", want: repo},
		// Somebody else's forge, an SSH host alias a multi-account setup uses, a path with no
		// repository in it, and a local clone: none of them name a repository on GitHub.
		{url: "git@gitlab.com:lividlabs/codefall-cli.git", want: ""},
		{url: "https://github.example.com/lividlabs/codefall-cli.git", want: ""},
		{url: "git@github.com-work:lividlabs/codefall-cli.git", want: ""},
		{url: "https://github.com/lividlabs", want: ""},
		{url: "https://github.com/lividlabs/codefall-cli/tree/main", want: ""},
		{url: "/Users/djensen/code/lividlabs/codefall-cli", want: ""},
		{url: "", want: ""},
	} {
		t.Run(tc.url, func(t *testing.T) {
			got := repoFromRemoteURL(tc.url)

			if tc.want == "" {
				if got.IsPresent() {
					t.Errorf("repoFromRemoteURL(%q) = %v, want None", tc.url, got)
				}

				return
			}

			if repo, ok := got.Get(); !ok || repo != tc.want {
				t.Errorf("repoFromRemoteURL(%q) = %v, want Some(%q)", tc.url, got, tc.want)
			}
		})
	}
}
