package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// The trackers the survey shows but does not accept. Huh has no disabled option, so they are offered
// with the state in their label and refused by the field's own validation — which is more use to a
// reader than leaving them out, because "not yet" is the answer they are looking for.
const (
	trackerJira   = "jira"
	trackerLinear = "linear"
)

// NewInitCommand builds `codefall init`, the run that happens once: it makes a directory ready for
// codefall, and a project that already has .codefall/manifest.json is `codefall upgrade`'s.
func NewInitCommand(initialize InitializeUseCase) *cobra.Command {
	flags := &initFlags{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set this directory up for codefall",
		Long: "Creates .codefall/settings.json from your answers, including which coding harnesses the " +
			"project uses. Every question is also a flag, so a scripted run passes them and is never " +
			"prompted. The codefall extension is installed for each harness chosen: Claude Code gets it " +
			"at project scope into .claude/settings.json, and a harness that reads the .agents/skills " +
			"convention gets the extension's tree under .agents/. It also asks where the project's " +
			"test cases live and creates that tree. " +
			"Init runs once: a project that already has .codefall/manifest.json is brought current by " +
			"codefall upgrade, which init names and refuses to repeat. " +
			"Run below the root of a git repository, init asks whether to install there or at the " +
			"root; --location answers without asking.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd, initialize, flags)
		},
	}

	flags.register(cmd)

	return cmd
}

// initFlags is every value the survey asks for, plus where to install. Each prompted value is a flag,
// which is what keeps the command usable from a script (ADR-002).
type initFlags struct {
	tracker        string
	issuesRepo     string
	issuesProject  int
	reviewPostToPR bool
	harnesses      []string
	testDir        string
	location       string
}

func (f *initFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.tracker, "tracker", "",
		"issue tracker to use ("+strings.Join(settings.Trackers(), ", ")+")")
	cmd.Flags().StringVar(&f.issuesRepo, "issues-repo", "",
		"repository whose issues the project files against, as owner/name on GitHub (defaults to the "+
			"repository this directory belongs to; required when the tracker is "+settings.TrackerGitHub+
			" and that cannot be worked out)")
	cmd.Flags().IntVar(&f.issuesProject, "issues-project", 0,
		"number of the GitHub Project those issues are organised into (optional)")
	cmd.Flags().BoolVar(&f.reviewPostToPR, "review-post-to-pr", false,
		"let codefall-review post its findings to a pull request (optional)")
	registerHarnessFlag(cmd, &f.harnesses, "coding harness to set up")
	// No default: a project that has not declared a testing root is asked. A default here would
	// answer for the project on a run that could still ask.
	cmd.Flags().StringVar(&f.testDir, "test-dir", "",
		"directory the project's test cases live in, relative to this one (default "+
			settings.DefaultTestDir+")")
	registerLocationFlag(cmd, &f.location)
}

// runInit is the command's body: collect the answers, run the steps, say what happened.
func runInit(cmd *cobra.Command, initialize InitializeUseCase, flags *initFlags) error {
	// The working directory is the one piece of environment the command reads; the use case
	// receives it as a value.
	dir, err := os.Getwd()
	if err != nil {
		return initKind.wrap(err)
	}

	// A bad flag, an unanswerable question, or a project that is already set up is the user's own
	// message; it is returned as it is so Fang renders that sentence and not a prefix in front of it.
	request, err := buildRequest(cmd, initialize, flags, dir)
	if err != nil {
		return err
	}

	// One colour-profile writer for the whole run.
	out := ui.NewWriter(cmd.OutOrStdout())

	if _, err := runInitialize(cmd.Context(), initKind, initialize, request, out); err != nil {
		return initKind.wrap(err)
	}

	return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(nextStep))
}

