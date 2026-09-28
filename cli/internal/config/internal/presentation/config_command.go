// Package presentation builds `codefall config`: the interactive editor a person opens with no
// arguments in a terminal, and the subcommands a script runs. Both are thin over the one use case:
// the subcommands read their arguments, call it, and print what came back, and the editor collects
// the same calls from key presses and prints the same lines once it has left the screen (ADR-012).
package presentation

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// ConfigUseCase is what the commands need from the application layer, declared by their consumer.
type ConfigUseCase interface {
	Show(dir string) (domain.Configuration, error)
	Agents(dir string) ([]settings.Entry, error)
	SetList(dir, active, feature string, agents []settings.Agent) (domain.Write, error)
	ClearList(dir, active, feature string) (domain.Write, error)
	SetPosting(dir string, on bool) (domain.Write, error)
	Persona(dir string) (domain.Persona, error)
	SetPersona(dir, name string) ([]domain.Write, error)
}

// stdoutIsTerminal reports whether there is a screen to draw the editor on. It is a variable so the
// tests can take the path a script takes; nothing else reassigns it.
var stdoutIsTerminal = ui.StdoutIsTerminal

// settingsName is the settings file as a person reads its path.
const settingsName = ".codefall/settings.json"

// NewConfigCommand builds `codefall config`. With no arguments in a terminal it opens the editor;
// anywhere else it prints what `show` prints and says where the editor is. The subcommands are for
// scripts, and for anyone who knows exactly what to change.
func NewConfigCommand(config ConfigUseCase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Edit who reviews and consults for this project, and your persona",
		Long: "With no arguments in a terminal, opens an editor over the agents, reviews, and your persona. " +
			"The agents are who codefall asks to review or for a second opinion, chosen by the harness you " +
			"are running in; they and the review settings live in " + settingsName + ", which is checked in. " +
			"The persona lives in " + userfile.Name + ", which is yours alone and kept out of version " +
			"control. The subcommands change one value each, take every value as an argument, and never " +
			"prompt, so a script can run them. A write the settings would not accept is refused, in the " +
			"editor and the subcommands alike, and the file is left as it was.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if stdoutIsTerminal() {
				return runEditor(cmd.Context(), config, dir, cmd.OutOrStdout())
			}

			shown, err := config.Show(dir)
			if err != nil {
				return err
			}

			lines := append(configurationLines(shown), "",
				"Run codefall config in a terminal to edit these, or use its subcommands; codefall config --help lists them.")

			return writeLines(cmd.OutOrStdout(), lines)
		},
	}

	cmd.AddCommand(
		newShowCommand(config),
		newAgentsCommand(config),
		newReviewCommand(config),
		newPersonaCommand(config),
	)

	return cmd
}

func newShowCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print who reviews and consults, whether review posts, and your persona",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			shown, err := config.Show(dir)
			if err != nil {
				return err
			}

			return writeLines(cmd.OutOrStdout(), configurationLines(shown))
		},
	}
}

// configurationLines is the configuration as show prints it: plain text, one fact per line, so it
// reads in a terminal and greps in a script.
func configurationLines(shown domain.Configuration) []string {
	heading := "agents:"
	if shown.Default {
		heading = "agents (the built-in default; " + settingsName + " sets none):"
	}

	lines := append([]string{heading}, entryLines(shown.Entries)...)

	posting := "off"
	if shown.Posting {
		posting = "on"
	}

	lines = append(lines, "review posting: "+posting)

	return append(lines, "persona: "+personaLine(shown.Persona))
}

// entryLines is the agents list, one active agent per heading and one list per line under it.
func entryLines(entries []settings.Entry) []string {
	var lines []string

	for _, entry := range entries {
		lines = append(lines, "  "+activeAgentLabel(entry.ActiveAgent)+":")

		for _, feature := range settings.Features() {
			lines = append(lines, "    "+feature+": "+listLabel(entry.List(feature)))
		}
	}

	return lines
}

// activeAgentLabel is an active agent as a person reads it.
func activeAgentLabel(active string) string {
	if active == settings.ActiveDefault {
		return "default (any harness without its own entry)"
	}

	return "when running in " + active
}

// listLabel is one list as a person reads it: the agents in order, or what an absent list means.
func listLabel(agents mo.Option[[]settings.Agent]) string {
	list, ok := agents.Get()
	if !ok {
		return "same as default"
	}

	parts := make([]string, 0, len(list))
	for _, agent := range list {
		parts = append(parts, agentLabel(agent))
	}

	return strings.Join(parts, ", then ")
}

// agentLabel is one agent as a person reads it: the harness and its model, with current spelled out.
func agentLabel(agent settings.Agent) string {
	if agent.Harness == settings.HarnessCurrent {
		return "this harness"
	}

	return settings.DescribeAgent(agent)
}

