// Package settings is the one definition of the .codefall/settings.json format: the constants the
// file carries, the tables of what each tracker's block may hold, and the rules that decide whether
// a document or a single field is acceptable. Both components need it — initcmd writes that file and
// doctor reads it — so it lives here rather than in either one's domain (ADR-003).
//
// It is a pure shared module: it imports the standard library and nothing else, which is what lets
// domain/ and application/ name it. Adding a dependency here breaks that permission and fails the
// pure-shared-modules rule in .golangci.yml.
//
// What belongs here is the format. How a component builds a settings value from a survey's answers,
// or reports on one it read, belongs to that component.
package settings

import (
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// Document is a decoded settings file: the generic shape a JSON decoder produces (strings, float64
// numbers, bools, nil, and nested maps). Validation reads that shape without knowing it came from
// JSON, so the field tables below are the one definition of what settings may contain.
type Document map[string]any

// The published constants of the settings format. The schema test in this package holds
// schemas/settings.schema.json equal to these, so the two definitions cannot drift apart silently.
const (
	Version       = 1
	TrackerBeads  = "beads"
	TrackerGitHub = "github"
	RepoPattern   = `^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`
	SchemaID      = "https://raw.githubusercontent.com/lividlabs/codefall-cli/main/schemas/settings.schema.json"
	BlockReview   = "review"
	// BlockConsult is what a verb reaches for when a run cannot settle a question on its own
	// (ADR-009). It carries only an order of agents; absent, the top-level order applies.
	BlockConsult = "consult"
	// BlockLocal is the project's two local-environment commands (ADR-005): what brings the services
	// it develops against up, and what makes the local environment match the checkout. Both are
	// shell commands run from the project root, so a project may point at a Makefile target, a
	// package script, or a script of its own.
	BlockLocal       = "local"
	FieldLocalStart  = "start"
	FieldLocalUpdate = "update"
	// BlockTest is what the project has declared about its test cases (ADR-007): where they live,
	// and which runners are installed to run them. The division is by reader — what a program reads
	// is here, and what an agent reads is in the testing root's own AGENTS.md — so the block holds
	// the runner's name and never the command that runs it.
	BlockTest    = "test"
	FieldTestDir = "dir"
	// FieldTestRunners is written by codefall-equip rather than by hand: a runner is declared once it
	// is installed, and installing it is what equip does.
	FieldTestRunners = "runners"
	// DefaultTestDir is the root a project takes when it has no reason to choose another. codefall
	// init offers it as the answer and nothing infers it at run time.
	DefaultTestDir = "testing"
	// TestDirPattern is how a testing root may be written: path segments of ordinary file-name
	// characters, each beginning with one that is not a dot. That makes every root relative — no
	// leading slash and no drive letter — and leaves no way to write "." or ".." as a segment, so a
	// declared root cannot climb out of the project that declared it.
	TestDirPattern = `^[A-Za-z0-9_-][A-Za-z0-9._-]*(/[A-Za-z0-9_-][A-Za-z0-9._-]*)*$`
	// The runners codefall knows. A runner is chosen per surface: Playwright for a browser front end,
	// an Electron shell, or an HTTP API; go test for a Go surface (ADR-007).
	RunnerPlaywright = "playwright"
	RunnerGoTest     = "go-test"
	// FieldHarnesses is the harnesses a project is set up for. It is required: which harnesses a
	// project uses is a decision the project made, and codefall cannot work it out from which
	// directories happen to exist — several harnesses share one, and codefall writes those
	// directories itself.
	FieldHarnesses = "harnesses"
)

// The .ignore file, which is not settings but is the other file init writes and doctor checks — so
// it lives here for the same reason the settings format does (ADR-003).
//
// codefall-review commits its findings so that patterns across reviews stay visible, and an agentic
// test run commits its report for the same reason (ADR-007), which puts both in the same tree as the
// code. Ripgrep reads .ignore and git does not, so one line each keeps them out of every search that
// goes through ripgrep while leaving them tracked.
const (
	// IgnoreName is the file, beside .codefall/ rather than at the repository root: a run below the
	// root installs there, and ripgrep reads .ignore files down the tree.
	IgnoreName = ".ignore"
	// IgnoreEntry is the line itself.
	IgnoreEntry = ".codefall/reviews/"
	// IgnoreComment says why the line is there, for whoever finds the file later.
	IgnoreComment = "# codefall review findings: tracked in git, skipped by ripgrep."
	// IgnoreEntryTests is the same treatment for the run reports codefall-test commits, which are
	// prose about a run and would otherwise be read as part of the codebase.
	IgnoreEntryTests   = ".codefall/tests/"
	IgnoreTestsComment = "# codefall test run reports: tracked in git, skipped by ripgrep."
)

// The refresh stamp and the .gitignore entry that keeps it out of the repository (ADR-005). The
// stamp is the commit the local environment was last brought current at, on this machine: refresh
// writes it, preflight reads it, and it means nothing on any other machine, so it is never
// committed. init adds the entry and doctor warns when it is missing.
const (
	// GitIgnoreName is git's own ignore file, at the same level as .codefall/.
	GitIgnoreName = ".gitignore"
	// RefreshStamp is the stamp's path, which is also the .gitignore entry.
	RefreshStamp = ".codefall/refresh.stamp"
	// GitIgnoreComment says why the line is there, for whoever finds the file later.
	GitIgnoreComment = "# codefall refresh stamp: the commit this machine's local environment was last brought current at."
	// TestArtifactsComment says why a testing root's run output is kept out of the repository: it is
	// what a run produced on one machine, and the report beside it is what the run had to say
	// (ADR-007).
	TestArtifactsComment = "# codefall test run output: logs, traces, reports, and whatever a run created."
)

// bd's optional interaction log, and the .gitattributes entry that keeps it from conflicting. With
// `audit.enabled` set, bd appends one JSON line per agent interaction to .beads/interactions.jsonl,
// and the file is committed alongside whatever bead work it accompanies. Two branches that each
// appended to it conflict on every merge unless git is told the file is append-only, which is what
// merge=union says: keep both sides, in order. init adds the entry whether or not the log is on,
// because the cost of a line nothing reads is nothing, and doctor warns when it is missing.
const (
	// GitAttributesName is git's attributes file, at the same level as .gitignore.
	GitAttributesName = ".gitattributes"
	// InteractionsAttribute is the line itself: the path, and the merge driver for it.
	InteractionsAttribute = ".beads/interactions.jsonl merge=union"
	// GitAttributesComment says why the line is there, for whoever finds the file later.
	GitAttributesComment = "# bd's interaction log is append-only: a merge keeps both sides instead of conflicting."
)

// TestArtifacts is the .gitignore entry for a testing root's run output. The root is the project's
// to choose, so the entry is built from it rather than written down.
func TestArtifacts(root string) string {
	return strings.TrimSuffix(root, "/") + "/.artifacts/"
}

// NamesEntry reports whether an ignore file's contents already name an entry.
//
// The comparison is per line and ignores surrounding space, so an entry someone indented still
// counts and a commented-out one does not — ripgrep does not read the comment either. Both
// components need the same answer: init decides whether to append, doctor decides whether to warn.
func NamesEntry(body, entry string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(line) == entry {
			return true
		}
	}

	return false
}

