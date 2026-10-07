// Package update is the facade for `codefall update`, the command that brings the codefall binary
// level with the latest release. It finds out how the running binary was installed first, and
// replaces it only when the install script put it there (ADR-015).
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package update

import (
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/update/internal/application"
	"github.com/lividlabs/codefall/cli/internal/update/internal/infrastructure"
	"github.com/lividlabs/codefall/cli/internal/update/internal/presentation"
)

// Register wires update's object graph into the app's injector. This is the only place the
// component's concrete types are named.
//
// Every provider's static return type is the interface, never the concrete type — a provider that
// returns the concrete type compiles and then fails at runtime with "could not find service".
func Register(injector do.Injector) {
	do.Provide(injector, func(do.Injector) (application.Host, error) {
		return infrastructure.NewOSHost(), nil
	})

	do.Provide(injector, func(do.Injector) (application.FileSystem, error) {
		return infrastructure.NewOSFileSystem(), nil
	})

	do.Provide(injector, func(do.Injector) (application.Releases, error) {
		return infrastructure.NewHTTPReleases(infrastructure.ReleasesURL), nil
	})

	do.Provide(injector, func(i do.Injector) (presentation.UpdateUseCase, error) {
		return application.NewUpdate(
			do.MustInvoke[application.Host](i),
			do.MustInvoke[application.FileSystem](i),
			do.MustInvoke[application.Releases](i),
		), nil
	})
}

// Command returns update's command for the composition root to mount.
func Command(injector do.Injector) *cobra.Command {
	return presentation.NewUpdateCommand(do.MustInvoke[presentation.UpdateUseCase](injector))
}
