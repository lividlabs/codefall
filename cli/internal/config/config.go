// Package config is the facade for `codefall config`, the command that shows the effective
// configuration and changes the agents list and its orders in .codefall/settings.json and the persona in
// .codefall/user.json one command at a time, without prompting, so a script can run it.
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package config

import (
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/config/internal/application"
	"github.com/lividlabs/codefall/cli/internal/config/internal/infrastructure"
	"github.com/lividlabs/codefall/cli/internal/config/internal/presentation"
)

// Register wires config's object graph into the app's injector. This is the only place the
// component's concrete types are named: main cannot import them (ADR-GO-01 as amended 2026-08-19).
//
// Every provider's static return type is the interface, never the concrete type — a provider that
// returns the concrete type compiles and then fails at runtime with "could not find service".
func Register(injector do.Injector) {
	do.Provide(injector, func(do.Injector) (application.FileSystem, error) {
		return infrastructure.NewOSFileSystem(), nil
	})

	do.Provide(injector, func(i do.Injector) (presentation.ConfigUseCase, error) {
		return application.NewConfig(do.MustInvoke[application.FileSystem](i)), nil
	})
}

// Command returns config's command, with its subcommands, for the composition root to mount. A
// *cobra.Command is a delivery type, not a domain entity, so ADR-001 does not apply to it.
func Command(injector do.Injector) *cobra.Command {
	return presentation.NewConfigCommand(do.MustInvoke[presentation.ConfigUseCase](injector))
}