var (
	repoRegexp    = regexp.MustCompile(RepoPattern)
	testDirRegexp = regexp.MustCompile(TestDirPattern)
)

// fieldSpec is one row of the settings shape. Check returns "" when the value is acceptable and the
// reason it is not otherwise.
type fieldSpec struct {
	Name     string
	Required bool
	Check    func(v any) string
}

// topLevelFields is the settings file's own shape. Definition order is report order.
//
// The review, local, and test blocks are optional at the top level and complete when they are there:
// a project set up before a block existed is still valid settings, and one that carries it carries
// every field. The agents list and its per-harness override are optional too, and absent means the
// default (ADR-009).
var topLevelFields = []fieldSpec{
	{"$schema", false, isString},
	{"version", true, isVersion},
	{"tracker", true, isTracker},
	{FieldHarnesses, true, isHarnesses},
	{FieldAgents, false, isAgents},
	{FieldAgentsByHarness, false, isObject},
	{BlockReview, false, isObject},
	{BlockConsult, false, isObject},
	{BlockLocal, false, isObject},
	{BlockTest, false, isObject},
}

// reviewFields is the shape of the review block, which codefall-review reads and nothing else
// writes. Posting findings to a pull request is visible to everyone on it, so the field exists to
// make that a decision the project made rather than a default it inherited.
//
// The block may also carry its own order of agents, an ordered subset of the top-level list, when
// review should walk a different order from every other use (ADR-009).
var reviewFields = []fieldSpec{
	{"postToPullRequest", true, isBool},
	{FieldReviewAgents, false, isNameList},
}

