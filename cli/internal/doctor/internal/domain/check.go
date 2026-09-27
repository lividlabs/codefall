// Package domain holds doctor's entities and value objects: what a check is, what its outcome looks
// like, and the rules for reading settings and token scopes. It performs no IO and names no
// delivery or infrastructure type (ADR-BASE-01).
package domain

import "github.com/samber/mo"

// Status is the outcome of a single check. A WARN is a condition worth reporting that does not stop
// codefall from working; a FAIL does.
type Status int

// The three outcomes, ordered from healthiest to worst.
const (
	StatusPass Status = iota
	StatusWarn
	StatusFail
)

// String returns the fixed-width word the report prints for the status.
func (s Status) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusWarn:
		return "WARN"
	case StatusFail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

// Category is the part of a project's setup a check belongs to. The report groups by category, so a
// healthy project is six lines rather than twenty-three.
type Category struct {
	ID    string
	Title string
}

// The six categories, in the order doctor reports them. Testing sits between Local environment and
// Beads for the same reason Local environment sits where it does: the first four are about the
// project, and the last two are about tools on the machine.
var (
	CategorySettings  = Category{ID: "settings", Title: "Settings"}
	CategoryHarnesses = Category{ID: "harnesses", Title: "Harnesses"}
	CategoryLocal     = Category{ID: "local", Title: "Local environment"}
	CategoryTesting   = Category{ID: "testing", Title: "Testing"}
	CategoryBeads     = Category{ID: "beads", Title: "Beads"}
	CategoryGitHub    = Category{ID: "github", Title: "GitHub CLI"}
)

// Check identifies one thing doctor looks at. The ID is stable and machine-readable; the Title is
// what a person reads; the Category is the section it is reported under.
type Check struct {
	ID       string
	Title    string
	Category Category
}

// Result is the outcome of running one check. Detail is absent on a bare PASS, and Remedy is absent
// whenever nothing useful can be suggested — doctor never invents a fix it does not believe in.
type Result struct {
	Check  Check
	Status Status
	Detail mo.Option[string]
	Remedy mo.Option[string]
}

// Pass records a check that succeeded with nothing worth saying about it.
func (c Check) Pass() Result {
	return Result{Check: c, Status: StatusPass, Detail: mo.None[string](), Remedy: mo.None[string]()}
}

// PassWithDetail records a check that succeeded and has something to report, such as a version.
func (c Check) PassWithDetail(detail string) Result {
	return Result{Check: c, Status: StatusPass, Detail: mo.Some(detail), Remedy: mo.None[string]()}
}

// Warn records a condition that does not stop codefall from working.
func (c Check) Warn(detail string, remedy mo.Option[string]) Result {
	return Result{Check: c, Status: StatusWarn, Detail: mo.Some(detail), Remedy: remedy}
}

// Fail records a condition that does stop codefall from working.
func (c Check) Fail(detail string, remedy mo.Option[string]) Result {
	return Result{Check: c, Status: StatusFail, Detail: mo.Some(detail), Remedy: remedy}
}

// The twenty-three checks doctor runs, in the order it runs them.
var (
	CodefallDir      = Check{ID: "codefall-dir", Title: ".codefall/ exists", Category: CategorySettings}
	SettingsFile     = Check{ID: "settings-file", Title: ".codefall/settings.json exists", Category: CategorySettings}
	SettingsJSON     = Check{ID: "settings-json", Title: "settings.json is valid JSON", Category: CategorySettings}
	SettingsComplete = Check{ID: "settings-complete", Title: "settings.json is complete", Category: CategorySettings}
	ReviewsIgnored   = Check{ID: "reviews-ignored", Title: ".ignore hides review findings", Category: CategorySettings}
	// TestsIgnored is the same question for the reports an agentic test run commits (ADR-007).
	TestsIgnored = Check{ID: "tests-ignored", Title: ".ignore hides test run reports", Category: CategorySettings}
	// StampIgnored is whether the refresh stamp, a per-machine file, is kept out of the repository.
	StampIgnored = Check{ID: "stamp-ignored", Title: ".gitignore hides the refresh stamp", Category: CategorySettings}
	// InteractionsMerged is whether bd's append-only interaction log is merged by union, so two
	// branches that both appended to it do not conflict.
	InteractionsMerged = Check{ID: "interactions-merged",
		Title: ".gitattributes merges bd's interaction log by union", Category: CategorySettings}
	// HarnessNames is whether the settings and the manifest name each harness the way codefall does
	// now: a harness is named for its binary, and two were named for their products before that.
	HarnessNames = Check{ID: "harness-names",
		Title: "harnesses are recorded under their current names", Category: CategoryHarnesses}
	// HarnessesInstalled is whether codefall's extension is actually where each harness the settings
	// record would read it.
	HarnessesInstalled = Check{ID: "harnesses-installed",
		Title: "codefall is installed for every harness", Category: CategoryHarnesses}
	// HarnessesLeftOver is the other direction: an install the manifest records for a harness the
	// settings no longer name.
	HarnessesLeftOver = Check{ID: "harnesses-leftover",
		Title: "no install is left over from a dropped harness", Category: CategoryHarnesses}
	// AgentsRunnable is whether every agent the settings define runs on a harness this machine can
	// start (ADR-009). A run skips one it cannot, so this warns.
	AgentsRunnable = Check{ID: "agents-runnable",
		Title: "every agent runs on a harness this machine can start", Category: CategoryHarnesses}
	// AgentsCurrent is whether every order of agents names one on current, the harness running the
	// session, so a run always has a reader it can start.
	AgentsCurrent = Check{ID: "agents-current",
		Title: "every agent order names one on the current harness", Category: CategoryHarnesses}
	// LocalDeclared is whether the settings name the project's start and update commands (ADR-005).
	LocalDeclared = Check{ID: "local-declared",
		Title: "the local start and update commands are declared", Category: CategoryLocal}
	// LocalRunnable is whether the program each declared command runs can be found: on PATH, or in
	// the project when the command names a path.
	LocalRunnable = Check{ID: "local-runnable",
		Title: "the local commands name programs that exist", Category: CategoryLocal}
	// TestDeclared is whether the settings say where the project's test cases live (ADR-007).
	TestDeclared = Check{ID: "test-declared",
		Title: "the testing root is declared", Category: CategoryTesting}
	// TestEquipped is whether a runner is declared, which is what says the harness has been set up.
	TestEquipped = Check{ID: "test-equipped",
		Title: "a test runner is declared", Category: CategoryTesting}
	// TestDirExists is whether the directory the project declared is there for a verb to find.
	TestDirExists = Check{ID: "test-dir-exists",
		Title: "the declared testing directory exists", Category: CategoryTesting}
	BeadsInstalled   = Check{ID: "bd-installed", Title: "bd is on PATH", Category: CategoryBeads}
	BeadsInitialized = Check{ID: "beads-initialized", Title: "Beads is initialized here", Category: CategoryBeads}
	GHInstalled      = Check{ID: "gh-installed", Title: "gh is on PATH", Category: CategoryGitHub}
	GHAuthenticated  = Check{ID: "gh-auth", Title: "gh is logged in to github.com", Category: CategoryGitHub}
	GHScopes         = Check{ID: "gh-scopes", Title: "gh token has the required scopes", Category: CategoryGitHub}
)
