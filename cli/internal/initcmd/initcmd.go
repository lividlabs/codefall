// Package initcmd is the facade for `codefall init` and `codefall upgrade`: the command that makes a
// directory ready for codefall, once, and the command that brings that install level with the binary
// on every run after it. Both run one use case — write the .codefall/settings.json that doctor
// checks, install the codefall extension for the harnesses, initialise the tracker, register the
// hooks, write the marked sections, make the testing tree, record the run in the manifest — and
// differ only in how the request is built: init from a survey or flags, upgrade from what the
// settings and the manifest already record (ADR-010).
//
// The package is named initcmd rather than init because a package called init cannot be imported
// without an alias — `import ".../internal/init"` does not compile, since init must be a func.
// initcmd keeps the directory named after the first command it held.
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package initcmd

import (
	"context"
	"io"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	changelog "github.com/lividlabs/codefall"
	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/infrastructure"
	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/presentation"
	"github.com/lividlabs/codefall/extensions"
)

// Register wires initcmd's object graph into the app's injector. This is the only place the
// component's concrete types are named: main cannot import them (ADR-GO-01 as amended 2026-08-19).
//
// Every provider's static return type is the interface, never the concrete type — a provider that
// returns the concrete type compiles and then fails at runtime with "could not find service".
func Register(injector do.Injector) {
	do.Provide(injector, func(do.Injector) (application.FileSystem, error) {
		return infrastructure.NewOSFileSystem(), nil
	})

	do.Provide(injector, func(do.Injector) (application.CommandRunner, error) {
		return infrastructure.NewExecCommandRunner(), nil
	})

	do.Provide(injector, func(do.Injector) (application.ExtensionSource, error) {
		return infrastructure.NewEmbeddedExtensionFetcher(extensions.Files(), extensions.SkillRenames()), nil
	})

	do.Provide(injector, func(do.Injector) (application.ChangeLog, error) {
		return infrastructure.NewChangeLogSource(changelog.Text()), nil
	})

	do.Provide(injector, func(i do.Injector) (presentation.InitializeUseCase, error) {
		return application.NewInitialize(
			do.MustInvoke[application.FileSystem](i),
			do.MustInvoke[application.CommandRunner](i),
			do.MustInvoke[application.ExtensionSource](i),
			do.MustInvoke[application.ChangeLog](i),
		), nil
	})
}

// Commands returns both of initcmd's commands, init and upgrade, for the composition root to mount. A
// *cobra.Command is a delivery type, not a domain entity, so ADR-001 does not apply to it.
//
// prerequisites is the check init runs before its survey, which the composition root takes
// from doctor's facade. Upgrade does not run it: it repairs an install, and a machine without gh can
// still do that.
func Commands(
	injector do.Injector, prerequisites func(context.Context, io.Writer) error,
) []*cobra.Command {
	return []*cobra.Command{Command(injector, prerequisites), UpgradeCommand(injector)}
}

// Command returns the init command alone. create runs init in the directory it makes and needs an
// instance of its own, never the one mounted on the root. create has run the prerequisite check
// before it made the directory, so the instance built for it is given Checked.
func Command(injector do.Injector, prerequisites func(context.Context, io.Writer) error) *cobra.Command {
	return presentation.NewInitCommand(
		do.MustInvoke[presentation.InitializeUseCase](injector), prerequisites)
}

// Checked is the prerequisite check for an init command whose caller has already made it. It passes
// without asking any tool anything.
func Checked(context.Context, io.Writer) error { return nil }

// UpgradeCommand returns the upgrade command alone.
func UpgradeCommand(injector do.Injector) *cobra.Command {
	return presentation.NewUpgradeCommand(do.MustInvoke[presentation.InitializeUseCase](injector))
}