// consultFields is the shape of the consult block: nothing but its own order of agents, and that
// optional, because the block exists only to walk a different order from every other use.
var consultFields = []fieldSpec{
	{FieldConsultAgents, false, isNameList},
}

// localFields is the shape of the local block. Both commands are required once the block is there:
// update assumes start has run, so a project that declares one without the other has declared
// half of a contract (ADR-005).
var localFields = []fieldSpec{
	{FieldLocalStart, true, isString},
	{FieldLocalUpdate, true, isString},
}

// Local is the local block's two commands, read out of a document that Validate accepts.
type Local struct {
	Start  string
	Update string
}

// testFields is the shape of the test block. The directory is required once the block is there —
// nothing else in the block means anything without it, and codefall init is what asks for it — while
// the runners are optional, because a project declares its root before it is equipped and
// codefall-equip writes the runners afterwards (ADR-007).
var testFields = []fieldSpec{
	{FieldTestDir, true, isTestDir},
	{FieldTestRunners, false, isRunners},
}

// Test is the test block's declaration, read out of a document that Validate accepts: where the
// cases live, and which runners are installed. Runners is empty for a project that has declared a
// root and is not equipped yet.
type Test struct {
	Dir     string
	Runners []string
}

// testRunners is every runner codefall knows, as a set, so the validator and the schema read one
// definition.
var testRunners = map[string]bool{
	RunnerPlaywright: true,
	RunnerGoTest:     true,
}

// TestRunners returns the known runner names, sorted.
func TestRunners() []string {
	return slices.Sorted(maps.Keys(testRunners))
}

// trackerFields is the shape of each known tracker's block, keyed by the value of "tracker" that
// selects it. Adding a tracker is one row here plus one oneOf branch in the schema; the
// "exactly one tracker block" rule then applies to it without further code.
var trackerFields = map[string][]fieldSpec{
	// bd keeps its own configuration under .beads/, so the block carries no fields of its own today;
	// it exists so the "exactly one tracker block" rule applies to it uniformly.
	TrackerBeads: {},
	// The fields say what they point at rather than which forge holds it: the block is already named
	// github, and "repo" alone read as the repository being set up rather than the one issues are
	// filed in — which need not be the same repository.
	TrackerGitHub: {
		{"issuesRepo", true, matches(repoRegexp, "owner/name")},
		{"issuesProject", false, isPositiveInteger},
	},
}

// Trackers returns the known tracker names, sorted.
func Trackers() []string {
	return slices.Sorted(maps.Keys(trackerFields))
}

// RequiredFields returns the top-level fields settings must carry, in definition order.
func RequiredFields() []string {
	return requiredNames(topLevelFields)
}

// RequiredTrackerFields returns the fields the named tracker's block must carry, in definition
// order. An unknown tracker has none.
func RequiredTrackerFields(tracker string) []string {
	return requiredNames(trackerFields[tracker])
}

