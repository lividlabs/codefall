// Package presentation builds initcmd's command. It is thin: it collects answers — from flags, from a
// survey, or from gh — hands them to the use case as a contract, and prints what each step did. The
// palette, the writer, and the spinner are the shared UI module's; what belongs here is which tone a
// step's outcome is drawn in.
package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// InitializeUseCase is what the command needs from the application layer, declared by its consumer.
// The two questions beside Run are what the command asks before it prompts: whether prompting is
// worth doing at all, and what to offer as the answer to the one question a tool can answer itself.
type InitializeUseCase interface {
	Run(ctx context.Context, request application.Request, observer application.Observer) (domain.Report, error)
	SettingsExist(dir string) (bool, error)
	SuggestIssuesRepo(ctx context.Context, dir string) mo.Option[string]
	// RepositoryRoot reports the root of the git work tree dir sits below, or None when dir is the
	// root or not in a work tree. It is what decides whether there is a location to ask about.
	RepositoryRoot(ctx context.Context, dir string) mo.Option[string]
	// Installed reads what finished runs recorded in .codefall/manifest.json: the version each
	// harness was installed at, or None when there is no manifest or it records no usable version.
	// The gate compares it one harness at a time against the binary's own version.
	Installed(dir string) (mo.Option[application.Installation], error)
	// ChosenHarnesses reads the harnesses .codefall/settings.json records, or None when there are no
	// settings or they record none. A rerun installs for what the project already chose rather than
	// asking again.
	ChosenHarnesses(dir string) (mo.Option[[]string], error)
	// DeclaredTestDir reads the testing root .codefall/settings.json records, or None when there are
	// no settings or they declare none. A rerun works with the root the project already declared and
	// never moves it (ADR-007).
	DeclaredTestDir(dir string) (mo.Option[string], error)
	// FormerHarnessNames reads the old harness spellings .codefall/settings.json and
	// .codefall/manifest.json still carry. A rerun that finds one has a rewrite to make, so it is
	// never a no-op.
	FormerHarnessNames(dir string) ([]string, error)
}

// The trackers the survey shows but does not accept. Huh has no disabled option, so they are offered
// with the state in their label and refused by the field's own validation — which is more use to a
// reader than leaving them out, because "not yet" is the answer they are looking for.
const (
	trackerJira   = "jira"
	trackerLinear = "linear"
)

// Where a run from below the repository root installs. The root is where a project usually keeps
// its harness configuration; the directory the command runs in is for a team that wants codefall in
// its part of a larger repository without setting it up for everyone else's.
const (
	locationHere = "here"
	locationRoot = "root"
)

// nextStep is the line that closes a successful run. Init writes what doctor checks, so doctor is
// what to run next.
const nextStep = "Next: codefall doctor"

// errCancelled is what both places a run can be interrupted report. Huh says the user aborted the
// form and Bubble Tea says the program was interrupted, but Ctrl-C during a question and Ctrl-C
// during the spinner are one event to the person who pressed it. It is returned as it is rather
// than wrapped, so Fang renders that sentence and not a prefix in front of it.
var errCancelled = errors.New("init cancelled")

