package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// agentsRemedy is what to do about an agent this machine cannot start. A run skips one it cannot
// start and moves to the next in its list (ADR-009.2), so nothing is broken; the person may want the
// binary anyway, may want the agent gone from the list, or may leave it.
const agentsRemedy = "install the missing binary, run codefall config agents <activeAgent> <review|consult> " +
	"<harness[:model]>... to write the list without it, or leave it: a run skips an agent it cannot start here"

// currentRemedy is what to do about a list that names no agent on current: name one in it, or clear
// it so the default entry's list applies.
const currentRemedy = "run codefall config agents <activeAgent> <review|consult> <harness[:model]>... " +
	"naming current, or --clear the list so the default entry's applies, so a run always has a reader it can start"

// agents runs checks 15 and 16: every agent the settings name runs on a harness this machine can
// start, and every list ends somewhere a run can always start (ADR-009.2).
//
// It reads the same settings the settings group validated, so whatever skipped those checks skips
// these, and the checks are absent from the report rather than present with a status of their own.
// Whether the list itself is well formed is the settings-complete check's to say; these two ask
// about the machine and about the lists, which no field's own check can see.
func (d *Diagnose) agents(_ context.Context, dir string, results []domain.Result) []domain.Result {
	if !settingsAreComplete(results) {
		return results
	}

	doc, read := d.settingsDocument(dir)
	if !read {
		return results
	}

	entries := settings.Agents(doc)

	results = append(results, d.agentsRunnable(entries))

	return append(results, agentsEndAtCurrent(entries))
}

// agentsRunnable is check 15: every harness an agent in any list runs on is on PATH, or is current,
// which is always runnable because it is the harness running the session.
//
// It warns rather than fails. An agent this machine cannot start is skipped by every run, which is
// the configured behaviour and not an error; what the warning adds is that the person sees it before
// a run does, and can tell a missing install from a deliberate choice.
func (d *Diagnose) agentsRunnable(entries []settings.Entry) domain.Result {
	var missing []string

	for _, runs := range harnessesNamed(entries) {
		if runs == settings.HarnessCurrent || d.runner.LookPath(runs).IsPresent() {
			continue
		}

		missing = append(missing, runs+" is not on PATH")
	}

	if len(missing) == 0 {
		return domain.AgentsRunnable.PassWithDetail(describeEntries(entries))
	}

	return domain.AgentsRunnable.Warn(strings.Join(missing, "; "), mo.Some(agentsRemedy))
}

// harnessesNamed is every harness any list names, each once, in the order first named.
func harnessesNamed(entries []settings.Entry) []string {
	var named []string

	for _, entry := range entries {
		for _, feature := range settings.Features() {
			for _, agent := range entry.List(feature).OrElse(nil) {
				if !slices.Contains(named, agent.Harness) {
					named = append(named, agent.Harness)
				}
			}
		}
	}

	return named
}

// agentsEndAtCurrent is check 16: every list the settings write names at least one agent on current.
// An active agent with no entry, or an entry with a list left out, resolves through the default
// entry's list, which is one of the lists checked here, so nothing resolves past the check. A list
// without current can end with nothing to run when every external harness is missing or fails, and a
// project that wants exactly that stop is told what it has chosen.
func agentsEndAtCurrent(entries []settings.Entry) domain.Result {
	var without []string

	for _, entry := range entries {
		for _, feature := range settings.Features() {
			if agents, has := entry.List(feature).Get(); has && !settings.HasCurrent(agents) {
				without = append(without, entry.ActiveAgent+" "+feature)
			}
		}
	}

	if len(without) == 0 {
		return domain.AgentsCurrent.Pass()
	}

	return domain.AgentsCurrent.Warn(
		fmt.Sprintf("%s names no agent on current", strings.Join(without, ", ")), mo.Some(currentRemedy))
}

// describeEntries is the entries as a report names them: the active agent and its lists, so a reader
// can match the report to the settings. "default: review current, consult current; muse: review
// claude, codex:gpt-5-codex".
func describeEntries(entries []settings.Entry) string {
	parts := make([]string, 0, len(entries))

	for _, entry := range entries {
		var lists []string

		for _, feature := range settings.Features() {
			if agents, has := entry.List(feature).Get(); has {
				lists = append(lists, feature+" "+settings.DescribeAgents(agents))
			}
		}

		if len(lists) == 0 {
			lists = append(lists, "nothing of its own")
		}

		parts = append(parts, entry.ActiveAgent+": "+strings.Join(lists, ", "))
	}

	return strings.Join(parts, "; ")
}