// RequiredReviewFields returns the fields the review block must carry once it is present, in
// definition order.
func RequiredReviewFields() []string {
	return requiredNames(reviewFields)
}

// RequiredLocalFields returns the fields the local block must carry once it is present, in
// definition order.
func RequiredLocalFields() []string {
	return requiredNames(localFields)
}

// RequiredTestFields returns the fields the test block must carry once it is present, in definition
// order.
func RequiredTestFields() []string {
	return requiredNames(testFields)
}

// Harnesses returns the harnesses a document names, in the order it names them. A document whose
// harnesses field is missing or not a list of names yields none; Validate is what says so.
func Harnesses(doc Document) []string {
	values, _ := doc[FieldHarnesses].([]any)

	names := make([]string, 0, len(values))

	for _, value := range values {
		if name, ok := value.(string); ok {
			names = append(names, name)
		}
	}

	return names
}

// LocalCommands returns the local block's commands when the document declares a block with both,
// and None otherwise. A block Validate would reject — missing a field, or holding something other
// than a string — is None here too: the caller has already been told what is wrong with it, and a
// half-declared block is not a declaration (ADR-005).
func LocalCommands(doc Document) mo.Option[Local] {
	value, present := lookup(doc, BlockLocal)
	if !present {
		return mo.None[Local]()
	}

	block, ok := value.(map[string]any)
	if !ok {
		return mo.None[Local]()
	}

	start, startOK := lookup(block, FieldLocalStart)
	update, updateOK := lookup(block, FieldLocalUpdate)

	if !startOK || !updateOK {
		return mo.None[Local]()
	}

	startText, startIsText := start.(string)
	updateText, updateIsText := update.(string)

	if !startIsText || !updateIsText {
		return mo.None[Local]()
	}

	return mo.Some(Local{Start: startText, Update: updateText})
}

// TestDeclaration returns what the document declares about testing, and None when it declares
// nothing. A block Validate would reject — no directory, a directory that is not a relative path, a
// runner codefall does not know — is None here too, by the same rule the local block follows: the
// caller has already been told what is wrong with it, and a block nothing can act on is not a
// declaration (ADR-007).
func TestDeclaration(doc Document) mo.Option[Test] {
	value, present := lookup(doc, BlockTest)
	if !present {
		return mo.None[Test]()
	}

	block, ok := value.(map[string]any)
	if !ok {
		return mo.None[Test]()
	}

	dir, dirPresent := lookup(block, FieldTestDir)
	if !dirPresent {
		return mo.None[Test]()
	}

	root, isText := dir.(string)
	if !isText || isTestDir(root) != "" {
		return mo.None[Test]()
	}

	declared := Test{Dir: root}

	runners, listed := block[FieldTestRunners]
	if !listed || runners == nil {
		return mo.Some(declared)
	}

	if isRunners(runners) != "" {
		return mo.None[Test]()
	}

	values, _ := runners.([]any)
	for _, value := range values {
		name, _ := value.(string)
		declared.Runners = append(declared.Runners, name)
	}

	return mo.Some(declared)
}

func requiredNames(fields []fieldSpec) []string {
	names := []string{}

	for _, field := range fields {
		if field.Required {
			names = append(names, field.Name)
		}
	}

	return names
}

// ParseTracker returns the tracker name when it is one codefall knows, and an error listing the
// known ones when it is not.
func ParseTracker(name string) (string, error) {
	if _, known := trackerFields[name]; known {
		return name, nil
	}

	return "", fmt.Errorf("unknown tracker %q (known trackers: %s)", name, strings.Join(Trackers(), ", "))
}

// ValidateRepo reports whether a repository is written the way GitHub names one.
//
// This and ValidateProject are the field-level halves of what Validate checks across a whole
// document. They are separate because a form validates a field as it is typed, before there are
// enough answers to have a document at all — and because what a person reads while typing says how
// to write the value, where a report says which path is wrong.
func ValidateRepo(repo string) error {
	if !repoRegexp.MatchString(repo) {
		return fmt.Errorf("repository %q must be written as owner/name", repo)
	}

	return nil
}

