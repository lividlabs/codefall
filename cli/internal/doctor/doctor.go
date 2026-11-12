// Package doctor is the facade for `codefall doctor`, the command that reports whether a project has
// what it needs to run codefall end to end. It reports and never repairs; `codefall init` and
// `codefall upgrade` own the fixes. It also exports the checks about the machine as a prerequisite
// check, which `codefall init` and `codefall create` run before their first step.
//
// Exported identifiers here are the component's whole public API. Its layers live under this
// package's own internal/, where the compiler keeps them (ADR-BASE-02).
package doctor

import (
	"context"
	"io"

	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/application"
	"github.com/lividlabs/codefall/cli/internal/doctor/internal/infrastructure"
	"github.com/lividlabs/codefall/cli/internal/doctor/internal/presentation"
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

// Prerequisites returns the check `codefall init` and `codefall create` run before their first step:
// doctor's checks on git, bd, and gh, and none of the checks that describe an install. The composition
// root passes it to those commands, which is how they reach it without importing this package.
//
// The check writes nothing and returns nil when every check passes. When one fails it writes each
// failing category to w as `codefall doctor` prints it and returns an error counting the failures. A
// run stopped with Ctrl-C returns context.Canceled. What crosses the facade is a function over
// standard types, so ADR-001 has no entity to keep in.
func Prerequisites(injector do.Injector) func(ctx context.Context, w io.Writer) error {
	return presentation.NewPrerequisiteCheck(do.MustInvoke[presentation.DiagnoseUseCase](injector))
}
