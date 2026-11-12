package application

import (
	"context"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/domain"
)

// git runs check 22: git is on PATH. It asks nothing about the directory. Whether this is a work
// tree is init's question, asked where a run is about to write into it, and create runs before
// there is a directory at all.
func (d *Diagnose) git(ctx context.Context, dir string, results []domain.Result) []domain.Result {
	path, installed := d.runner.LookPath("git").Get()
	if !installed {
		return append(results, domain.GitInstalled.Fail("git is not on PATH", mo.Some("brew install git")))
	}

	return append(results, domain.GitInstalled.PassWithDetail(d.toolVersion(ctx, dir, path, "git", "--version")))
}
