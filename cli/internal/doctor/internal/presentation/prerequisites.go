package presentation

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/lividlabs/codefall/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/ui"
)

// prerequisitesLabel is what the spinner says while git, bd, and gh are asked about themselves.
const prerequisitesLabel = "Checking prerequisites…"

// NewPrerequisiteCheck builds the check `codefall init` and `codefall create` run before their first
// step. It runs doctor's checks on git, bd, and gh. When every one passes it writes nothing and returns
// nil. When one fails it writes each failing category to w the way `codefall doctor` prints it, the
// remedy under the problem, and returns an error that counts the failures, so the person clears all
// of them in one pass. A warning is not a failure and is not printed: doctor is where it is read.
//
// A run stopped with Ctrl-C returns context.Canceled, which is the caller's to put in its own words.
//
// w is the caller's colour-profile writer, the one the rest of its output goes through.
func NewPrerequisiteCheck(diagnose DiagnoseUseCase) func(ctx context.Context, w io.Writer) error {
	return func(ctx context.Context, w io.Writer) error {
		// The three tools answer the same way from any directory; the working directory is one that
		// exists.
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		report, err := ui.RunWithSpinner(ctx, w, prerequisitesLabel, context.Canceled,
			func(ctx context.Context, _ ui.Progress) (domain.Report, error) {
				return diagnose.Prerequisites(ctx, dir)
			})
		if err != nil {
			return err
		}

		failed := report.Failed()
		if failed == 0 {
			return nil
		}

		for _, section := range report.Sections() {
			if section.Status() != domain.StatusFail {
				continue
			}

			if err := ui.WriteLine(w, sectionTree(section)); err != nil {
				return err
			}
		}

		return fmt.Errorf("%d prerequisite %s failed", failed, plural(failed, "check", "checks"))
	}
}
