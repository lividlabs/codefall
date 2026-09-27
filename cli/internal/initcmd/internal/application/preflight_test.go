package application

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// beadsRequest is the run every preflight test makes: the tracker and the harness are settled, and
// what the test changes is the state of the machine and of the directory.
func beadsRequest() Request {
	return Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude}}
}

// A tool that is missing is found before anything has been done, so a run that cannot finish has
// not started either: no step began and no file was written. Every missing tool is named, because
// someone clearing prerequisites wants the whole list in one go.
func TestRunChecksItsToolsBeforeTheFirstStep(t *testing.T) {
	files := newFakeFileSystem()
	observer := &recordingObserver{}

	_, err := NewInitialize(files, newFakeCommandRunner(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), observer)

	want := "bd is not on PATH (brew install beads); " +
		"git is not on PATH"
	if err == nil || err.Error() != want {
		t.Errorf("Run error = %v, want %q", err, want)
	}

	if len(observer.started) != 0 || len(observer.finished) != 0 {
		t.Errorf("observer saw %d started and %d finished, want none of either",
			len(observer.started), len(observer.finished))
	}

	if len(files.files) != 0 || len(files.made) != 0 {
		t.Errorf("wrote %v and created %v, want nothing touched", files.files, files.made)
	}
}

// A missing tool is one complaint, not also every question that tool would have answered: without
// git there is no way to know whether this is a repository, so nothing is claimed about it.
func TestPreflightDoesNotComplainTwiceAboutOneMissingTool(t *testing.T) {
	runner := uninitialized()
	delete(runner.paths, gitCommand)

	_, err := NewInitialize(settled(""), runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)

	want := "git is not on PATH"
	if err == nil || err.Error() != want {
		t.Errorf("Run error = %v, want exactly %q", err, want)
	}

	if len(runner.calls) != 0 {
		t.Errorf("asked %+v, want nothing asked of a tool that is not there", runner.calls)
	}
}

// bd init run outside a repository silently runs git init first, which is a decision about the
// project that init has no business making — so a directory that is not a work tree refuses the run.
//
// The answer is the word git printed, not its exit code: inside a bare repository and inside a .git
// directory, `rev-parse --is-inside-work-tree` prints false and exits 0, and neither is somewhere bd
// init could commit.
func TestPreflightRefusesADirectoryThatIsNotAGitWorkTree(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result CommandResult
	}{
		{
			name: "git says it is not a repository at all",
			result: CommandResult{
				ExitCode: 128,
				Stderr:   "fatal: not a git repository (or any of the parent directories): .git\n",
			},
		},
		{
			name:   "a bare repository, where git answers false and exits 0",
			result: CommandResult{Stdout: "false\n"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := uninitialized()
			runner.runs[gitWorkTree] = tc.result

			files := newFakeFileSystem()

			_, err := NewInitialize(files, runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)

			want := "not a git work tree (bd init would create a repository here; run git init first)"
			if err == nil || err.Error() != want {
				t.Errorf("Run error = %v, want exactly %q", err, want)
			}

			// Nothing is asked about a repository that is not there.
			if got := commandsAsked(runner); slices.Contains(got, gitStaged) ||
				slices.Contains(got, gitStatus) {
				t.Errorf("asked %q, want no question about what is waiting in it", got)
			}

			if len(files.files) != 0 || len(files.made) != 0 {
				t.Errorf("wrote %v and created %v, want nothing touched", files.files, files.made)
			}
		})
	}
}

// bd init commits with no pathspec, so whatever is staged is swept into a commit that says it
// initialised Beads. A run that would do that refuses to start.
func TestPreflightRefusesAStagedIndexWhenBeadsIsNotInitializedYet(t *testing.T) {
	runner := uninitialized()
	runner.runs[gitStaged] = CommandResult{ExitCode: 1}

	files := newFakeFileSystem()

	_, err := NewInitialize(files, runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)

	want := "the git index has staged changes; bd init commits the whole index, so commit or unstage them first"
	if err == nil || err.Error() != want {
		t.Errorf("Run error = %v, want exactly %q", err, want)
	}

	if len(files.files) != 0 || len(files.made) != 0 {
		t.Errorf("wrote %v and created %v, want nothing touched", files.files, files.made)
	}
}

