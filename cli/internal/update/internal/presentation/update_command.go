// Package presentation builds update's command. It is thin: it asks the use case how the running
// binary was installed, and either reports that, says what to run instead, or runs the update.
package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/cli/internal/shared/ui"
	"github.com/lividlabs/codefall/cli/internal/shared/version"
	"github.com/lividlabs/codefall/cli/internal/update/internal/domain"
)

// errCancelled is what an interrupted download reports, returned as it is so Fang renders that
// sentence and not a prefix in front of it.
var errCancelled = errors.New("update cancelled")

// UpdateUseCase is what the command needs from the application layer, declared by its consumer.
type UpdateUseCase interface {
	Inspect() (domain.Installation, error)
	Latest(ctx context.Context) (version.Version, error)
	Run(ctx context.Context, installation domain.Installation, requested mo.Option[string]) (domain.Outcome, error)
}

// NewUpdateCommand builds `codefall update`.
func NewUpdateCommand(update UpdateUseCase) *cobra.Command {
	var check bool

	command := &cobra.Command{
		Use:   "update [version]",
		Short: "Update the codefall binary to the latest release",
		Long: "Finds out how the running codefall was installed before it changes anything. A binary the " +
			"install script installed is replaced with the latest release, or the version given, after " +
			"its checksum is verified. A binary mise or go install manages is left alone, and the " +
			"command that updates it is printed. To bring a project's install level with the binary, " +
			"run codefall upgrade after this.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := ui.NewWriter(cmd.OutOrStdout())

			installation, err := update.Inspect()
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}

			if check {
				return report(cmd.Context(), out, update, installation)
			}

			if installation.Method != domain.MethodNative {
				return refuse(out, installation)
			}

			requested := mo.None[string]()
			if len(args) == 1 {
				requested = mo.Some(args[0])
			}

			outcome, err := ui.RunWithSpinner(cmd.Context(), out, "Updating codefall…", errCancelled,
				func(ctx context.Context, _ ui.Progress) (domain.Outcome, error) {
					return update.Run(ctx, installation, requested)
				})

			return finish(out, outcome, err)
		},
	}

	command.Flags().BoolVar(&check, "check", false,
		"report how codefall was installed and the latest release, and change nothing")

	return command
}

// report is --check: the install method, the binary, its version, and the latest release.
func report(ctx context.Context, out io.Writer, update UpdateUseCase, installation domain.Installation) error {
	latest, err := update.Latest(ctx)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return writeLines(out,
		"Installed by  "+installation.Method.String(),
		"Path          "+installation.Path,
		"Version       "+installation.Build,
		"Latest        "+latest.String(),
	)
}

// refuse says how a binary update leaves alone was installed and what updates it, and returns the
// error that makes the run exit non-zero.
func refuse(out io.Writer, installation domain.Installation) error {
	switch installation.Method {
	case domain.MethodMise:
		if err := writeLines(out,
			fmt.Sprintf("This codefall was installed by mise: %s, version %s.", installation.Path, installation.Build),
			"To update it within the version your mise config allows:",
			"  mise upgrade "+domain.MiseTool,
			"To update it to the latest release and rewrite the version in your mise config:",
			"  mise upgrade --bump "+domain.MiseTool,
		); err != nil {
			return err
		}

		return errors.New("update leaves a mise install to mise")
	case domain.MethodSource:
		if err := writeLines(out,
			fmt.Sprintf("This codefall was built from source: %s, version %s.", installation.Path, installation.Build),
			"To update it:",
			"  "+domain.GoInstallCommand,
		); err != nil {
			return err
		}

		return errors.New("update leaves a source build to go install")
	default:
		if err := writeLines(out,
			"Update cannot tell how "+installation.Path+" was installed.",
			"To install the latest release with the install script, which update keeps current:",
			"  "+domain.InstallScriptCommand,
		); err != nil {
			return err
		}

		return errors.New("update replaces only a binary the install script installed")
	}
}

// finish reports a native update. A directory update cannot write to gets the install script's line,
// because update never asks for elevated privileges.
func finish(out io.Writer, outcome domain.Outcome, err error) error {
	if errors.Is(err, domain.ErrNotWritable) {
		if writeErr := writeLines(out,
			"To reinstall codefall with the install script:",
			"  "+domain.InstallScriptCommand,
		); writeErr != nil {
			return writeErr
		}
	}

	if outcome.Changed {
		if writeErr := writeLines(out, updatedLine(outcome)); writeErr != nil {
			return writeErr
		}
	}

	if err != nil {
		if errors.Is(err, errCancelled) {
			return err
		}

		return fmt.Errorf("update: %w", err)
	}

	if !outcome.Changed {
		return writeLines(out, fmt.Sprintf("codefall %s is already up to date: %s", outcome.To, outcome.Path))
	}

	return nil
}

func updatedLine(outcome domain.Outcome) string {
	if from, ok := outcome.From.Get(); ok {
		return fmt.Sprintf("Updated codefall from %s to %s: %s", from, outcome.To, outcome.Path)
	}

	return fmt.Sprintf("Updated codefall to %s: %s", outcome.To, outcome.Path)
}

func writeLines(out io.Writer, lines ...string) error {
	for _, line := range lines {
		if err := ui.WriteLine(out, line); err != nil {
			return err
		}
	}

	return nil
}