// buildRequest turns the flags into the use case's contract, asking for whatever they left out.
func buildRequest(
	cmd *cobra.Command, initialize InitializeUseCase, flags *initFlags, dir string,
) (application.Request, error) {
	chosen, err := parseHarnesses(flags.harnesses)
	if err != nil {
		return application.Request{}, err
	}

	// Where the run installs comes before everything else, because every other question — whether
	// there is a manifest, whether settings exist — is a question about that directory.
	dir, err = chooseLocation(cmd.Context(), initKind, initialize, flags.location, dir)
	if err != nil {
		return application.Request{}, err
	}

	request := application.Request{Dir: dir, Harnesses: chosen, CLIVersion: buildinfo.Version()}

	if flags.tracker != "" {
		tracker, err := settings.ParseTracker(flags.tracker)
		if err != nil {
			return application.Request{}, err
		}

		request.Tracker = tracker
	}

	if flags.issuesRepo != "" {
		if err := settings.ValidateRepo(flags.issuesRepo); err != nil {
			return application.Request{}, err
		}

		request.IssuesRepo = mo.Some(flags.issuesRepo)
	}

	if flags.testDir != "" {
		if err := settings.ValidateTestDir(flags.testDir); err != nil {
			return application.Request{}, err
		}

		request.TestDir = flags.testDir
	}

	// Same reasoning as the project number below: a bool flag left alone and one set to false are
	// different answers, and only the flag's own record of being set separates them.
	if cmd.Flags().Changed("review-post-to-pr") {
		request.ReviewPostToPullRequest = mo.Some(flags.reviewPostToPR)
	}

	// A project number is optional, so "not given" and "given as zero" are different answers and
	// only the flag's own record of being set can tell them apart.
	if cmd.Flags().Changed("issues-project") {
		if flags.issuesProject < 1 {
			// The flag name is never the message's first word: Fang title-cases it before
			// rendering, which would turn --issues-project into --Issues-Project.
			return application.Request{}, fmt.Errorf(
				"the --issues-project flag must be a positive integer, not %d", flags.issuesProject)
		}

		request.IssuesProject = mo.Some(flags.issuesProject)
	}

	// A manifest is a finished run's record, and init runs once. Everything a second run would do
	// is upgrade's, which reads the manifest this command would otherwise have to guess around.
	recorded, err := initialize.ManifestExists(dir)
	if err != nil {
		return application.Request{}, initKind.wrap(err)
	}

	if recorded {
		return application.Request{}, errors.New(
			"codefall is already set up here; run codefall upgrade to bring the install current")
	}

	// Settings with no manifest behind them are a project set up before the manifest existed, and
	// this is its one run of init: nothing is asked, because the answers are already on record, and
	// the run finishes by writing the manifest that sends every later run to upgrade.
	settled, err := initialize.SettingsExist(dir)
	if err != nil {
		return application.Request{}, initKind.wrap(err)
	}

	if settled {
		if len(request.Harnesses) == 0 {
			recorded, err := initialize.ChosenHarnesses(dir)
			if err != nil {
				return application.Request{}, initKind.wrap(err)
			}

			names, ok := recorded.Get()
			if !ok {
				return application.Request{}, errors.New(
					"the settings here record no harnesses; pass --harness")
			}

			request.Harnesses = names
		}

		// The testing root is read back the same way, and for the same reason: a project that has
		// declared one is not asked about it again, and this run never moves it (ADR-007).
		if request.TestDir == "" {
			declared, err := initialize.DeclaredTestDir(dir)
			if err != nil {
				return application.Request{}, initKind.wrap(err)
			}

			request.TestDir = declared.OrEmpty()
		}
	} else {
		request, err = collect(cmd.Context(), initialize, request)
		if err != nil {
			return application.Request{}, err
		}
	}

	// The tracker is settled by now, whether a flag or the survey chose it, so this is the last
	// place the two answers can be held against each other.
	if err := rejectGitHubFlags(cmd, request.Tracker); err != nil {
		return application.Request{}, err
	}

	return request, nil
}

// rejectGitHubFlags refuses the GitHub flags on a tracker that does not use them, in the terms the
// person typed. The domain refuses the same combination as an invariant, but it does so two layers
// away and in terms of settings, in a sentence that names neither flag; this is the sentence that
// does.
//
// "the" opens the sentence rather than the flag name: Fang title-cases the first word of every
// error it renders, which would turn --issues-repo into --Issues-Repo.
func rejectGitHubFlags(cmd *cobra.Command, tracker string) error {
	if tracker == "" || tracker == settings.TrackerGitHub {
		return nil
	}

	for _, name := range []string{"issues-repo", "issues-project"} {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("the --%s flag is only used with --tracker %s", name, settings.TrackerGitHub)
		}
	}

	return nil
}

// collect fills in the answers the flags did not supply, by asking a person when there is one and by
// failing with the flag's name when there is not (ADR-002).
func collect(
	ctx context.Context, initialize InitializeUseCase, request application.Request,
) (application.Request, error) {
	if !needsAnswers(request) {
		return request, nil
	}

	// gh knows what repository this directory belongs to, which makes it an answer rather than a
	// question: it pre-fills the field when someone is answering, and stands in for the flag when
	// nobody is.
	suggestion := mo.None[string]()
	if mightUseGitHub(request) {
		suggestion = initialize.SuggestIssuesRepo(ctx, request.Dir)
	}

	if !stdinIsTerminal() {
		return withoutPrompting(request, suggestion)
	}

	return survey(ctx, request, suggestion)
}

// needsAnswers reports whether anything the settings cannot be built without is still missing. The
// project number is not one of those, so it is never on its own a reason to prompt.
func needsAnswers(request application.Request) bool {
	return len(request.Harnesses) == 0 ||
		request.Tracker == "" ||
		request.TestDir == "" ||
		(request.Tracker == settings.TrackerGitHub && request.IssuesRepo.IsAbsent())
}

