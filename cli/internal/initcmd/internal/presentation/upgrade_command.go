package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall/cli/internal/shared/ui"
)

// NewUpgradeCommand builds `codefall upgrade`, every run after the first: it brings a project's
// install level with the binary, for the harnesses the settings record, and touches nothing it did
// not write.
func NewUpgradeCommand(initialize InitializeUseCase) *cobra.Command {
	flags := &upgradeFlags{}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Bring this project's codefall install up to date",
		Long: "Reinstalls the codefall extension, its shared files, and its hooks for the harnesses " +
			".codefall/settings.json records, replaces the sections codefall wrote into AGENTS.md, " +
			"rewrites a harness name the settings or the manifest still spell the old way, points a " +
			"$schema URL an earlier release wrote at the current one, installs the skills under the " +
			"prefix the settings' skillPrefix names and removes the ones under the prefix they had, and records " +
			"the run in .codefall/manifest.json, removing what the previous install wrote that this " +
			"one does not ship. Before it changes anything it prints the breaking changes recorded " +
			"between the installed version and this one and asks to continue. Every run reinstalls " +
			"the extension, so a project already installed at this binary's version gets back any " +
			"skill, shared file, ignore entry, AGENTS.md section, hook registration, or testing file " +
			"that has gone missing or been edited; upgrade reports what it restored, and reports the " +
			"install as up to date only when nothing changed. " +
			"--harness adds a harness the project did not choose at init and records it in the settings. " +
			"Upgrade needs a manifest: a project that has none is codefall init's. " +
			"Run below the root of a git repository, upgrade asks whether the install is there or at " +
			"the root; --location answers without asking.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpgrade(cmd, initialize, flags)
		},
	}

	flags.register(cmd)

	return cmd
}

// upgradeFlags is what a run can be told: a harness to add, where the install is, and that the
// version question is answered.
type upgradeFlags struct {
	harnesses []string
	location  string
	yes       bool
}

func (f *upgradeFlags) register(cmd *cobra.Command) {
	registerHarnessFlag(cmd, &f.harnesses,
		"coding harness to add to the ones the settings record, and install for alongside them")
	registerLocationFlag(cmd, &f.location)
	cmd.Flags().BoolVarP(&f.yes, "yes", "y", false,
		"upgrade to this binary's version without asking, breaking changes included")
}

// runUpgrade is the command's body: read what the project has, decide whether there is anything to
// do, run the steps, say what happened.
func runUpgrade(cmd *cobra.Command, initialize InitializeUseCase, flags *upgradeFlags) error {
	// The working directory is the one piece of environment the command reads; the use case
	// receives it as a value.
	dir, err := os.Getwd()
	if err != nil {
		return upgradeKind.wrap(err)
	}

	// One colour-profile writer for the whole run, opened before the request is built because the
	// breaking changes are printed while it is.
	out := ui.NewWriter(cmd.OutOrStdout())

	// A bad flag or a project that was never set up is the user's own message; it is returned as it
	// is so Fang renders that sentence and not a prefix in front of it.
	request, err := buildUpgradeRequest(cmd, initialize, flags, dir, out)
	if err != nil {
		return err
	}

	report, err := runInitialize(cmd.Context(), upgradeKind, initialize, request, out)
	if err != nil {
		return upgradeKind.wrap(err)
	}

	// A current install printed only the steps that changed something, the extension copy
	// included. When none did, the install is exactly as this version leaves it.
	if request.Current && !changedAnything(report) {
		return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(
			"already up to date with "+request.CLIVersion))
	}

	return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(nextStep))
}

