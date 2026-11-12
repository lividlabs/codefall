package application

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/text"
)

// beads runs checks 23 and 24. A missing bd is a failure; a directory bd does not yet know about is
// only a warning, because `codefall init` will own initialising it.
//
// There is deliberately no test for a literal .beads/ directory: BEADS_DIR relocates it, so `bd
// info` run in the working directory is the only reliable question to ask.
func (d *Diagnose) beads(ctx context.Context, dir string, results []domain.Result) []domain.Result {
	results, installed := d.beadsInstalled(ctx, dir, results)
	if !installed {
		return results
	}

	initRemedy := mo.Some("bd init")

	result, err := d.runner.Run(ctx, dir, "bd", "info")

	switch {
	case err != nil:
		return append(results, domain.BeadsInitialized.Warn(
			fmt.Sprintf("Could not run bd info: %v", err), initRemedy))
	case result.ExitCode != 0:
		return append(results, domain.BeadsInitialized.Warn(
			fmt.Sprintf("This repository has no Beads database (bd info exited %d: %s)",
				result.ExitCode, text.FirstLine(result.Stderr)), initRemedy))
	}

	return append(results, domain.BeadsInitialized.Pass())
}

// beadsInstalled runs check 23 alone, and reports whether bd is there to be asked anything further.
// It is the half of the Beads checks that is about the machine: Prerequisites runs it without check
// 24, because a directory with no Beads database is what `codefall init` is run to fix.
func (d *Diagnose) beadsInstalled(
	ctx context.Context, dir string, results []domain.Result,
) ([]domain.Result, bool) {
	path, installed := d.runner.LookPath("bd").Get()
	if !installed {
		return append(results,
			domain.BeadsInstalled.Fail("bd is not on PATH", mo.Some("brew install beads"))), false
	}

	version := d.toolVersion(ctx, dir, path, "bd", "version")

	return append(results, domain.BeadsInstalled.PassWithDetail(version)), true
}

// toolVersion is the PASS detail for an installed tool: what it says about itself, or where it lives
// when it will not say.
func (d *Diagnose) toolVersion(ctx context.Context, dir, path, name string, args ...string) string {
	result, err := d.runner.Run(ctx, dir, name, args...)
	if err != nil || result.ExitCode != 0 {
		return path
	}

	if line := text.FirstLine(result.Stdout); line != "" {
		return line
	}

	return path
}
