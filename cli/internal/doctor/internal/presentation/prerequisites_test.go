package presentation

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/ui"
)

// check runs the prerequisite check against a fake use case, through the writer a command would
// hand it, and returns everything it wrote.
func check(t *testing.T, diagnose DiagnoseUseCase) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")

	var out bytes.Buffer

	err := NewPrerequisiteCheck(diagnose)(t.Context(), ui.NewWriter(&out))

	return out.String(), err
}

// A machine with everything is not told about it: init and create go on to their own output.
func TestPrerequisiteCheckWritesNothingWhenEveryCheckPasses(t *testing.T) {
	out, err := check(t, &fakeDiagnose{report: domain.NewReport(
		domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
		domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
		domain.GHAuthenticated.PassWithDetail("logged in as djensen47"),
		domain.GHScopes.Pass(),
	)})
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if out != "" {
		t.Errorf("output = %q, want none", out)
	}
}

// Every failure is listed with its remedy, in the form doctor prints, so the person clears all of
// them in one pass. The error counts checks, not categories: it is the number of things to fix.
func TestPrerequisiteCheckListsEveryFailureWithItsRemedy(t *testing.T) {
	out, err := check(t, &fakeDiagnose{report: domain.NewReport(
		domain.BeadsInstalled.Fail("bd is not on PATH", mo.Some("brew install beads")),
		domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
		domain.GHAuthenticated.Fail("No active github.com account", mo.Some("gh auth login")),
	)})
	if err == nil {
		t.Fatalf("check: want an error, got nil\n%s", out)
	}

	if got, want := err.Error(), "2 prerequisite checks failed"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}

	want := "[✗] Beads\n" +
		"╰─ ✗ bd is not on PATH\n" +
		"   ╰─ fix: brew install beads\n" +
		"[✗] GitHub CLI (gh version 2.97.0 (2026-07-31))\n" +
		"╰─ ✗ No active github.com account\n" +
		"   ╰─ fix: gh auth login\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// A category that passed is left out of a refusal: the list is what there is to fix.
func TestPrerequisiteCheckLeavesOutACategoryThatPassed(t *testing.T) {
	out, err := check(t, &fakeDiagnose{report: domain.NewReport(
		domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
		domain.GHInstalled.Fail("gh is not on PATH", mo.Some("brew install gh")),
	)})
	if err == nil || err.Error() != "1 prerequisite check failed" {
		t.Fatalf("error = %v, want %q", err, "1 prerequisite check failed")
	}

	want := "[✗] GitHub CLI\n" +
		"╰─ ✗ gh is not on PATH\n" +
		"   ╰─ fix: brew install gh\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// A token without the project scopes limits codefall and does not stop it, so doctor warns and a run
// is not refused. Doctor is where the warning is read.
func TestPrerequisiteCheckDoesNotRefuseOnAWarning(t *testing.T) {
	out, err := check(t, &fakeDiagnose{report: domain.NewReport(
		domain.BeadsInstalled.PassWithDetail("bd version 1.2.2 (Homebrew)"),
		domain.GHInstalled.PassWithDetail("gh version 2.97.0 (2026-07-31)"),
		domain.GHAuthenticated.PassWithDetail("logged in as djensen47"),
		domain.GHScopes.Warn("The token is missing read:project, project",
			mo.Some("gh auth refresh -s read:project -s project")),
	)})
	if err != nil {
		t.Fatalf("check: %v", err)
	}

	if out != "" {
		t.Errorf("output = %q, want none", out)
	}
}

// A check that was stopped is reported as the context's own error, which each command says in its
// own words.
func TestPrerequisiteCheckReturnsACancelledContextAsItIs(t *testing.T) {
	_, err := check(t, &fakeDiagnose{err: context.Canceled})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
