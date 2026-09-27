package presentation

import (
	"context"
	"io"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// runInitialize runs the use case under the shared spinner, which is what turns a run's progress
// into something to look at: the running step's title while there is a terminal to draw on, and a
// plain line per finished step everywhere else. The kind supplies the label the spinner opens with
// and the error a cancelled run reports.
func runInitialize(
	ctx context.Context, kind commandKind, initialize InitializeUseCase, request application.Request, out io.Writer,
) (domain.Report, error) {
	return ui.RunWithSpinner(ctx, out, kind.opening, kind.cancelled,
		func(ctx context.Context, progress ui.Progress) (domain.Report, error) {
			return initialize.Run(ctx, request, progressObserver{progress: progress})
		})
}

// progressObserver is the whole of what a run has to say while it works, in the shared module's
// terms: a step that starts renames the label, and a step that finishes is a line.
type progressObserver struct {
	progress ui.Progress
}

func (o progressObserver) StepStarted(step domain.Step) {
	o.progress.Label(step.Title + "…")
}

func (o progressObserver) StepFinished(result domain.StepResult) {
	o.progress.Line(stepLine(result))
}
