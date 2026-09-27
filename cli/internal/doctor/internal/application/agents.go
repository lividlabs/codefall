package application

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// agentsRemedy is what to do about an agent this machine cannot start. A run skips one it cannot
// start and moves to the next in the order (ADR-009), so nothing is broken; the person may want the
// binary anyway, or may want the order to say what this machine can do.
const agentsRemedy = "install the missing binary, or leave it: a run skips an agent it cannot start here"

// currentRemedy is what to do about an order with nothing this harness can always run.
const currentRemedy = "add an agent whose harness is current to the order, so a run always has a reader it can start"

// agents runs checks 12 and 13: every agent the settings define runs on a harness this machine can
// start, and every order ends somewhere a run can always start (ADR-009).
//
// It reads the same settings the settings group validated, so whatever skipped those checks skips
// these, and the checks are absent from the report rather than present with a status of their own.
// Whether the list itself is well formed is the settings-complete check's to say; these two ask
// about the machine and about the orders, which no field's own check can see.
func (d *Diagnose) agents(_ context.Context, dir string, results []domain.Result) []domain.Result {
	if !settingsAreComplete(results) {
		return results
	}

	doc, read := d.settingsDocument(dir)
	if !read {
		return results
	}

	defined := settings.Agents(doc)

	results = append(results, d.agentsRunnable(defined))

	return append(results, agentsEndAtCurrent(doc, defined))
}

// agentsRunnable is check 12: each agent's harness is on PATH, or is current, which is always
// runnable because it is the harness running the session.
//
// It warns rather than fails. An agent this machine cannot start is skipped by every run, which is
// the configured behaviour and not an error; what the warning adds is that the person sees it before
// a run does, and can tell a missing install from a deliberate choice.
func (d *Diagnose) agentsRunnable(defined []settings.Agent) domain.Result {
	var missing []string

	for _, agent := range defined {
		if agent.Harness == settings.HarnessCurrent || d.runner.LookPath(agent.Harness).IsPresent() {
			continue
		}

		missing = append(missing, fmt.Sprintf("%s runs on %s, which is not on PATH", agent.Name, agent.Harness))
	}

	if len(missing) == 0 {
		return domain.AgentsRunnable.PassWithDetail(settings.DescribeAgents(defined))
	}

	return domain.AgentsRunnable.Warn(strings.Join(missing, "; "), mo.Some(agentsRemedy))
}

// agentsEndAtCurrent is check 13: the top-level order, the review and consult blocks' own orders
// when they have one, and each per-harness order name at least one agent on current. An order without one can end
// with nothing to run when every external harness is missing or fails, and a project that wants
// exactly that stop is told what it has chosen.
func agentsEndAtCurrent(doc settings.Document, defined []settings.Agent) domain.Result {
	var without []string

	if !settings.HasCurrent(defined, settings.AgentNames(defined)) {
		without = append(without, settings.FieldAgents)
	}

	if order, ok := settings.ReviewAgents(doc).Get(); ok && !settings.HasCurrent(defined, order) {
		without = append(without, settings.BlockReview+"."+settings.FieldReviewAgents)
	}

	if order, ok := settings.ConsultAgents(doc).Get(); ok && !settings.HasCurrent(defined, order) {
		without = append(without, settings.BlockConsult+"."+settings.FieldConsultAgents)
	}

	byHarness := settings.AgentsByHarness(doc)
	for _, name := range slices.Sorted(maps.Keys(byHarness)) {
		if !settings.HasCurrent(defined, byHarness[name]) {
			without = append(without, settings.FieldAgentsByHarness+"."+name)
		}
	}

	if len(without) == 0 {
		return domain.AgentsCurrent.Pass()
	}

	return domain.AgentsCurrent.Warn(
		fmt.Sprintf("%s names no agent on current", strings.Join(without, ", ")), mo.Some(currentRemedy))
}
