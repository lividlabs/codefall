package domain

import "testing"

func TestStepResults(t *testing.T) {
	done := SettingsStep.Done("wrote .codefall/project.json")
	if done.Outcome != OutcomeDone || done.Step != SettingsStep || done.Detail != "wrote .codefall/project.json" {
		t.Errorf("Done() = %+v, want the step, DONE, and the detail", done)
	}

	skipped := SettingsStep.Skipped("already exists")
	if skipped.Outcome != OutcomeSkipped || skipped.Detail != "already exists" {
		t.Errorf("Skipped() = %+v, want SKIPPED and the detail", skipped)
	}
}

// A step is named by its id in the error that stops a run, so two steps cannot share one. The
// constants because the harness CLI is given them separately.
func TestTheStepsAreIdentifiedConsistently(t *testing.T) {
	seen := map[string]bool{}

	for _, step := range []Step{SettingsStep, ExtensionStep, BeadsStep, HookStep, AgentsStep} {
		if seen[step.ID] {
			t.Errorf("two steps have the id %q, want them distinct", step.ID)
		}

		seen[step.ID] = true
	}
}

func TestReportKeepsStepOrder(t *testing.T) {
	first := SettingsStep.Done("one")
	second := Step{ID: "later", Title: "Later"}.Skipped("two")

	results := NewReport(first, second).Results()
	if len(results) != 2 || results[0] != first || results[1] != second {
		t.Errorf("Results() = %+v, want the results in the order they were given", results)
	}
}

func TestZeroReportHasNoResults(t *testing.T) {
	if got := (Report{}).Results(); len(got) != 0 {
		t.Errorf("Report{}.Results() = %+v, want none", got)
	}
}
