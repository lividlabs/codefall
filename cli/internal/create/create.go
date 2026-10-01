// Package create is the facade for `codefall create`, the command that starts a project from nothing:
// it makes the directory, the git repository and its remote, and a first commit holding a README.md
// and a .gitignore, then runs `codefall init` there and offers to push.
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package create

import (
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/create/internal/application"
	"github.com/lividlabs/codefall/cli/internal/create/internal/infrastructure"
	"github.com/lividlabs/codefall/cli/internal/create/internal/presentation"
)

// Register wires create's object graph into the app's injector. This is the only place the
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

	do.Provide(injector, func(i do.Injector) (presentation.CreateUseCase, error) {
		return application.NewCreate(
			do.MustInvoke[application.FileSystem](i),
			do.MustInvoke[application.CommandRunner](i),
		), nil
	})
}

// Command returns create's command for the composition root to mount. initCommand is init's command,
// built by the composition root for create alone — create runs it and takes its flags, so it must not
// be the instance mounted on the root. Both are delivery types, not domain entities, so ADR-001 does
// not apply to them.
func Command(injector do.Injector, initCommand *cobra.Command) *cobra.Command {
	return presentation.NewCreateCommand(do.MustInvoke[presentation.CreateUseCase](injector), initCommand)
}
