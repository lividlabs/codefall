// Package doctor is the facade for `codefall doctor`, the command that reports whether a project has
// what it needs to run codefall end to end. It reports and never repairs; `codefall init` and
// `codefall upgrade` own the fixes.
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package doctor

import (
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/infrastructure"
	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/presentation"
)

// Register wires doctor's object graph into the app's injector. This is the only place the
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

	do.Provide(injector, func(i do.Injector) (presentation.DiagnoseUseCase, error) {
		return application.NewDiagnose(
			do.MustInvoke[application.FileSystem](i),
			do.MustInvoke[application.CommandRunner](i),
		), nil
	})
}

// Command returns doctor's command for the composition root to mount. A *cobra.Command is a delivery
// type, not a domain entity, so ADR-001 does not apply to it.
func Command(injector do.Injector) *cobra.Command {
	return presentation.NewDoctorCommand(do.MustInvoke[presentation.DiagnoseUseCase](injector))
}
