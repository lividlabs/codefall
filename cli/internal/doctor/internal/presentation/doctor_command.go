// Package presentation builds doctor's command. It is thin: it reads the working directory, calls
// the use case, and prints what comes back. The palette, the writer, and the spinner are the shared
// UI module's; what belongs here is which tone a check's status is drawn in.
package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
)

// heading is the report's first line, drawn as a chip. The colorprofile writer strips the styling on
// a non-terminal stdout and under NO_COLOR, leaving the word itself.
const heading = "DOCTOR SUMMARY"

// errCancelled is what an interrupted run reports. It is returned as it is rather than wrapped, so
// Fang renders that sentence and not a prefix in front of it.
var errCancelled = errors.New("doctor cancelled")

// DiagnoseUseCase is what the command needs from the application layer, declared by its consumer.
type DiagnoseUseCase interface {
	Run(ctx context.Context, dir string) (domain.Report, error)
}

// NewDoctorCommand builds `codefall doctor`.
func NewDoctorCommand(diagnose DiagnoseUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that this directory and your tools are ready for codefall",
		Long: "Reports on .codefall/settings.json and .codefall/user.json, whether codefall is installed for each harness it " +
			"records and still installed for any it no longer names, whether the local start and " +
			"update commands are declared and can be run, Beads, and gh. It never changes anything.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// The working directory is the one piece of environment the command reads; the use case
			// receives it as a value.
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("doctor: %w", err)
			}

			// One colour-profile writer for the whole report.
			out := ui.NewWriter(cmd.OutOrStdout())

			report, err := runDiagnose(cmd.Context(), diagnose, dir, out)
			if err != nil {
				if errors.Is(err, errCancelled) {
					return err
				}

				return fmt.Errorf("doctor: %w", err)
			}

			if err := renderReport(out, report); err != nil {
				return err
			}

			issues := report.SectionsWithIssues()

			// Fang renders this; nothing here calls os.Exit. A failing run prints no summary line
			// because the error is the summary.
			if report.Failed() > 0 {
				return fmt.Errorf("doctor found issues in %d %s",
					issues, plural(issues, "category", "categories"))
			}

			return ui.WriteLine(out, summaryLine(issues))
		},
	}
}

// spinnerLabel is what the spinner says while the checks run. The checks shell out to bd and gh, so
// a run is slow enough to be worth acknowledging.
const spinnerLabel = "Running checks…"

// runDiagnose runs the use case under the shared spinner. Doctor has nothing to say while it works —
// the report is the whole of what it has to say — so it ignores the progress it is handed.
func runDiagnose(
	ctx context.Context, diagnose DiagnoseUseCase, dir string, out io.Writer,
) (domain.Report, error) {
	return ui.RunWithSpinner(ctx, out, spinnerLabel, errCancelled,
		func(ctx context.Context, _ ui.Progress) (domain.Report, error) {
			return diagnose.Run(ctx, dir)
		})
}

// renderReport prints the heading, then one header line per category and, under it, one line per
// problem. A check that passed says all it has to say in its section's header.
func renderReport(w io.Writer, report domain.Report) error {
	// The trailing newline is the blank line between the heading and the first section.
	if err := ui.WriteLine(w, ui.HeadingStyle().Render(heading)+"\n"); err != nil {
		return err
	}

	for _, section := range report.Sections() {
		if err := ui.WriteLine(w, sectionTree(section)); err != nil {
			return err
		}
	}

	return nil
}

// sectionTree draws one category: the header is the root, each problem is a child of it, and a
// remedy is a child of the problem it repairs. A section whose checks all passed is a root on its
// own, because its header already says everything there is to say.
func sectionTree(section domain.Section) string {
	guide := faintStyle().PaddingRight(1)

	root := tree.Root(sectionHeader(section)).
		Enumerator(enumerator).
		Indenter(indenter).
		EnumeratorStyle(guide).
		IndenterStyle(guide)

	for _, result := range section.Results {
		if result.Status == domain.StatusPass {
			continue
		}

		problem := tree.Root(problemLine(result))

		if remedy, ok := result.Remedy.Get(); ok {
			problem.Child(faintStyle().Render("fix: " + remedy))
		}

		root.Child(problem)
	}

	return root.String()
}

// enumerator draws the elbow in front of a child, rounded on the last one. Two characters rather
// than the tree package's three, which keeps a remedy's text at the column its problem's mark sits
// in.
func enumerator(children tree.Children, index int) string {
	if children.Length()-1 == index {
		return "╰─"
	}

	return "├─"
}

// indenter carries the vertical down the left of a problem's remedy while the section still has
// problems to come, and drops it once there are none.
func indenter(children tree.Children, index int) string {
	if children.Length()-1 == index {
		return "  "
	}

	return "│ "
}

// sectionHeader is the category's status, its title, and whatever its passing checks reported about
// themselves — the versions and the account, the things worth knowing when nothing is wrong.
func sectionHeader(section domain.Section) string {
	header := bracketedMark(section.Status()) + " " + section.Category.Title

	var details []string

	for _, result := range section.Results {
		if detail, ok := result.Detail.Get(); ok && result.Status == domain.StatusPass {
			details = append(details, detail)
		}
	}

	if len(details) > 0 {
		header += " " + faintStyle().Render("("+strings.Join(details, ", ")+")")
	}

	return header
}

// problemLine is a warning or a failure, hung off its header by the tree. The check's title is not
// printed: the detail is written as a sentence that stands on its own.
func problemLine(result domain.Result) string {
	line := mark(result.Status)

	if detail, ok := result.Detail.Get(); ok {
		line += " " + detail
	}

	return line
}

// summaryLine closes a run that had nothing to fail on, so it carries the status it is reporting:
// the pass colour when the report is clean, the warning colour when it is not. A failing run never
// reaches here — its error is its summary, and Fang renders that.
func summaryLine(issues int) string {
	if issues == 0 {
		return statusStyle(domain.StatusPass).Render("• No issues found.")
	}

	return statusStyle(domain.StatusWarn).Render(
		fmt.Sprintf("• Doctor found issues in %d %s.", issues, plural(issues, "category", "categories")))
}

// mark is the bare glyph a problem line hangs off, in its status's colour.
func mark(status domain.Status) string {
	return statusStyle(status).Render(glyph(status))
}

// bracketedMark is the header's form of the same thing. The brackets are rendered inside the style
// rather than around it, so the three characters read as one token instead of a coloured glyph
// between two uncoloured ones.
func bracketedMark(status domain.Status) string {
	return statusStyle(status).Render("[" + glyph(status) + "]")
}

func glyph(status domain.Status) string {
	return ui.Mark(tone(status))
}

// statusStyle is the one place a status becomes a style. An unknown status is left unstyled.
func statusStyle(status domain.Status) lipgloss.Style {
	return ui.Style(tone(status))
}

// tone is doctor's whole share of the palette: which of the shared tones each status is drawn in.
// Only the mark is styled — the rest of a line is a path, a version, or a command to type, where
// colour would be decoration rather than information.
func tone(status domain.Status) ui.Tone {
	switch status {
	case domain.StatusPass:
		return ui.TonePrimary
	case domain.StatusWarn:
		return ui.ToneWarn
	case domain.StatusFail:
		return ui.ToneFail
	default:
		return ui.ToneNone
	}
}

// Secondary text is dimmed and tinted teal-grey: the versions and accounts in a header, and the
// remedy under a problem. What is left at full strength is the sentence that says what is wrong,
// which is the only thing a reader has to take in.
func faintStyle() lipgloss.Style {
	return ui.Style(ui.ToneFaint)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}

	return many
}