// mightUseGitHub reports whether a repository could still be wanted — either because the tracker is
// GitHub, or because it has not been chosen yet and might be.
func mightUseGitHub(request application.Request) bool {
	return request.IssuesRepo.IsAbsent() &&
		(request.Tracker == "" || request.Tracker == settings.TrackerGitHub)
}

// withoutPrompting is the path a script, CI, or an agent takes: what is known is used, and what is
// missing is an error naming the flag that would have supplied it. Nothing waits on input that
// cannot arrive.
func withoutPrompting(
	request application.Request, suggestion mo.Option[string],
) (application.Request, error) {
	if len(request.Harnesses) == 0 {
		return application.Request{}, missingFlag("--harness")
	}

	if request.Tracker == "" {
		return application.Request{}, missingFlag("--tracker")
	}

	if request.Tracker == settings.TrackerGitHub && request.IssuesRepo.IsAbsent() {
		repo, ok := suggestion.Get()
		if !ok {
			return application.Request{}, missingFlag("--issues-repo")
		}

		request.IssuesRepo = mo.Some(repo)
	}

	// The default is what the survey offers, not what a scripted run gets: where a project's test
	// cases live is a decision the project makes, and nothing infers it (ADR-007).
	if request.TestDir == "" {
		return application.Request{}, missingFlag("--test-dir")
	}

	return request, nil
}

func missingFlag(flag string) error {
	return fmt.Errorf("missing %s (stdin is not a terminal)", flag)
}

// survey asks for the answers that are still missing and nothing else: a question whose flag was
// given is not asked again.
func survey(
	ctx context.Context, request application.Request, suggestion mo.Option[string],
) (application.Request, error) {
	// GitHub Issues is the first option and the starting value, so the highlighted answer is the
	// one most projects want.
	tracker := settings.TrackerGitHub
	if request.Tracker != "" {
		tracker = request.Tracker
	}

	repo := request.IssuesRepo.OrElse(suggestion.OrEmpty())

	project := ""
	if number, ok := request.IssuesProject.Get(); ok {
		project = strconv.Itoa(number)
	}

	postToPR := request.ReviewPostToPullRequest.OrElse(false)

	harnesses := request.Harnesses

	testDir := request.TestDir
	if testDir == "" {
		testDir = settings.DefaultTestDir
	}

	var groups []*huh.Group

	// Which harnesses the project uses comes first: it is the question the rest of the install
	// depends on, and the one only the person answering can settle.
	if len(request.Harnesses) == 0 {
		groups = append(groups, huh.NewGroup(harnessField(&harnesses)))
	}

	if request.Tracker == "" {
		groups = append(groups, huh.NewGroup(trackerField(&tracker)))
	}

	// The GitHub questions are a group of their own so they can disappear: which tracker is chosen
	// may only be known once the form is running.
	if fields := gitHubFields(request, &repo, &project); len(fields) > 0 {
		groups = append(groups, huh.NewGroup(fields...).
			WithHideFunc(func() bool { return tracker != settings.TrackerGitHub }))
	}

	if request.ReviewPostToPullRequest.IsAbsent() {
		groups = append(groups, huh.NewGroup(reviewPostToPRField(&postToPR)))
	}

	// Last, because it is the one question about a directory this run makes rather than about what
	// codefall reads, and because the answer is already on the line: the default is offered as the
	// value, so the question is one keypress for a project that has no reason to move it.
	if request.TestDir == "" {
		groups = append(groups, huh.NewGroup(testDirField(&testDir)))
	}

	if err := runForm(ctx, initKind, groups); err != nil {
		return application.Request{}, err
	}

	request.ReviewPostToPullRequest = mo.Some(postToPR)
	request.Harnesses = harnesses
	request.TestDir = strings.TrimSpace(testDir)

	return answered(request, tracker, repo, project)
}

// testDirField asks where the project's test cases live. The default is the starting value rather
// than a silent fallback: a project that wants `e2e/` or a directory inside one package says so
// here, and one that does not presses enter.
func testDirField(dir *string) huh.Field {
	return huh.NewInput().
		Title("Where should the project's test cases live?").
		Description("codefall creates the directory, its test-cases/ folder, and skeleton AGENTS.md and README.md files.").
		Placeholder(settings.DefaultTestDir).
		Value(dir).
		Validate(func(value string) error { return settings.ValidateTestDir(strings.TrimSpace(value)) })
}

