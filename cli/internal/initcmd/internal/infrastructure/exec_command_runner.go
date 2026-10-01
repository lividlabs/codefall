package infrastructure

import (
	"context"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall/cli/internal/shared/process"
)

// ExecCommandRunner runs external tools as child processes, through the shared process module. What
// it adds is initcmd's own terms: the use case declares the gateway and the result type it wants
// back, so the translation happens here rather than in the shared module.
type ExecCommandRunner struct{}

// NewExecCommandRunner builds the real command gateway.
func NewExecCommandRunner() *ExecCommandRunner {
	return &ExecCommandRunner{}
}

// LookPath resolves a tool on PATH. Any failure means the same thing to the caller — the tool is not
// available — so the reason is dropped.
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
