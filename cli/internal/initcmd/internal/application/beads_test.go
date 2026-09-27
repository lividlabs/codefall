package application

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
)

// beadsResult is what the third step did in a run that got that far.
func beadsResult(t *testing.T, report domain.Report) domain.StepResult {
	t.Helper()

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	return results[2]
}

// A repository bd already knows about is left alone: re-running bd init there is an error rather
// than a no-op, so the question is asked first.
func TestBeadsStepSkipsARepositoryThatAlreadyHasBeads(t *testing.T) {
	runner := toolsInstalled()

	report, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := beadsResult(t, report)
	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	if want := "Beads is already initialized here"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	got := commandsAsked(runner)
	if slices.Contains(got, beadsInit) {
		t.Errorf("asked %q, want bd init not to have run", got)
	}

	// A database that was already there may have turned the interaction log on; that is its choice.
	if slices.Contains(got, beadsAuditOff) {
		t.Errorf("asked %q, want the interaction log's default left alone", got)
	}
}

// bd declining to write the default is a clause of the sentence, not a failed step: the log is off
// by default, so the database works either way.
func TestBeadsStepReportsWhenTheInteractionLogDefaultCouldNotBeWritten(t *testing.T) {
	runner := uninitialized()
	runner.runs[beadsInit] = CommandResult{Stdout: "bd initialized successfully!\n"}
	runner.runs[beadsAuditOff] = CommandResult{ExitCode: 1, Stderr: "Error: unknown key\n"}

	report, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := beadsResult(t, report)
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "initialized Beads; bd config set audit.enabled false exited 1 (Error: unknown key), " +
		"so the interaction log is at bd's default"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// The step runs bd with the arguments that keep it inside .beads/: non-interactive because init has
// already asked its questions, and --skip-agents because codefall owns AGENTS.md and CLAUDE.md.
// What bd said about a database it has only just made is not worth repeating, so a successful run
// reports the one thing a reader could not have guessed — whether bd committed.
func TestBeadsStepInitializesBeads(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result CommandResult
		want   string
	}{
		{
			name: "bd committed what it wrote",
			result: CommandResult{
				Stdout: "✓ Created .beads/\n  ✓ Committed beads files to git\nbd initialized successfully!\n",
				Stderr: "⚠ No Dolt remote configured\n",
			},
			want: `initialized Beads (bd init committed what it wrote as ` +
				`"bd init: initialize beads issue tracking"), with the interaction log off in its config.yaml`,
		},
		{
			name:   "bd had nothing to commit",
			result: CommandResult{Stdout: "✓ Created .beads/\nbd initialized successfully!\n"},
			want:   "initialized Beads, with the interaction log off in its config.yaml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := uninitialized()
			runner.runs[beadsInit] = tc.result

			report, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), nil)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			result := beadsResult(t, report)
			if result.Outcome != domain.OutcomeDone {
				t.Errorf("outcome = %v, want DONE", result.Outcome)
			}

			if result.Detail != tc.want {
				t.Errorf("detail = %q, want %q", result.Detail, tc.want)
			}

			got := commandsAsked(runner)
			if !slices.Contains(got, beadsInit) {
				t.Errorf("asked %q, want %q among them", got, beadsInit)
			}

			// The interaction log's default is written into the database bd init just made, after it.
			if slices.Index(got, beadsAuditOff) < slices.Index(got, beadsInit) {
				t.Errorf("asked %q, want %q after %q", got, beadsAuditOff, beadsInit)
			}

			// bd is asked about the directory init was pointed at, not wherever the process started.
			for _, call := range runner.calls {
				if call.dir != workingDir {
					t.Errorf("ran %q in %q, want %q", call.command, call.dir, workingDir)
				}
			}
		})
	}
}

