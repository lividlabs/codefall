package presentation

import (
	"context"
	"errors"
	"os"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
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
			"rewrites a harness name the settings or the manifest still spell the old way, and records " +
			"the run in .codefall/manifest.json. It changes nothing it did not write. A project already " +
			"installed at this binary's version is reported as up to date and left alone. " +
			"--harness adds a harness the project did not choose at init. " +
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
	registerHarnessFlag(cmd, &f.harnesses, "coding harness to add to the ones the settings record")
	registerLocationFlag(cmd, &f.location)
	cmd.Flags().BoolVarP(&f.yes, "yes", "y", false,
		"upgrade to this binary's version without asking")
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

	// A bad flag or a project that was never set up is the user's own message; it is returned as it
	// is so Fang renders that sentence and not a prefix in front of it.
	request, err := buildUpgradeRequest(cmd, initialize, flags, dir)
	if err != nil {
		return err
	}

	// One colour-profile writer for the whole run.
	out := ui.NewWriter(cmd.OutOrStdout())

	if request.NoOp {
		return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(
			"already up to date with "+request.CLIVersion))
	}

	if _, err := runInitialize(cmd.Context(), upgradeKind, initialize, request, out); err != nil {
		return upgradeKind.wrap(err)
	}

	return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render(nextStep))
}

// buildUpgradeRequest builds the use case's contract from what the project already records. Nothing
// is surveyed: the answers were given at init, and the one question this command asks is whether to
// move the installed version.
func buildUpgradeRequest(
	cmd *cobra.Command, initialize InitializeUseCase, flags *upgradeFlags, dir string,
) (application.Request, error) {
	chosen, err := parseHarnesses(flags.harnesses)
	if err != nil {
		return application.Request{}, err
	}

	// Where the install is comes before everything else, because every other question — whether
	// there is a manifest, what version it records — is a question about that directory.
	dir, err = chooseLocation(cmd.Context(), upgradeKind, initialize, flags.location, dir)
	if err != nil {
		return application.Request{}, err
	}

	request := application.Request{Dir: dir, Harnesses: chosen, CLIVersion: buildinfo.Version()}

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
	// rather than asking again — and the gate below cannot compare what it has not been told.
	if len(request.Harnesses) == 0 {
		names, err := initialize.ChosenHarnesses(dir)
		if err != nil {
			return application.Request{}, upgradeKind.wrap(err)
		}

		chosen, ok := names.Get()
		if !ok {
			return application.Request{}, errors.New("the settings here record no harnesses; pass --harness")
		}

		request.Harnesses = chosen
	}

	// The testing root is read back the same way, and for the same reason: a project that has
	// declared one is not asked about it again, and this run never moves it (ADR-007). A project
	// settled before the block existed declares none, and the use case takes the format's default.
	declaredTest, err := initialize.DeclaredTestDir(dir)
	if err != nil {
		return application.Request{}, upgradeKind.wrap(err)
	}

	request.TestDir = declaredTest.OrEmpty()

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
	// doctor's remedy for it is this command too.
	if declaredTest.IsPresent() && len(formers) == 0 &&
		installedEverything(installation, request.Harnesses, request.CLIVersion) {
		request.NoOp = true

		return request, nil
	}

	// The question is about moving a version, so it is asked only when a version moves. A harness
	// with no record at all is work rather than an upgrade, and is not asked about.
	if versionMoves(installation, request.Harnesses, request.CLIVersion) && !flags.yes {
		if err := confirmUpgrade(cmd.Context(), request.CLIVersion); err != nil {
			return application.Request{}, err
		}
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

// confirmUpgrade is the one prompt this command asks before it files its own changes: upgrade to the
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
			return upgradeKind.cancelled
		}

		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return upgradeKind.cancelled
	default:
		return upgradeKind.wrap(err)
	}
}
