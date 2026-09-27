package application

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// equipTestRemedy is what to do about a project whose test harness has not been set up: the verb
// that installs a runner and declares it (ADR-007). Doctor names it and never runs it, which is the
// rule for everything equip sets up.
const equipTestRemedy = "run /codefall-equip to set the project's test harness up"

// testing runs checks 16 to 18: the project says where its test cases live, a runner is declared to
// run them, and the directory it declared is there (ADR-007).
//
// It reads the same settings the settings group validated, so whatever skipped those checks skips
// these, and the checks are absent from the report rather than present with a status of their own.
func (d *Diagnose) testing(_ context.Context, dir string, results []domain.Result) []domain.Result {
	if !settingsAreComplete(results) {
		return results
	}

	doc, read := d.settingsDocument(dir)
	if !read {
		return results
	}

	declared, ok := settings.TestDeclaration(doc).Get()
	if !ok {
		// A warning, not a failure: a project with no cases is a project nothing is missing from yet.
		// What the block decides is where the cases go, and init is what asks — so the two checks
		// below have nothing to ask about until it has, and are absent from the report.
		return append(results, domain.TestDeclared.Warn(
			"no testing root is declared in settings.json", mo.Some(initRemedy)))
	}

	results = append(results, domain.TestDeclared.PassWithDetail(declared.Dir+"/"))

	if len(declared.Runners) == 0 {
		// Also a warning. A project declares its root before it is equipped, and until a runner is
		// installed there is nothing to run a spec with — which is equip's work and nobody else's.
		results = append(results, domain.TestEquipped.Warn(
			"no test runner is declared in settings.json", mo.Some(equipTestRemedy)))
	} else {
		results = append(results, domain.TestEquipped.PassWithDetail(strings.Join(declared.Runners, ", ")))
	}

	return d.testDirExists(dir, declared.Dir, results)
}

// testDirExists is check 18: the directory the settings declare is there.
//
// It fails rather than warns. The project declared it, and every verb that writes a case or runs one
// looks for it, so a declaration pointing at nothing is one nothing can act on. The remedy is init,
// which makes the tree from the root the settings already name.
func (d *Diagnose) testDirExists(dir, root string, results []domain.Result) []domain.Result {
	there, err := d.files.DirExists(filepath.Join(dir, filepath.FromSlash(root)))
	if err != nil {
		return append(results, domain.TestDirExists.Fail(
			fmt.Sprintf("Cannot stat %s: %v", root, err), mo.None[string]()))
	}

	if !there {
		return append(results, domain.TestDirExists.Fail(
			root+"/ is declared in settings.json and is not there", mo.Some(initRemedy)))
	}

	return append(results, domain.TestDirExists.Pass())
}
