// Package application holds doctor's one use case. It owns the gateway interfaces its checks need
// and depends on nothing but the standard library, mo, doctor's own domain, and the pure shared
// modules (ADR-003).
package application

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
)

// FileSystem is the doctor's view of the working directory (one gateway role).
type FileSystem interface {
	// DirExists reports whether path exists and is a directory.
	DirExists(path string) (bool, error)
	// Exists reports whether anything is at path — a file, a directory, or a link.
	Exists(path string) (bool, error)
	// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist).
	ReadFile(path string) ([]byte, error)
}

// CommandRunner locates and runs the external tools codefall depends on (one gateway role).
type CommandRunner interface {
	// LookPath returns where a tool lives, or None when it is not on PATH. There is no useful "why"
	// behind a missing binary, so this is an Option and not an error (ADR-GO-03).
	LookPath(name string) mo.Option[string]
	// Run executes a tool and waits for it. A non-zero exit is a normal result; the error is
	// reserved for a command that could not be started at all.
	Run(ctx context.Context, dir, name string, args ...string) (CommandResult, error)
}

// CommandResult is what one external command produced.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Diagnose inspects a working directory and reports what codefall needs and what is missing. It
// reports and never repairs — every unmet check carries the remedy as text, and `codefall init`
// will be the thing that runs them.
type Diagnose struct {
	files  FileSystem
	runner CommandRunner
}

// NewDiagnose builds the use case over its two gateways.
func NewDiagnose(files FileSystem, runner CommandRunner) *Diagnose {
	return &Diagnose{files: files, runner: runner}
}

// Run executes every check that is worth running and returns the report. A check whose prerequisite
// failed is skipped and absent from the report. The error return is for a cancelled context only:
// an unreachable tool or an unreadable file is a finding, not a failure to diagnose.
func (d *Diagnose) Run(ctx context.Context, dir string) (domain.Report, error) {
	var results []domain.Result

	groups := []func(context.Context, string, []domain.Result) []domain.Result{
		d.settings,
		d.harnesses,
		d.agents,
		d.local,
		d.testing,
		d.beads,
		d.github,
	}

	for _, group := range groups {
		if err := ctx.Err(); err != nil {
			return domain.Report{}, fmt.Errorf("diagnose: %w", err)
		}

		results = group(ctx, dir, results)
	}

	return domain.NewReport(results...), nil
}
