// Package presentation builds initcmd's two commands. It is thin: it collects answers — from flags,
// from a survey, from gh, or from what the settings and the manifest already record — hands them to
// the one use case as a contract, and prints what each step did. The palette, the writer, and the
// spinner are the shared UI module's; what belongs here is which tone a step's outcome is drawn in.
//
// This file holds what both commands share. init_command.go is `codefall init`, the first run, and
// upgrade_command.go is `codefall upgrade`, every run after it.
package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// InitializeUseCase is what both commands need from the application layer, declared by their
// consumer. The questions beside Run are what a command asks before it prompts or refuses: whether
// there is a manifest, which decides which command this project is for; whether prompting is worth
// doing at all; and what to offer as the answer to the one question a tool can answer itself.
type InitializeUseCase interface {
	Run(ctx context.Context, request application.Request, observer application.Observer) (domain.Report, error)
	// ManifestExists reports whether a finished run has recorded itself in .codefall/manifest.json.
	// init refuses a project that has one and upgrade refuses a project that has none, so the two
	// commands never do each other's work.
	ManifestExists(dir string) (bool, error)
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
	// settings or they record none. A run over settled settings installs for what the project
	// already chose rather than asking again.
	ChosenHarnesses(dir string) (mo.Option[[]string], error)
	// DeclaredTestDir reads the testing root .codefall/settings.json records, or None when there are
	// no settings or they declare none. A run over settled settings works with the root the project
	// already declared and never moves it (ADR-007).
	DeclaredTestDir(dir string) (mo.Option[string], error)
	// FormerHarnessNames reads the old harness spellings .codefall/settings.json and
	// .codefall/manifest.json still carry. An upgrade that finds one has a rewrite to make, so it is
	// never a no-op.
	FormerHarnessNames(dir string) ([]string, error)
	// BreakingChanges reads the breaking changes recorded between the version the manifest records
	// and binary, earliest first, or None when the range cannot be determined. Upgrade prints them
	// before it changes anything (ADR-010).
	BreakingChanges(dir, binary string) (mo.Option[[]application.BreakingRelease], error)
}

// commandKind is what a shared helper needs to know about the command running it: the word its
// errors open with, the sentence a cancelled run reports, and what the spinner says before the
// first step reports itself. Both commands run one use case, and this is the whole of what differs
// in how they talk about it.
type commandKind struct {
	name      string
	cancelled error
	opening   string
}

// The two commands. Each cancellation error is returned as it is rather than wrapped, so Fang
// renders that sentence and not a prefix in front of it: Huh says the user aborted the form and
// Bubble Tea says the program was interrupted, but Ctrl-C during a question and Ctrl-C during the
// spinner are one event to the person who pressed it.
var (
	initKind    = commandKind{name: "init", cancelled: errors.New("init cancelled"), opening: "Setting up…"}
	upgradeKind = commandKind{name: "upgrade", cancelled: errors.New("upgrade cancelled"), opening: "Upgrading…"}
)

// wrap prefixes a failure with the command's name, so the reader knows which command failed. A
// cancellation is returned as it is.
func (k commandKind) wrap(err error) error {
	if errors.Is(err, k.cancelled) {
		return err
	}

	return fmt.Errorf("%s: %w", k.name, err)
}

// Where a run from below the repository root installs. The root is where a project usually keeps
// its harness configuration; the directory the command runs in is for a team that wants codefall in
// its part of a larger repository without setting it up for everyone else's.
const (
	locationHere = "here"
	locationRoot = "root"
)

// nextStep is the line that closes a successful run of either command. Both write what doctor
// checks, so doctor is what to run next.
const nextStep = "Next: codefall doctor"

// registerLocationFlag adds --location, which both commands take: which directory holds the install
// is a question about the checkout, not about which run this is.
func registerLocationFlag(cmd *cobra.Command, location *string) {
	cmd.Flags().StringVar(location, "location", "",
		"where to install when run below the repository root ("+locationHere+" for this directory, "+
			locationRoot+" for the root)")
}

// registerHarnessFlag adds --harness. init needs it the first time, and upgrade takes it to add a
// harness; neither has a default, because codefall cannot know which harnesses a project means to
// use, and a default would choose one on the user's behalf.
func registerHarnessFlag(cmd *cobra.Command, harnesses *[]string, usage string) {
	cmd.Flags().StringSliceVar(harnesses, "harness", nil,
		usage+" — repeat the flag, or separate names with commas, for several ("+
			strings.Join(harness.All(), ", ")+")")
}

// chooseLocation settles which directory the run installs into. Only a run below the repository root
// has a choice to make; at the root, and outside a repository, the working directory is the answer
// and --location is not needed. Below the root, the flag answers, then a person, and a script with
// neither is told which flag it is missing (ADR-002).
func chooseLocation(
	ctx context.Context, kind commandKind, initialize InitializeUseCase, location, dir string,
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
		if err := runForm(ctx, kind, []*huh.Group{huh.NewGroup(locationField(&location, dir, root))}); err != nil {
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

// runForm runs a form. A form is a terminal program, so a cancelled command cancels it (ADR-002) and
// ACCESSIBLE chooses the screen-reader mode, as Huh's own examples do.
//
// There is always at least one group: a form is reached only when an answer is missing, and each
// answer that could be missing adds a group of its own.
func runForm(ctx context.Context, kind commandKind, groups []*huh.Group) error {
	err := huh.NewForm(groups...).
		WithAccessible(os.Getenv("ACCESSIBLE") != "").
		RunWithContext(ctx)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return kind.cancelled
	default:
		return kind.wrap(err)
	}
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