// bd init git-adds the files it might touch by name and then commits with no pathspec, so an
// uncommitted change to any of them lands in a commit under bd's message rather than its author's.
// A run that would do that refuses to start, naming the paths git reported and not all six.
func TestPreflightRefusesUncommittedChangesToWhatBeadsWouldCommit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		want   string
	}{
		{
			name:   "a modified AGENTS.md that is not staged",
			status: " M AGENTS.md\n",
			want: "bd init would commit uncommitted changes to AGENTS.md; " +
				"commit or stash them first",
		},
		{
			name:   "an untracked .claude/settings.json",
			status: "?? .claude/settings.json\n",
			want: "bd init would commit uncommitted changes to .claude/settings.json; " +
				"commit or stash them first",
		},
		{
			name:   "several at once, in the order git reported them",
			status: " M AGENTS.md\n?? .claude/settings.json\n M CLAUDE.md\n",
			want: "bd init would commit uncommitted changes to " +
				"AGENTS.md, .claude/settings.json, CLAUDE.md; commit or stash them first",
		},
		{
			name:   "a rename, which is the new name that would be committed",
			status: "R  AGENTS.old -> AGENTS.md\n",
			want: "bd init would commit uncommitted changes to AGENTS.md; " +
				"commit or stash them first",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := uninitialized()
			runner.runs[gitStatus] = CommandResult{Stdout: tc.status}

			files := newFakeFileSystem()

			_, err := NewInitialize(files, runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Run error = %v, want it to say %q", err, tc.want)
			}

			if len(files.files) != 0 || len(files.made) != 0 {
				t.Errorf("wrote %v and created %v, want nothing touched", files.files, files.made)
			}
		})
	}
}

// A directory with nothing waiting in it passes: the guard has something to say only when git does.
func TestPreflightPassesACleanWorkTree(t *testing.T) {
	runner := uninitialized()

	if _, err := NewInitialize(settled(""), runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := commandsAsked(runner); !slices.Contains(got, gitStatus) {
		t.Errorf("asked %q, want %q among them", got, gitStatus)
	}
}

// Beads that is already initialised means bd init will not run, and then what the directory has
// waiting is nobody's business but the person who left it there. Neither question is even asked.
func TestPreflightIgnoresWhatIsWaitingWhenBeadsIsAlreadyInitialized(t *testing.T) {
	runner := toolsInstalled()
	runner.runs[gitStaged] = CommandResult{ExitCode: 1}
	runner.runs[gitStatus] = CommandResult{Stdout: " M AGENTS.md\n"}

	if _, err := NewInitialize(settled(""), runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := commandsAsked(runner); slices.Contains(got, gitStaged) || slices.Contains(got, gitStatus) {
		t.Errorf("asked %q, want no question about what is waiting in the directory", got)
	}
}

// A run that cannot start still unwraps to what stopped it, so a cancelled context is one errors.Is
// away rather than a sentence inside a flattened message.
func TestPreflightKeepsTheErrorsItJoined(t *testing.T) {
	runner := uninitialized()
	runner.errs[gitStaged] = context.Canceled

	_, err := NewInitialize(newFakeFileSystem(), runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run error = %v, want it to unwrap to context.Canceled", err)
	}
}

// git failing for a reason of its own is neither a clean index nor a dirty one, so it is reported as
// what it is, with what git said.
func TestPreflightReportsGitFailingForItsOwnReasons(t *testing.T) {
	runner := uninitialized()
	runner.runs[gitStaged] = CommandResult{ExitCode: 128, Stderr: "fatal: bad object HEAD\nsee git-diff(1)\n"}

	_, err := NewInitialize(newFakeFileSystem(), runner, newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)

	want := "git diff --cached --quiet exited 128: fatal: bad object HEAD"
	if err == nil || err.Error() != want {
		t.Errorf("Run error = %v, want exactly %q", err, want)
	}
}

// commandsAsked is every command the runner was given, probes included — which is what a test about
// preflight is looking at.
func commandsAsked(runner *fakeCommandRunner) []string {
	got := make([]string, 0, len(runner.calls))
	for _, call := range runner.calls {
		got = append(got, call.command)
	}

	return got
}