func newAgentsCommand(config ConfigUseCase) *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use: "agents [<" + strings.Join(settings.ActiveAgents(), "|") + "> <" +
			strings.Join(settings.Features(), "|") + "> <harness[:model]>...]",
		Short: "Print the agents, or set who reviews or consults for sessions in one harness",
		Long: "With no arguments, prints the agents list. With arguments, sets one list in " + settingsName +
			": the active agent is the harness you run codefall in (" + strings.Join(settings.ActiveAgents(), ", ") +
			"; default covers every harness without an entry of its own), the feature is " +
			strings.Join(settings.Features(), " or ") + ", and each agent is a harness (" +
			strings.Join(settings.AgentHarnesses(), ", ") + ") with an optional model after a colon; " +
			settings.HarnessCurrent + " is a subagent of the harness you are running in. Agents are tried in the order " +
			"given. --clear in place of the agents removes the list, so the default entry's applies.",
		Args: agentsArgs(&clear),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(args) == 0 {
				return runAgentsList(cmd, config, dir)
			}

			var write domain.Write

			if clear {
				write, err = config.ClearList(dir, args[0], args[1])
			} else {
				agents, parseErr := domain.ParseAgents(args[2:])
				if parseErr != nil {
					return parseErr
				}

				write, err = config.SetList(dir, args[0], args[1], agents)
			}

			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "remove the list, so the default entry's applies")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Print the agents, one active agent at a time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			return runAgentsList(cmd, config, dir)
		},
	})

	return cmd
}

// agentsArgs checks the agents command's arguments: nothing, to list; or an active agent, a feature,
// and at least one agent to set, or none with --clear.
func agentsArgs(clear *bool) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		switch {
		case len(args) == 0:
			return nil
		case len(args) == 1:
			return fmt.Errorf("name the list too: %s", strings.Join(settings.Features(), " or "))
		case *clear && len(args) > 2:
			return errors.New("--clear takes no agents")
		case !*clear && len(args) == 2:
			return errors.New("name at least one agent as harness[:model], or pass --clear")
		}

		return nil
	}
}

func runAgentsList(cmd *cobra.Command, config ConfigUseCase, dir string) error {
	entries, err := config.Agents(dir)
	if err != nil {
		return err
	}

	return writeLines(cmd.OutOrStdout(), entryLines(entries))
}

func newReviewCommand(config ConfigUseCase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Change how review behaves: whether it posts its findings to the pull request",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(&cobra.Command{
		Use:       "posting <on|off>",
		Short:     "Let review post its findings to the pull request, or not",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"on", "off"},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			var on bool

			switch args[0] {
			case "on":
				on = true
			case "off":
				on = false
			default:
				return fmt.Errorf("posting is on or off, not %q", args[0])
			}

			write, err := config.SetPosting(dir, on)
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	})

	return cmd
}

func newPersonaCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "persona [" + strings.Join(userfile.Personas(), "|") + "]",
		Short: "Print your persona, or set it in " + userfile.Name,
		Long: "With no argument, prints the persona the skills use for you and where it comes from. " +
			"With one, writes it to " + userfile.Name + ", creating the file when there is none and " +
			"keeping whatever else it holds, and adds the file to .gitignore when it is not there.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: userfile.Personas(),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(args) == 0 {
				persona, err := config.Persona(dir)
				if err != nil {
					return err
				}

				return writeLines(cmd.OutOrStdout(), []string{personaLine(persona)})
			}

			writes, err := config.SetPersona(dir, args[0])
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), writes...)
		},
	}
}

// personaLine is the persona and where it came from.
func personaLine(persona domain.Persona) string {
	if persona.FromFile {
		return persona.Name + " (from " + userfile.Name + ")"
	}

	return persona.Name + " (default)"
}

// writeLines prints plain lines through the colour-profile writer.
func writeLines(w io.Writer, lines []string) error {
	out := ui.NewWriter(w)

	for _, line := range lines {
		if err := ui.WriteLine(out, line); err != nil {
			return err
		}
	}

	return nil
}

// writeResults prints one line per write in the voice init and upgrade use: a tick for a file that
// changed, a dash for one that already said what was asked.
func writeResults(w io.Writer, writes ...domain.Write) error {
	out := ui.NewWriter(w)

	for _, write := range writes {
		mark, tone := "✓", ui.TonePrimary
		if !write.Changed {
			mark, tone = "-", ui.ToneWarn
		}

		if err := ui.WriteLine(out, ui.Style(tone).Render(mark)+" "+write.Detail); err != nil {
			return err
		}
	}

	return nil
}

// given is a form field's value as an option: a field left empty was not given.
func given(value string) mo.Option[string] {
	if strings.TrimSpace(value) == "" {
		return mo.None[string]()
	}

	return mo.Some(value)
}