// harnessField asks which harnesses the project uses. Nothing is selected to begin with, and at
// least one answer is required: codefall cannot know which harnesses a project means to use, and a
// preselected option would be the same decision made on the user's behalf that a default flag value
// was.
func harnessField(harnesses *[]string) huh.Field {
	names := harness.All()

	options := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		options = append(options, huh.NewOption(harnessLabel(name), name))
	}

	return huh.NewMultiSelect[string]().
		Title("Which coding harnesses should codefall set up?").
		Description("Choose every harness this project uses. Its skills and hooks are installed for each.").
		Options(options...).
		Value(harnesses).
		Validate(atLeastOneHarness)
}

// harnessLabels is how each harness is written in the survey, as its makers write it. A harness with
// no entry here reads as its own name, which is wrong in its capitals rather than absent from the
// list.
var harnessLabels = map[string]string{
	harness.Agy:      "Antigravity",
	harness.Claude:   "Claude Code",
	harness.Codex:    "Codex",
	harness.Muse:     "Muse",
	harness.OpenCode: "OpenCode",
}

func harnessLabel(name string) string {
	if label, ok := harnessLabels[name]; ok {
		return label
	}

	return name
}

// atLeastOneHarness is what makes the question unskippable. Huh accepts an empty multi-select
// otherwise, and a run for no harness has nowhere to install.
func atLeastOneHarness(chosen []string) error {
	if len(chosen) == 0 {
		return errors.New("choose at least one harness")
	}

	return nil
}

func gitHubFields(request application.Request, repo, project *string) []huh.Field {
	var fields []huh.Field

	if request.IssuesRepo.IsAbsent() {
		fields = append(fields, repoField(repo))
	}

	if request.IssuesProject.IsAbsent() {
		fields = append(fields, projectField(project))
	}

	return fields
}

// answered folds the survey's strings back into the contract. Everything here has already been
// validated by the field that collected it.
func answered(
	request application.Request, tracker, repo, project string,
) (application.Request, error) {
	request.Tracker = tracker

	if tracker != settings.TrackerGitHub {
		return request, nil
	}

	request.IssuesRepo = mo.Some(strings.TrimSpace(repo))

	trimmed := strings.TrimSpace(project)
	if trimmed == "" {
		return request, nil
	}

	number, err := strconv.Atoi(trimmed)
	if err != nil {
		return application.Request{}, fmt.Errorf("project number %q: %w", project, err)
	}

	request.IssuesProject = mo.Some(number)

	return request, nil
}

func trackerField(tracker *string) huh.Field {
	return huh.NewSelect[string]().
		Title("Which issue tracker should codefall use?").
		Options(
			huh.NewOption("GitHub Issues", settings.TrackerGitHub),
			huh.NewOption("Beads", settings.TrackerBeads),
			huh.NewOption("Jira (not yet available)", trackerJira),
			huh.NewOption("Linear (not yet available)", trackerLinear),
		).
		Value(tracker).
		Validate(availableTracker)
}

// availableTracker is what makes the last two options unchoosable. Huh v2 has no disabled option, so
// the refusal happens on submit, and its message says what to choose instead.
//
// The name is not the first word of the message because an error that opens with a capital fails the
// linter, and "Jira" is a name rather than a sentence starting in the wrong case.
func availableTracker(tracker string) error {
	switch tracker {
	case trackerJira:
		return notAvailable("Jira")
	case trackerLinear:
		return notAvailable("Linear")
	default:
		_, err := settings.ParseTracker(tracker)

		return err
	}
}

func notAvailable(tracker string) error {
	return fmt.Errorf("codefall cannot use %s yet — choose GitHub Issues or Beads", tracker)
}

func repoField(repo *string) huh.Field {
	return huh.NewInput().
		Title("Which GitHub repository holds the issues?").
		Placeholder("owner/name").
		Value(repo).
		Validate(func(value string) error { return settings.ValidateRepo(strings.TrimSpace(value)) })
}

// reviewPostToPRField asks the one question the review block holds. No is the starting value:
// posting is visible to everyone on the pull request, so it is something a project turns on rather
// than something it discovers already on.
func reviewPostToPRField(postToPR *bool) huh.Field {
	return huh.NewConfirm().
		Title("Let codefall-review post its findings to a pull request?").
		Description("Findings are always written to .codefall/reviews/ either way.").
		Affirmative("Yes").
		Negative("No").
		Value(postToPR)
}

func projectField(project *string) huh.Field {
	return huh.NewInput().
		Title("Which GitHub Project number? Leave it blank for none.").
		Placeholder("3").
		Value(project).
		Validate(validateProject)
}

// validateProject accepts a blank answer, because a project is optional and blank is how a form says
// none.
func validateProject(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	number, err := strconv.Atoi(trimmed)
	if err != nil || number < 1 {
		return fmt.Errorf("a project number is a positive integer, not %q", value)
	}

	return nil
}
