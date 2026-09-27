// Package presentation builds create's command. It is thin: it collects the description and the
// remote — from flags or from a form — hands them to the use case, runs init in the new directory,
// and offers the push last.
package presentation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
	"github.com/samber/mo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/lividlabs/codefall-cli/cli/internal/create/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/create/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// CreateUseCase is what the command needs from the application layer, declared by its consumer.
type CreateUseCase interface {
	Scaffold(ctx context.Context, request application.Request, observer application.Observer) (domain.Report, error)
	Push(ctx context.Context, dir string) (domain.StepResult, error)
}

// errCancelled is what every place a run can be interrupted reports, returned as it is so Fang
// renders that sentence and not a prefix in front of it.
var errCancelled = errors.New("create cancelled")

// skippedInitFlags are init's flags that mean nothing in a directory create has just made: it is the
// repository root, so there is no location to choose.
var skippedInitFlags = []string{"location"}

// NewCreateCommand builds `codefall create`. initCommand is init's own command, built by the
// composition root; create runs it in the new directory and takes its flags as its own, so a
// scripted run answers init's questions on the same command line.
func NewCreateCommand(create CreateUseCase, initCommand *cobra.Command) *cobra.Command {
	flags := &createFlags{}

	cmd := &cobra.Command{
		Use:   "create <directory>",
		Short: "Create a new project and set it up for codefall",
		Long: "Creates the directory, initializes a git repository, adds the remote when there is one, " +
			"and commits a README.md and a stack-agnostic .gitignore. Then it runs codefall init in the " +
			"new directory, and offers to push to the remote. Every question is also a flag, including " +
			"init's, so a scripted run passes them and is never prompted.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, create, initCommand, flags, args[0])
		},
	}

	flags.register(cmd)
	adoptFlags(cmd, initCommand)

	return cmd
}

// createFlags is every value create asks for. Each is a flag, which is what keeps the command usable
// from a script (ADR-002).
type createFlags struct {
	description string
	remote      string
	push        bool
}

func (f *createFlags) register(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.description, "description", "",
		"one sentence about the project, written under the title in README.md (optional)")
	cmd.Flags().StringVar(&f.remote, "remote", "",
		"URL of the git remote to add as "+domain.RemoteName+" (optional)")
	cmd.Flags().BoolVar(&f.push, "push", false,
		"push to the remote once init has finished (needs a remote)")
}

// adoptFlags adds init's flags to create. The flag values are shared rather than copied: init reads
// its answers, and whether each flag was set, from the same *pflag.Flag create's parser filled in.
func adoptFlags(cmd, initCommand *cobra.Command) {
	initCommand.Flags().VisitAll(func(flag *pflag.Flag) {
		if slices.Contains(skippedInitFlags, flag.Name) || cmd.Flags().Lookup(flag.Name) != nil {
			return
		}

		cmd.Flags().AddFlag(flag)
	})
}

// runCreate is the command's body: collect the answers, scaffold, run init, offer the push.
func runCreate(
	cmd *cobra.Command, create CreateUseCase, initCommand *cobra.Command, flags *createFlags, target string,
) error {
	dir, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	request, err := buildRequest(cmd, flags, dir)
	if err != nil {
		return err
	}

	out := ui.NewWriter(cmd.OutOrStdout())

	if _, err := ui.RunWithSpinner(cmd.Context(), out, "Creating…", errCancelled,
		func(ctx context.Context, progress ui.Progress) (domain.Report, error) {
			return create.Scaffold(ctx, request, progressObserver{progress: progress})
		}); err != nil {
		if errors.Is(err, errCancelled) {
			return err
		}

		return fmt.Errorf("create: %w", err)
	}

	if err := runInit(cmd, initCommand, dir); err != nil {
		return err
	}

	if err := offerPush(cmd, create, flags, request); err != nil {
		return err
	}

	return ui.WriteLine(out, ui.Style(ui.ToneFaint).Render("Project created in "+target))
}

// buildRequest turns the flags into the use case's contract, asking for whatever they left out when
// there is someone to ask. Both answers are optional, so a script that gave neither flag gets
// neither rather than an error.
func buildRequest(cmd *cobra.Command, flags *createFlags, dir string) (application.Request, error) {
	description, remote := flags.description, flags.remote

	if stdinIsTerminal() {
		var fields []huh.Field

		if !cmd.Flags().Changed("description") {
			fields = append(fields, descriptionField(&description))
		}

		if !cmd.Flags().Changed("remote") {
			fields = append(fields, remoteField(&remote))
		}

		if len(fields) > 0 {
			if err := runForm(cmd.Context(), huh.NewGroup(fields...)); err != nil {
				return application.Request{}, err
			}
		}
	}

	request := application.Request{Dir: dir, Title: filepath.Base(dir)}

	if trimmed := strings.TrimSpace(description); trimmed != "" {
		request.Description = mo.Some(trimmed)
	}

	if trimmed := strings.TrimSpace(remote); trimmed != "" {
		request.Remote = mo.Some(trimmed)
	}

	// "the" opens the sentence rather than the flag name: Fang title-cases the first word of every
	// error it renders.
	if flags.push && request.Remote.IsAbsent() {
		return application.Request{}, errors.New("the --push flag needs a remote to push to; pass --remote as well")
	}

	return request, nil
}