// bd refusing stops the run naming the step, and carries what bd said. The hook step never runs,
// because there is nothing yet for a session to be primed with.
func TestBeadsStepStopsTheRunWhenBeadsRefuses(t *testing.T) {
	runner := uninitialized()
	runner.runs[beadsInit] = CommandResult{ExitCode: 1, Stderr: "Error: dolt is not installed\nsee the docs\n"}

	files := settled("{}")
	observer := &recordingObserver{}

	report, err := NewInitialize(files, runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), observer)
	if err == nil {
		t.Fatalf("Run = %+v, want an error", report)
	}

	if !strings.HasPrefix(err.Error(), domain.BeadsStep.ID+": ") {
		t.Errorf("error = %q, want it to name the step it failed in", err)
	}

	want := "bd init --non-interactive --skip-agents exited 1: Error: dolt is not installed"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to mention %q", err, want)
	}

	// The settings and extension steps finished; the beads step started and never did.
	if len(observer.started) != 3 || len(observer.finished) != 2 {
		t.Errorf("observer saw %d started and %d finished, want 3 and 2",
			len(observer.started), len(observer.finished))
	}

	if got := string(files.files[claudeFull]); got != "{}" {
		t.Errorf(".claude/settings.json = %q, want the hook step never to have run", got)
	}
}

// A bd that could not be started at all is a different failure from one that refused, and says so.
func TestBeadsStepStopsTheRunWhenBeadsCannotBeStarted(t *testing.T) {
	runner := uninitialized()
	runner.errs[beadsInit] = errors.New("broken pipe")

	_, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), nil)
	if err == nil || !strings.Contains(err.Error(), "run bd init --non-interactive --skip-agents") {
		t.Errorf("Run error = %v, want it to say the command could not be run", err)
	}
}

// A repository holds one Beads database: bd init initializes the directory it runs in, and the
// multi-repo configuration that shares one is scoped to repositories rather than directories. So an
// install below the root asks in its own directory and, when the answer is that Beads is already
// there, leaves it alone rather than starting a second database beside it.
func TestBeadsStepDoesNotStartASecondDatabaseBelowTheRoot(t *testing.T) {
	runner := toolsInstalled()
	request := beadsRequest()
	request.Dir = filepath.Join(workingDir, "apps", "web")

	report, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result := beadsResult(t, report); result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	for _, call := range runner.calls {
		if strings.HasPrefix(call.command, beadsCommand+" init") {
			t.Errorf("ran %q in %s, want no second database below the root", call.command, call.dir)
		}

		if call.command == beadsCommand+" info" && call.dir != request.Dir {
			t.Errorf("asked %q in %s, want the install directory %s", call.command, call.dir, request.Dir)
		}
	}
}

// A repository has one git hook path, and bd init claims it for Beads' own hooks. At the root that
// is Beads' business; below it, the path belongs to the whole repository and to everyone working in
// it, so an install in one part of a larger repository leaves it alone and says that it did.
func TestBeadsStepLeavesTheRepositoryGitHooksAloneBelowTheRoot(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prefix  string
		command string
		detail  string
	}{
		{
			name:    "at the root",
			command: "bd init --non-interactive --skip-agents",
			detail:  "initialized Beads, with the interaction log off in its config.yaml",
		},
		{
			name:    "below the root",
			prefix:  "apps/web/",
			command: "bd init --non-interactive --skip-agents --skip-hooks",
			detail: "initialized Beads without its git hooks, which belong to the whole repository, " +
				"with the interaction log off in its config.yaml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := uninitialized()
			runner.runs[gitPrefix] = CommandResult{Stdout: tc.prefix + "\n"}

			request := beadsRequest()
			request.Dir = filepath.Join(workingDir, filepath.FromSlash(tc.prefix))

			report, err := NewInitialize(settled("{}"), runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), request, nil)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			if result := beadsResult(t, report); result.Detail != tc.detail {
				t.Errorf("detail = %q, want %q", result.Detail, tc.detail)
			}

			var initialized []string
			for _, call := range runner.calls {
				if strings.HasPrefix(call.command, beadsCommand+" init") {
					initialized = append(initialized, call.command)
				}
			}

			if !slices.Equal(initialized, []string{tc.command}) {
				t.Errorf("bd init calls = %q, want %q", initialized, tc.command)
			}
		})
	}
}