// buildUpgradeRequest builds the use case's contract from what the project already records. Nothing
// is surveyed: the answers were given at init, and the one question this command asks is whether to
// move the installed version.
func buildUpgradeRequest(
	cmd *cobra.Command, initialize InitializeUseCase, flags *upgradeFlags, dir string, out io.Writer,
) (application.Request, error) {
	added, err := parseHarnesses(flags.harnesses)
	if err != nil {
		return application.Request{}, err
	}

	// Where the install is comes before everything else, because every other question — whether
	// there is a manifest, what version it records — is a question about that directory.
	dir, err = chooseLocation(cmd.Context(), upgradeKind, initialize, flags.location, dir)
	if err != nil {
		return application.Request{}, err
	}

	request := application.Request{Dir: dir, CLIVersion: buildinfo.Version()}

	// No manifest means no finished run to bring current. That project is init's, and init is what
	// writes the manifest this command needs.
	recorded, err := initialize.ManifestExists(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	if !recorded {
		return application.Request{}, errors.New(
			"codefall is not set up here; run codefall init first")
	}

	// The harnesses the project chose are recorded in its settings, so the run installs for those
	// rather than asking again, plus whatever --harness adds — and the gate below cannot compare
	// what it has not been told.
	names, err := initialize.ChosenHarnesses(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	chosenBySettings := names.OrEmpty()

	request.Harnesses = append(slices.Clone(chosenBySettings), added...)
	if len(request.Harnesses) == 0 {
		return application.Request{}, errors.New("the settings here record no harnesses; pass --harness")
	}

	// A harness the flag names that the settings do not is work whatever the versions say: the run
	// records it, and installs for it.
	unrecorded := slices.ContainsFunc(added, func(name string) bool {
		return !slices.Contains(chosenBySettings, name)
	})

	// The testing root is read back the same way, and for the same reason: a project that has
	// declared one is not asked about it again, and this run never moves it (ADR-007). A project
	// settled before the block existed declares none, and the use case takes the format's default.
	declaredTest, err := initialize.DeclaredTestDir(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	request.TestDir = declaredTest.OrEmpty()

	// The prefix the settings name the skills with, the default when they name none. The run
	// installs under it, and the gate below holds it against the prefix the manifest records
	// (ADR-015).
	request.SkillPrefix, err = initialize.DeclaredSkillPrefix(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	previous, err := initialize.Installed(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	formers, err := initialize.FormerHarnessNames(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	installation, ok := previous.Get()
	if !ok {
		// A manifest that records no usable version is a record with nothing to compare against,
		// so the run does its work rather than guessing that there is none.
		return request, nil
	}

	// A run for a harness the project has never been set up for is work to do, however current the
	// version that installed the others is — and so is a project that has never declared a testing
	// root, because the tree is what this run would make and doctor's remedy for an undeclared root
	// is this command. A harness still recorded under an old spelling is work for the same reason:
	// doctor's remedy for it is this command too. So is a settings prefix the manifest does not
	// record: the skills on disk are named one way and the project has asked for another, and this
	// run is what renames them (ADR-015).
	//
	// A current install still runs every step but Beads, and is not asked about a version that does
	// not move: the copy out of the binary and the repair steps put back what a person may have
	// removed since the last run, such as a skill, a shared script, an ignore line, or a section of
	// AGENTS.md, which doctor's remedies send them here to put back.
	if declaredTest.IsPresent() && len(formers) == 0 && !unrecorded &&
		installation.SkillPrefix == request.SkillPrefix &&
		installedEverything(installation, request.Harnesses, request.CLIVersion) {
		request.Current = true

		return request, nil
	}

	// What the person has to hear before anything moves: the breaking changes between the version
	// on record and this one. Printed here, before the question, so the answer is given knowing them.
	breaking, err := reportBreakingChanges(initialize, dir, request.CLIVersion, out)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	// The question is about moving a version, so it is asked only when a version moves. A harness
	// with no record at all is work rather than an upgrade, and is not asked about.
	if versionMoves(installation, request.Harnesses, request.CLIVersion) && !flags.yes {
		if err := confirmUpgrade(cmd.Context(), request.CLIVersion, breaking); err != nil {
			return application.Request{}, err
		}
	}

	return request, nil
}

// reportBreakingChanges prints the breaking changes recorded between the installed version and the
// binary's, grouped by release, and returns how many there were. A range that could not be
// determined is said once, in place of a list, so silence never reads as "none" (ADR-010).
func reportBreakingChanges(initialize InitializeUseCase, dir, binary string, out io.Writer) (int, error) {
	releases, err := initialize.BreakingChanges(dir, binary)
	if err != nil {
		return 0, err
	}

	between, ok := releases.Get()
	if !ok {
		return 0, ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(
			"the breaking changes between the installed version and "+binary+" could not be determined"))
	}

	count := 0

	for _, release := range between {
		if err := ui.WriteLine(out, ui.Style(ui.ToneWarn).Render("Breaking changes in "+release.Version)); err != nil {
			return 0, err
		}

		for _, note := range release.Notes {
			count++

			if err := ui.WriteLine(out, "  • "+note); err != nil {
				return 0, err
			}
		}
	}

	return count, nil
}

// installedEverything reports whether every harness this run is for is already installed at this
// binary's version, which is what lets a rerun skip Beads and the version question. A run for no
// harness has nothing installed rather than everything.
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

// confirmUpgrade is the one prompt this command asks before it files its own changes: upgrade to the
// binary's tag, with the breaking changes just printed in view. The user's "no" is not a cancellation
// of the run — it is a decline to move a version.
func confirmUpgrade(ctx context.Context, version string, breaking int) error {
	var goForIt bool

	title := "Upgrade the installed extension to " + version + "?"
	if breaking > 0 {
		title = fmt.Sprintf("%d breaking change%s listed above. Upgrade the installed extension to %s?",
			breaking, plural(breaking), version)
	}

	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(title).
			Value(&goForIt),
	)).WithAccessible(os.Getenv("ACCESSIBLE") != "").RunWithContext(ctx)

	switch {
	case err == nil:
		if !goForIt {
			return upgradeKind.cancelled
		}

		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return upgradeKind.cancelled
	default:
		return upgradeKind.wrap(err)
	}
}

// plural is the ending a count takes: "1 breaking change", "2 breaking changes".
func plural(count int) string {
	if count == 1 {
		return " is"
	}

	return "s are"
}
