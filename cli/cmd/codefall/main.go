// Command codefall is the CLI's entry point and the app's composition root: it builds the one
// injector, calls each component's registration function, mounts the commands those components
// expose, and hands the tree to Fang (ADR-GO-01, ADR-002).
package main

import (
	"context"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/term"
	"github.com/samber/do/v2"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/create"
	"github.com/lividlabs/codefall-cli/cli/internal/doctor"
	"github.com/lividlabs/codefall-cli/cli/internal/initcmd"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/buildinfo"
)

func main() {
	// Fang has already rendered the error; only the exit status is left.
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// run holds the body so it is testable. Taking the arguments and both streams as parameters keeps
// main() the only part of the app bound to the real process.
func run(args []string, stdout, stderr io.Writer) error {
	// Cobra reads os.Args when its argument slice is nil, which under `go test` is the test binary's
	// own flags.
	if args == nil {
		args = []string{}
	}

	injector := do.New()
	defer func() {
		if report := injector.Shutdown(); !report.Succeed {
			_, _ = fmt.Fprintln(stderr, report.Error())
		}
	}()

	create.Register(injector)
	doctor.Register(injector)
	initcmd.Register(injector)

	root := &cobra.Command{
		Use:   "codefall",
		Short: "Codefall's command-line tool",
	}
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.AddCommand(doctor.Command(injector))
	root.AddCommand(initcmd.Commands(injector)...)
	// create runs init in the directory it makes, so it gets an init command of its own.
	root.AddCommand(create.Command(injector, initcmd.Command(injector)))

	return fang.Execute(
		context.Background(),
		root,
		// The string carries its own commit, so fang.WithCommit is not used: Fang would append a
		// second one, and it has no way to say a build had uncommitted changes.
		fang.WithVersion(buildinfo.Display()),
		fang.WithColorSchemeFunc(codefallColorScheme),
		fang.WithErrorHandler(renderError),
	)
}

// codefallColorScheme is the teal trail palette. It replaces Fang's default
// playful purples and pinks with desaturated teals for primary UI, muted amber
// for warnings, and muted coral for errors, matching the brand image.
// Light and dark variants are chosen the way Fang does, via the LightDark
// function it passes in.
func codefallColorScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           c(lipgloss.Color("#3A3943"), lipgloss.Color("#DFDBDD")),
		Title:          c(lipgloss.Color("#2F6B6B"), lipgloss.Color("#6AB3B3")),
		Description:    c(lipgloss.Color("#3A3943"), lipgloss.Color("#DFDBDD")),
		Codeblock:      c(lipgloss.Color("#E8F0F0"), lipgloss.Color("#1E3436")),
		Program:        c(lipgloss.Color("#2F6B6B"), lipgloss.Color("#6AB3B3")),
		DimmedArgument: c(lipgloss.Color("#526D71"), lipgloss.Color("#8AA3A8")),
		Comment:        c(lipgloss.Color("#526D71"), lipgloss.Color("#8AA3A8")),
		Flag:           c(lipgloss.Color("#2F6B6B"), lipgloss.Color("#6AB3B3")),
		FlagDefault:    c(lipgloss.Color("#526D71"), lipgloss.Color("#8AA3A8")),
		Command:        c(lipgloss.Color("#2F6B6B"), lipgloss.Color("#6AB3B3")),
		QuotedString:   c(lipgloss.Color("#7E6217"), lipgloss.Color("#D9B44A")),
		Argument:       c(lipgloss.Color("#3A3943"), lipgloss.Color("#DFDBDD")),
		Help:           c(lipgloss.Color("#3A3943"), lipgloss.Color("#DFDBDD")),
		Dash:           c(lipgloss.Color("#526D71"), lipgloss.Color("#8AA3A8")),
		ErrorHeader: [2]color.Color{
			c(lipgloss.Color("#F5E6E3"), lipgloss.Color("#F5E6E3")),
			c(lipgloss.Color("#9E3A3A"), lipgloss.Color("#A8433C")),
		},
		ErrorDetails: c(lipgloss.Color("#9E3A3A"), lipgloss.Color("#E08878")),
	}
}

// renderError is Fang's error block on one line. The default indents the ERROR chip by two columns,
// puts a blank line above and below it, and then writes the message on its own indented line below
// that — four lines for one sentence, which under doctor's report reads as a broken layout rather
// than a summary. This keeps Fang's own styles, colours, capitalisation, trailing period, and
// non-terminal escape hatch, and only unsets the margins and the width that split the two apart.
//
// AGENTS.md allows changing Fang's rendering through its `With*` options and nothing else, and
// `main` is the only file that may import Fang, so the handler lives here.
func renderError(w io.Writer, styles fang.Styles, err error) {
	if w, ok := w.(term.File); ok {
		// if stderr is not a tty, simply print the error without any
		// styling or going through an [ErrorHandler]:
		if !term.IsTerminal(w.Fd()) {
			_, _ = fmt.Fprintln(w, err.Error())

			return
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w,
		styles.ErrorHeader.UnsetMargins().String()+" "+
			styles.ErrorText.UnsetWidth().UnsetMargins().Render(err.Error()+"."))
	_, _ = fmt.Fprintln(w)

	// The hint after a mistyped command or flag, exactly as Fang writes it.
	if isUsageError(err) {
		_, _ = fmt.Fprintln(w, lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ErrorText.UnsetWidth().Render("Try"),
			styles.Program.Flag.Render(" --help "),
			styles.ErrorText.UnsetWidth().UnsetMargins().UnsetTransform().Render("for usage."),
		))
		_, _ = fmt.Fprintln(w)
	}
}

// isUsageError is Fang's own unexported test for the errors Cobra reports without a type, copied
// because the hint above cannot ask Fang the question. See
// https://github.com/spf13/cobra/pull/2266.
func isUsageError(err error) bool {
	s := err.Error()
	for _, prefix := range []string{
		"flag needs an argument:",
		"unknown flag:",
		"unknown shorthand flag:",
		"unknown command",
		"invalid argument",
	} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}