// NewInitCommand builds `codefall init`.
func NewInitCommand(initialize InitializeUseCase) *cobra.Command {
	flags := &initFlags{}

	cmd := &cobra.Command{
		Use:     "init",
		Aliases: []string{"upgrade"},
		Short:   "Set this directory up for codefall",
		Long: "Creates .codefall/settings.json from your answers, including which coding harnesses the " +
			"project uses. Every question is also a flag, so a scripted run passes them and is never " +
			"prompted. The codefall extension is installed for each harness chosen: Claude Code gets it " +
			"at project scope into .claude/settings.json, and a harness that reads the .agents/skills " +
			"convention gets the extension's tree under .agents/. It also asks where the project's " +
			"test cases live and creates that tree. A rerun installs for the harnesses the settings " +
			"already record and keeps the testing root they declare, so --harness and --test-dir are " +
			"only needed the first time, or to add a harness. " +
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

// initFlags is every value the survey asks for, plus the two that change what a run does. Each
// prompted value is a flag, which is what keeps the command usable from a script (ADR-002).
type initFlags struct {
	tracker        string
	issuesRepo     string
	issuesProject  int
	reviewPostToPR bool
	harnesses      []string
	testDir        string
	location       string
	force          bool
	yes            bool
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
	// No default: codefall cannot know which harnesses a project means to use, and a default would
	// choose one on the user's behalf. A rerun takes them from the settings instead.
	cmd.Flags().StringSliceVar(&f.harnesses, "harness", nil,
		"coding harness to set up — repeat the flag, or separate names with commas, for several ("+
			strings.Join(harness.All(), ", ")+")")
	// No default: a project that has not declared a testing root is asked, and a rerun reads back the
	// one it declared. A default here would answer for the project on a run that could still ask.
	cmd.Flags().StringVar(&f.testDir, "test-dir", "",
		"directory the project's test cases live in, relative to this one (default "+
			settings.DefaultTestDir+"; only asked the first time)")
	cmd.Flags().StringVar(&f.location, "location", "",
		"where to install when run below the repository root ("+locationHere+" for this directory, "+
			locationRoot+" for the root)")
	cmd.Flags().BoolVar(&f.force, "force", false,
		"rewrite .codefall/settings.json if it is already there")
	cmd.Flags().BoolVarP(&f.yes, "yes", "y", false,
		"answer yes to the upgrade check without prompting")
}

// runInit is the command's body: collect the answers, run the steps, say what happened.
func runInit(cmd *cobra.Command, initialize InitializeUseCase, flags *initFlags) error {
	// The working directory is the one piece of environment the command reads; the use case
	// receives it as a value.
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}

	// A bad flag or an unanswerable question is the user's own message; it is returned as it is so
	// Fang renders that sentence and not a prefix in front of it.
	request, err := buildRequest(cmd, initialize, flags, dir)
	if err != nil {
		return err
	}

	// One colour-profile writer for the whole run.
	out := ui.NewWriter(cmd.OutOrStdout())

	if request.NoOp {
		return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(
			"already up to date with "+request.CLIVersion))
	}

	if _, err := runInitialize(cmd.Context(), initialize, request, out); err != nil {
		if errors.Is(err, errCancelled) {
			return err
		}

		return fmt.Errorf("init: %w", err)
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
	// settings exist, what version is installed — is a question about that directory.
	dir, err = chooseLocation(cmd.Context(), initialize, flags.location, dir)
	if err != nil {
		return application.Request{}, err
	}

	request := application.Request{Dir: dir, Harnesses: chosen, Force: flags.force,
		CLIVersion: buildinfo.Version()}

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

	// Settings that are already there and are not being rewritten are not worth surveying for: the
	// extension-copy is the same answer on a no-Fiorc run. Where a run is a no-op only because the
	// version matches, that's what the comparison reports.
	settled, err := initialize.SettingsExist(dir)
	if err != nil {
		return application.Request{}, fmt.Errorf("init: %w", err)
	}

	// The harnesses a settled project chose are recorded in its settings, so a rerun installs for
	// those rather than asking again — and the gate below cannot compare what it has not been told.
	if settled && len(request.Harnesses) == 0 {
		recorded, err := initialize.ChosenHarnesses(dir)
		if err != nil {
			return application.Request{}, fmt.Errorf("init: %w", err)
		}

		names, ok := recorded.Get()
		if !ok && !request.Force {
			return application.Request{}, errors.New("the settings here record no harnesses; " +
				"pass --harness, or --force to answer the questions again")
		}

		request.Harnesses = names
	}

	// The testing root is read back the same way, and for the same reason: a project that has
	// declared one is not asked about it again, and this run never moves it (ADR-007).
	declaredTest := mo.None[string]()

	if settled {
		declaredTest, err = initialize.DeclaredTestDir(dir)
		if err != nil {
			return application.Request{}, fmt.Errorf("init: %w", err)
		}

		if request.TestDir == "" {
			request.TestDir = declaredTest.OrEmpty()
		}
	}

	if settled && !request.Force {
		previous, err := initialize.Installed(dir)
		if err != nil {
			return application.Request{}, fmt.Errorf("init: %w", err)
		}

		formers, err := initialize.FormerHarnessNames(dir)
		if err != nil {
			return application.Request{}, fmt.Errorf("init: %w", err)
		}

		if recorded, ok := previous.Get(); ok {
			// A run for a harness the project has never been set up for is work to do, however
			// current the version that installed the others is — and so is a project that has never
			// declared a testing root, because the tree is what this run would make and doctor's
			// remedy for an undeclared root is this command. A harness still recorded under an old
			// spelling is work for the same reason: doctor's remedy for it is this command too.
			if declaredTest.IsPresent() && len(formers) == 0 &&
				installedEverything(recorded, request.Harnesses, request.CLIVersion) {
				request.NoOp = true
				return request, nil
			}

			// The question is about moving a version, so it is asked only when a version moves. A
			// harness with no record at all is work rather than an upgrade, and is not asked about.
			if versionMoves(recorded, request.Harnesses, request.CLIVersion) && !flags.yes {
				if err := confirmUpgrade(cmd.Context(), request.CLIVersion); err != nil {
					return application.Request{}, err
				}
			}
		}
	}

	if !settled || request.Force {
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

// installedEverything reports whether every harness this run is for is already installed at this
// binary's version, which is what makes a rerun a no-op. A run for no harness has nothing installed
// rather than everything.
func installedEverything(
	recorded application.Installation, harnesses []string, version string,
) bool {
	for _, name := range harnesses {
		if recorded.Versions[name] != version {
			return false
		}
	}

	return len(harnesses) > 0
}

// versionMoves reports whether any harness this run is for is installed at a different version,
// which is the one thing the upgrade confirmation is about.
func versionMoves(recorded application.Installation, harnesses []string, version string) bool {
	for _, name := range harnesses {
		if installed, recorded := recorded.Versions[name]; recorded && installed != version {
			return true
		}
	}

	return false
}

// chooseLocation settles which directory the run installs into. Only a run below the repository root
// has a choice to make; at the root, and outside a repository, the working directory is the answer
// and --location is not needed. Below the root, the flag answers, then a person, and a script with
// neither is told which flag it is missing (ADR-002).
func chooseLocation(
	ctx context.Context, initialize InitializeUseCase, location, dir string,
) (string, error) {
	if location != "" && location != locationHere && location != locationRoot {
		return "", fmt.Errorf("the --location flag must be %s or %s, not %q",
			locationHere, locationRoot, location)
	}

	root, below := initialize.RepositoryRoot(ctx, dir).Get()
	if !below {
		return dir, nil
	}

	if location == "" {
		if !stdinIsTerminal() {
			return "", fmt.Errorf("missing --location (stdin is not a terminal; %s is below the "+
				"repository root at %s)", dir, root)
		}

		location = locationHere
		if err := runForm(ctx, []*huh.Group{huh.NewGroup(locationField(&location, dir, root))}); err != nil {
			return "", err
		}
	}

	if location == locationRoot {
		return root, nil
	}

	return dir, nil
}

// locationField is the first question a run below the root asks. The working directory is the first
// option and the starting value: someone who ran init there most likely meant it.
func locationField(location *string, dir, root string) huh.Field {
	return huh.NewSelect[string]().
		Title("Install codefall here or at the repository root?").
		Options(
			huh.NewOption("Here: "+dir, locationHere),
			huh.NewOption("Repository root: "+root, locationRoot),
		).
		Value(location)
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

// parseHarnesses validates every name the flag was given and refuses the first one codefall cannot
// set up, in the order they were given so the message names the one the person typed.
func parseHarnesses(names []string) ([]string, error) {
	chosen := make([]string, 0, len(names))

	for _, name := range names {
		parsed, err := harness.Parse(strings.TrimSpace(name))
		if err != nil {
			return nil, err
		}

		chosen = append(chosen, parsed)
	}

	return chosen, nil
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

	if err := runForm(ctx, groups); err != nil {
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

// runForm runs the survey. A form is a terminal program, so a cancelled command cancels it
// (ADR-002) and ACCESSIBLE chooses the screen-reader mode, as Huh's own examples do.
//
// There is always at least one group: survey is reached only when an answer is missing, and each
// answer that could be missing adds a group of its own.
func runForm(ctx context.Context, groups []*huh.Group) error {
	err := huh.NewForm(groups...).
		WithAccessible(os.Getenv("ACCESSIBLE") != "").
		RunWithContext(ctx)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return errCancelled
	default:
		return fmt.Errorf("init: %w", err)
	}
}

// confirmUpgrade is the one prompt a re-run asks before it files its own changes: upgrade to the
// binary's tag. The user's "no" is not a cancellation of the run — it is a decline to move a version.
func confirmUpgrade(ctx context.Context, version string) error {
	var goForIt bool

	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Upgrade the installed extension to " + version + "?").
			Value(&goForIt),
	)).WithAccessible(os.Getenv("ACCESSIBLE") != "").RunWithContext(ctx)

	switch {
	case err == nil:
		if !goForIt {
			return errCancelled
		}

		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return errCancelled
	default:
		return fmt.Errorf("init: %w", err)
	}
}

// runForm runs the after-upgrade survey, if one is needed.

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

// stdinIsTerminal reports whether there is someone to answer a question. It is a variable so the
// tests can take the path a script takes; nothing else reassigns it.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(os.Stdin.Fd())
}

// The mark each outcome prints. A skipped step is a dash rather than the shared warning glyph:
// nothing is wrong, the work was simply already done.
var outcomeMarks = map[domain.Outcome]string{
	domain.OutcomeDone:    "✓",
	domain.OutcomeSkipped: "-",
}

// stepLine is one finished step: its mark, then the sentence the step wrote about itself.
func stepLine(result domain.StepResult) string {
	return outcomeStyle(result.Outcome).Render(glyph(result.Outcome)) + " " + result.Detail
}

func glyph(outcome domain.Outcome) string {
	if mark, ok := outcomeMarks[outcome]; ok {
		return mark
	}

	return "?"
}

// outcomeStyle is the one place an outcome becomes a style. An unknown outcome is left unstyled.
func outcomeStyle(outcome domain.Outcome) lipgloss.Style {
	return ui.Style(tone(outcome))
}

// tone is initcmd's whole share of the palette: the same teal doctor gives a passing check for a
// step that was done, and the same amber it gives a warning for one that was not, so the two
// commands read as one tool. Only the mark is coloured — the rest of a line is a path or a reason,
// where colour would be decoration rather than information.
func tone(outcome domain.Outcome) ui.Tone {
	switch outcome {
	case domain.OutcomeDone:
		return ui.TonePrimary
	case domain.OutcomeSkipped:
		return ui.ToneWarn
	default:
		return ui.ToneNone
	}
}
