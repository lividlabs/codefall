package domain

// Step identifies one unit of work an init run performs. The ID is stable and machine-readable; the
// Title is what a person reads while the step runs.
type Step struct {
	ID    string
	Title string
}

// The steps an init run performs, in the order it performs them. The settings file comes first, the
// harness extension second, Beads third, the hooks that tell the harness about Beads and guard the
// default branch fourth, and the sections that tell an agent how the project uses Beads, keeps its
// local environment current, and writes its test cases fifth. The hook step follows the extension
// step because the scripts the hooks run are what the extension step copies. The agents step follows
// Beads because bd init commits what it staged, and codefall's edit to AGENTS.md belongs in the
// author's own commit rather than bd's — which is the same reason the last two steps come last:
// the testing tree, .ignore, and .gitignore are the author's files to commit, not bd's.
//
// The cleanup step runs on an upgrade only, straight after the extension step: it compares what the
// previous finished run recorded with what the extension step just wrote and removes what this
// version no longer ships, so it has to follow the copy and precede the manifest (ADR-010). A first
// run has no record to compare and does not perform it.
//
// The testing step is sixth, after the agents step and before the ignore step. It belongs beside the
// agents step because the two write the same pair of documents under the same rule — an AGENTS.md
// and the CLAUDE.md that points at it — and it has to come before the ignore step, which keeps the
// testing root's run output out of the repository and cannot name a root before it is declared.
var (
	SettingsStep  = Step{ID: "settings", Title: "Writing .codefall/settings.json"}
	ExtensionStep = Step{ID: "extension", Title: "Installing the codefall extension"}
	CleanupStep   = Step{ID: "cleanup", Title: "Removing what this version no longer ships"}
	BeadsStep     = Step{ID: "beads", Title: "Initializing Beads"}
	HookStep      = Step{ID: "hook", Title: "Registering codefall's hooks"}
	AgentsStep    = Step{ID: "agents", Title: "Writing codefall's sections to AGENTS.md"}
	TestingStep   = Step{ID: "testing", Title: "Setting up the testing tree"}
	IgnoreStep    = Step{ID: "ignore", Title: "Writing the ignore and attributes entries"}
)

// Outcome is what a step did. A step that could not do its work returns an error instead: there is
// no failed outcome, because a failure stops the run.
//
// It has no String: presentation prints a mark rather than a word, so a name for each outcome would
// have nothing but a test to read it.
type Outcome int

// The two outcomes, in the order a reader is most likely to see them.
const (
	OutcomeDone Outcome = iota
	OutcomeSkipped
)

// StepResult is what one step did and the sentence that says so. Every result carries a detail:
// a step that ran has something to report, and a step that was skipped has a reason.
type StepResult struct {
	Step    Step
	Outcome Outcome
	Detail  string
}

// Done records a step that did its work.
func (s Step) Done(detail string) StepResult {
	return StepResult{Step: s, Outcome: OutcomeDone, Detail: detail}
}

// Skipped records a step that found its work already done and left it alone.
func (s Step) Skipped(detail string) StepResult {
	return StepResult{Step: s, Outcome: OutcomeSkipped, Detail: detail}
}

// Report is the ordered outcome of one init run. A step that never ran, because an earlier one
// failed, is absent from the report rather than present with an outcome of its own.
type Report struct {
	results []StepResult
}

// NewReport builds a report from results already in step order.
func NewReport(results ...StepResult) Report {
	return Report{results: results}
}

// Results returns the results in step order.
func (r Report) Results() []StepResult {
	return r.results
}
