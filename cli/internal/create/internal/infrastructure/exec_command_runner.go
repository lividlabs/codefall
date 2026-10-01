package infrastructure

import (
	"context"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/create/internal/application"
	"github.com/lividlabs/codefall/cli/internal/shared/process"
)

// ExecCommandRunner runs external tools as child processes, through the shared process module, and
// translates the result into create's own type.
type ExecCommandRunner struct{}

// NewExecCommandRunner builds the real command gateway.
func NewExecCommandRunner() *ExecCommandRunner {
	return &ExecCommandRunner{}
}

// LookPath resolves a tool on PATH.
func (*ExecCommandRunner) LookPath(name string) mo.Option[string] {
	return process.LookPath(name)
}

// Run executes a tool in dir and captures both streams. A non-zero exit is a normal result; the
// error is reserved for a command that could not be started at all.
func (*ExecCommandRunner) Run(
	ctx context.Context, dir, name string, args ...string,
) (application.CommandResult, error) {
	result, err := process.Run(ctx, dir, name, args...)
	if err != nil {
		return application.CommandResult{}, err
	}

	return application.CommandResult{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}, nil
}