// ValidateTestDir reports whether a testing root is written the way the format accepts one. It is
// the field-level half of the table's own check, for the form that validates the answer as it is
// typed, before there is a document to hold it.
func ValidateTestDir(dir string) error {
	if !testDirRegexp.MatchString(dir) {
		return fmt.Errorf("testing directory %q must be a relative path inside the project, such as %q",
			dir, DefaultTestDir)
	}

	return nil
}

// ValidateProject reports whether a GitHub project number could identify a project.
func ValidateProject(number int) error {
	if number < 1 {
		return fmt.Errorf("project number %d must be a positive integer", number)
	}

	return nil
}

// Validate reports every problem with a settings document at once, each as "path: reason", in
// definition order. An empty result means the settings are complete.
func Validate(doc Document) []string {
	var problems []string

	selected := ""
	// The top-level fields that were there and were wrong. A block the top level has already called
	// something other than an object is not looked inside, which would only say so a second time.
	rejected := map[string]bool{}

	for _, field := range topLevelFields {
		value, ok := lookup(doc, field.Name)
		if !ok {
			if field.Required {
				problems = append(problems, field.Name+": missing")
			}

			continue
		}

		if reason := field.Check(value); reason != "" {
			problems = append(problems, field.Name+": "+reason)
			rejected[field.Name] = true

			continue
		}

		if field.Name == "tracker" {
			selected, _ = value.(string)
		}
	}

	// The review, local, and test blocks are independent of the tracker, so they are checked before
	// the tracker's own block decides whether there is anything further to say.
	for _, block := range []struct {
		name   string
		fields []fieldSpec
	}{
		{BlockReview, reviewFields},
		{BlockConsult, consultFields},
		{BlockLocal, localFields},
		{BlockTest, testFields},
	} {
		if _, present := lookup(doc, block.name); present && !rejected[block.name] {
			problems = append(problems, validateBlock(doc, block.name, block.fields)...)
		}
	}

	// The orders name agents the top-level list defines, which no field's own check can see.
	problems = append(problems, agentReferences(doc, rejected)...)

	// A missing or unknown tracker selects no block, so there is nothing further to say.
	if _, known := trackerFields[selected]; !known {
		return problems
	}

	if _, present := lookup(doc, selected); !present {
		problems = append(problems,
			fmt.Sprintf("%s: missing (required when tracker is %q)", selected, selected))

		return problems
	}

	problems = append(problems, validateBlock(doc, selected, trackerFields[selected])...)

	for _, other := range Trackers() {
		if other == selected {
			continue
		}

		if _, present := doc[other]; present {
			problems = append(problems, fmt.Sprintf("%s: present but tracker is %q — remove it", other, selected))
		}
	}

	return problems
}

// validateBlock checks one present block against its field table. The tracker's block and the
// review block have the same shape — a name, an object, a table of fields — so they are one
// function. Whether a block being absent is a problem differs between them and is decided by the
// caller, which is why this one is only ever called on a block that is there.
func validateBlock(doc Document, name string, fields []fieldSpec) []string {
	var problems []string

	value, ok := lookup(doc, name)
	if !ok {
		return nil
	}

	block, ok := value.(map[string]any)
	if !ok {
		return []string{name + ": must be an object"}
	}

	for _, field := range fields {
		path := name + "." + field.Name

		fieldValue, present := lookup(block, field.Name)
		if !present {
			if field.Required {
				problems = append(problems, path+": missing")
			}

			continue
		}

		if reason := field.Check(fieldValue); reason != "" {
			problems = append(problems, path+": "+reason)
		}
	}

	return problems
}

// lookup reports whether a key carries a usable value. A null and an empty string both count as
// missing, so a half-filled template reads the same as an absent field.
func lookup(doc map[string]any, name string) (any, bool) {
	value, ok := doc[name]
	if !ok || value == nil {
		return nil, false
	}

	if text, isText := value.(string); isText && text == "" {
		return nil, false
	}

	return value, true
}