// runInit runs init in the new directory. Init reads the working directory rather than taking one,
// so the process moves there first; nothing after this point needs the directory it started in.
//
// A failure here leaves a project that is created but not set up, so the error says where it is and
// what finishes the job.
func runInit(cmd, initCommand *cobra.Command, dir string) error {
	if initCommand.RunE == nil {
		return errors.New("create: init has no command body to run")
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	initCommand.SetContext(cmd.Context())
	initCommand.SetOut(cmd.OutOrStdout())
	initCommand.SetErr(cmd.ErrOrStderr())

	if err := initCommand.RunE(initCommand, nil); err != nil {
		return fmt.Errorf("%w (the project is created in %s; run codefall init there to finish)", err, dir)
	}

	return nil
}

// offerPush pushes when --push says to, asks when nobody said and there is someone to ask, and does
// nothing otherwise. A project without a remote has nowhere to push, so it is not asked at all.
func offerPush(cmd *cobra.Command, create CreateUseCase, flags *createFlags, request application.Request) error {
	url, ok := request.Remote.Get()
	if !ok {
		return nil
	}

	push := flags.push

	if !cmd.Flags().Changed("push") {
		if !stdinIsTerminal() {
			return nil
		}

		if err := runForm(cmd.Context(), huh.NewGroup(pushField(&push, url))); err != nil {
			return err
		}
	}

	if !push {
		return nil
	}

	out := ui.NewWriter(cmd.OutOrStdout())

	_, err := ui.RunWithSpinner(cmd.Context(), out, domain.PushStep.Title+"…", errCancelled,
		func(ctx context.Context, progress ui.Progress) (domain.StepResult, error) {
			result, err := create.Push(ctx, request.Dir)
			if err != nil {
				return domain.StepResult{}, err
			}

			progress.Line(stepLine(result))

			return result, nil
		})

	switch {
	case err == nil:
		return nil
	case errors.Is(err, errCancelled):
		return err
	default:
		return fmt.Errorf("create: %s: %w", domain.PushStep.ID, err)
	}
}

func descriptionField(description *string) huh.Field {
	return huh.NewInput().
		Title("Describe the project in a sentence. Leave it blank for none.").
		Value(description)
}

func remoteField(remote *string) huh.Field {
	return huh.NewInput().
		Title("Which git remote should " + domain.RemoteName + " point at? Leave it blank for none.").
		Placeholder("git@github.com:owner/name.git").
		Value(remote)
}

// pushField asks the last question. No is the starting value, because a push is visible to everyone
// who can see the remote.
func pushField(push *bool, url string) huh.Field {
	return huh.NewConfirm().
		Title("Push to " + domain.RemoteName + "?").
		Description(url + "\nThe files init wrote after bd init's commit stay uncommitted either way.").
		Affirmative("Yes").
		Negative("No").
		Value(push)
}

// runForm runs one form. A cancelled command cancels it (ADR-002) and ACCESSIBLE chooses the
// screen-reader mode.
func runForm(ctx context.Context, group *huh.Group) error {
	err := huh.NewForm(group).
		WithAccessible(os.Getenv("ACCESSIBLE") != "").
		RunWithContext(ctx)

	switch {
	case err == nil:
		return nil
	case errors.Is(err, huh.ErrUserAborted):
		return errCancelled
	default:
		return fmt.Errorf("create: %w", err)
	}
}

// stdinIsTerminal reports whether there is someone to answer a question. It is a variable so the
// tests can take the path a script takes; nothing else reassigns it.
var stdinIsTerminal = func() bool {
	return term.IsTerminal(os.Stdin.Fd())
}

// stepLine is one finished step: its mark, then the sentence the step wrote about itself. The marks
// and tones are init's, so the two commands' output reads as one run.
func stepLine(result domain.StepResult) string {
	return outcomeStyle(result.Outcome).Render(glyph(result.Outcome)) + " " + result.Detail
}

func glyph(outcome domain.Outcome) string {
	switch outcome {
	case domain.OutcomeDone:
		return "✓"
	case domain.OutcomeSkipped:
		return "-"
	default:
		return "?"
	}
}

func outcomeStyle(outcome domain.Outcome) lipgloss.Style {
	switch outcome {
	case domain.OutcomeDone:
		return ui.Style(ui.TonePrimary)
	case domain.OutcomeSkipped:
		return ui.Style(ui.ToneWarn)
	default:
		return ui.Style(ui.ToneNone)
	}
}

// progressObserver is what create has to say while it works, in the shared module's terms.
type progressObserver struct {
	progress ui.Progress
}

func (o progressObserver) StepStarted(step domain.Step) {
	o.progress.Label(step.Title + "…")
}

func (o progressObserver) StepFinished(result domain.StepResult) {
	o.progress.Line(stepLine(result))
}