func isString(v any) string {
	if _, ok := v.(string); !ok {
		return "must be a string"
	}

	return ""
}

func isBool(v any) string {
	if _, ok := v.(bool); !ok {
		return "must be true or false"
	}

	return ""
}

func isObject(v any) string {
	if _, ok := v.(map[string]any); !ok {
		return "must be an object"
	}

	return ""
}

func isVersion(v any) string {
	number, ok := wholeNumber(v)
	if !ok || number != Version {
		return fmt.Sprintf("must be %d", Version)
	}

	return ""
}

func isTracker(v any) string {
	name, ok := v.(string)
	if !ok {
		return "must be a string"
	}

	if _, known := trackerFields[name]; !known {
		return unknownValue(name, Trackers())
	}

	return ""
}

// isHarnesses accepts the harnesses a project is set up for: at least one, each one codefall can set
// up. An empty list is refused rather than read as "none", because a project codefall sets up for no
// harness at all has nowhere to install.
//
// A name is checked through harness.Parse, so a spelling a harness had before it was named for its
// binary is accepted: the file is still valid, doctor warns about the spelling, and codefall init
// rewrites it. The schema lists only the current names, and the message for a name nobody knows
// offers only those.
func isHarnesses(v any) string {
	values, ok := v.([]any)
	if !ok {
		return "must be an array of harness names"
	}

	if len(values) == 0 {
		return "must name at least one harness"
	}

	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			return "must be an array of harness names"
		}

		if _, err := harness.Parse(name); err != nil {
			return unknownValue(name, harness.All())
		}
	}

	return ""
}

// isTestDir accepts the testing root: a relative path inside the project, which is what makes it
// mean the same thing to every machine the project is checked out on.
func isTestDir(v any) string {
	dir, ok := v.(string)
	if !ok {
		return "must be a string"
	}

	if !testDirRegexp.MatchString(dir) {
		return fmt.Sprintf("must be a relative path inside the project, such as %q", DefaultTestDir)
	}

	return ""
}

// isRunners accepts the runners a project has installed: any number of them, each one codefall
// knows, and none of them twice. An empty list is accepted rather than refused, because it is what a
// project that has declared a testing root and not been equipped yet looks like — doctor reports
// that and codefall-equip fills it in (ADR-007).
func isRunners(v any) string {
	values, ok := v.([]any)
	if !ok {
		return "must be an array of runner names"
	}

	seen := map[string]bool{}

	for _, value := range values {
		name, ok := value.(string)
		if !ok {
			return "must be an array of runner names"
		}

		if !testRunners[name] {
			return unknownValue(name, TestRunners())
		}

		if seen[name] {
			return fmt.Sprintf("names %q twice", name)
		}

		seen[name] = true
	}

	return ""
}

// unknownValue is how both closed sets of names report a value that is not one of them: the value it
// read, and the ones that would have been accepted, because that is what the reader needs next.
func unknownValue(name string, expected []string) string {
	quoted := make([]string, 0, len(expected))
	for _, value := range expected {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return fmt.Sprintf("unknown value %q (expected %s)", name, strings.Join(quoted, ", "))
}

func isPositiveInteger(v any) string {
	number, ok := wholeNumber(v)
	if !ok || number < 1 {
		return "must be a positive integer"
	}

	return ""
}

func matches(pattern *regexp.Regexp, shape string) func(v any) string {
	return func(v any) string {
		text, ok := v.(string)
		if !ok {
			return "must be a string"
		}

		if !pattern.MatchString(text) {
			return "must match " + shape
		}

		return ""
	}
}

// wholeNumber accepts a JSON number that has no fractional part. JSON has one number type, so an
// integer arrives as a float64 and 1.5 has to be rejected here rather than by the decoder.
func wholeNumber(v any) (int, bool) {
	number, ok := v.(float64)
	if !ok || number != math.Trunc(number) {
		return 0, false
	}

	return int(number), true
}
