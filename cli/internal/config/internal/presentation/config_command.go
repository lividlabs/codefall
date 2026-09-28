// Package presentation builds `codefall config` and its subcommands. They are thin: each reads the
// working directory and its arguments, calls the use case, and prints what came back. Nothing here
// prompts — every value is an argument or a flag — so the command runs the same under a script.
package presentation

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/samber/mo"
	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// ConfigUseCase is what the commands need from the application layer, declared by their consumer.
type ConfigUseCase interface {
	Show(dir string) (domain.Configuration, error)
	Agents(dir string) ([]settings.Agent, error)
	AddAgent(dir string, agent application.NewAgent) (domain.Write, error)
	RemoveAgent(dir, name string) (domain.Write, error)
	OrderAgents(dir string, order []string) (domain.Write, error)
	SetOrder(dir string, order application.Order, names []string) (domain.Write, error)
	ClearOrder(dir string, order application.Order) (domain.Write, error)
	Persona(dir string) (domain.Persona, error)
	SetPersona(dir, name string) ([]domain.Write, error)
}

// NewConfigCommand builds `codefall config`, which holds the subcommands and does nothing itself.
func NewConfigCommand(config ConfigUseCase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show and change this project's agents and your persona",
		Long: "Reads and writes the settings a person would otherwise edit by hand: the agents list and " +
			"the orders review, consult, and each harness walk in .codefall/settings.json, which is " +
			"checked in, and the persona in " + userfile.Name +
			", which is yours alone and kept out of git. Nothing prompts; every value is an argument, " +
			"so a script can run it. A write the settings would not accept is refused and the file is " +
			"left as it was.",
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(
		newShowCommand(config),
		newAgentsCommand(config),
		newUseCommand(config, settings.BlockReview, application.ReviewOrder()),
		newUseCommand(config, settings.BlockConsult, application.ConsultOrder()),
		newPersonaCommand(config),
	)

	return cmd
}

func newShowCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the effective agents, their orders, and your persona",
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
		heading = "agents (the default; " + settingsName + " lists none):"
	}

	lines := []string{heading}

	for at, agent := range shown.Agents {
		lines = append(lines, fmt.Sprintf("  %d. %s", at+1, settings.DescribeAgent(agent)))
	}

	if order, ok := shown.Review.Get(); ok {
		lines = append(lines, settings.BlockReview+"."+settings.FieldReviewAgents+": "+strings.Join(order, ", "))
	}

	if order, ok := shown.Consult.Get(); ok {
		lines = append(lines, settings.BlockConsult+"."+settings.FieldConsultAgents+": "+strings.Join(order, ", "))
	}

	for _, key := range slices.Sorted(maps.Keys(shown.ByHarness)) {
		lines = append(lines, settings.FieldAgentsByHarness+"."+key+": "+strings.Join(shown.ByHarness[key], ", "))
	}

	return append(lines, "persona: "+personaLine(shown.Persona))
}

// settingsName is the settings file as a person reads its path.
const settingsName = ".codefall/settings.json"

func newAgentsCommand(config ConfigUseCase) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "List, add, remove, and order the agents in " + settingsName + ", and set a harness's order",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		newAgentsListCommand(config),
		newAgentsAddCommand(config),
		newAgentsRemoveCommand(config),
		newAgentsOrderCommand(config),
		newAgentsForCommand(config),
	)

	return cmd
}

func newAgentsListCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Print the agents in order, one per line",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			agents, err := config.Agents(dir)
			if err != nil {
				return err
			}

			lines := make([]string, 0, len(agents))
			for _, agent := range agents {
				lines = append(lines, settings.DescribeAgent(agent))
			}

			return writeLines(cmd.OutOrStdout(), lines)
		},
	}
}

// addFlags is what `agents add` can be told beside the name.
type addFlags struct {
	harness string
	model   string
	before  string
	after   string
}

func newAgentsAddCommand(config ConfigUseCase) *cobra.Command {
	flags := &addFlags{}

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add an agent to the list, last unless --before or --after says where",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			write, err := config.AddAgent(dir, application.NewAgent{
				Name:    args[0],
				Harness: flags.harness,
				Model:   given(flags.model),
				Before:  given(flags.before),
				After:   given(flags.after),
			})
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}

	cmd.Flags().StringVar(&flags.harness, "harness", "",
		"the harness that runs the agent ("+strings.Join(settings.AgentHarnesses(), ", ")+"); "+
			settings.HarnessCurrent+" is whichever harness is running the session")
	cmd.Flags().StringVar(&flags.model, "model", "", "the model to ask that harness for, passed to it as written")
	cmd.Flags().StringVar(&flags.before, "before", "", "put the agent straight before this one")
	cmd.Flags().StringVar(&flags.after, "after", "", "put the agent straight after this one")

	_ = cmd.MarkFlagRequired("harness")

	cmd.MarkFlagsMutuallyExclusive("before", "after")

	return cmd
}

func newAgentsRemoveCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an agent from the list, unless an order still names it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			write, err := config.RemoveAgent(dir, args[0])
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}
}

func newAgentsOrderCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "order <name>...",
		Short: "Replace the list's order; every agent is named exactly once",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			write, err := config.OrderAgents(dir, args)
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}
}

// newUseCommand builds `codefall config review` or `codefall config consult`, which holds the command
// that sets that use's own order.
func newUseCommand(config ConfigUseCase, use string, order application.Order) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: "Set the order " + use + " walks, in " + settingsName,
		Args:  cobra.NoArgs,
	}

	var clear bool

	agents := &cobra.Command{
		Use:   "agents <name>...",
		Short: "Set the order " + use + " walks, or --clear it so the wider order applies",
		Long: "Sets " + use + ".agents in " + settingsName + " to the agents named, in that order. Each " +
			"must be an agent the list defines, and each is named once; the order need not name every " +
			"agent. --clear removes the key, so " + use + " walks the per-harness order or the top-level one.",
		Args: orderArgs(&clear, 0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, config, order, clear, args)
		},
	}

	agents.Flags().BoolVar(&clear, "clear", false, "remove the order, so the wider one applies")
	cmd.AddCommand(agents)

	return cmd
}

// newAgentsForCommand builds `codefall config agents for <harness>`, which sets the order a session
// running in that harness walks.
func newAgentsForCommand(config ConfigUseCase) *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use:   "for <harness> <name>...",
		Short: "Set the order a session in one harness walks, or --clear it",
		Long: "Sets " + settings.FieldAgentsByHarness + ".<harness> in " + settingsName + " to the agents " +
			"named, in that order, for a session running in that harness (" +
			strings.Join(harness.All(), ", ") + "). Each must be an agent the list defines, and each is " +
			"named once. --clear removes the harness's order, and " + settings.FieldAgentsByHarness +
			" with its last one, so the session walks each use's own order or the top-level one.",
		Args:      orderArgs(&clear, 1),
		ValidArgs: harness.All(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOrder(cmd, config, application.HarnessOrder(args[0]), clear, args[1:])
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "remove this harness's order, so the wider one applies")

	return cmd
}

// orderArgs checks an order command's arguments: the fixed ones first, then at least one agent name
// to set the order, or none with --clear.
func orderArgs(clear *bool, fixed int) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) < fixed {
			return fmt.Errorf("name a harness first (%s)", strings.Join(harness.All(), ", "))
		}

		names := len(args) - fixed

		switch {
		case *clear && names > 0:
			return errors.New("--clear takes no agent names")
		case !*clear && names == 0:
			return errors.New("name at least one agent, or pass --clear")
		}

		return nil
	}
}

// runOrder sets or clears one order and prints what that did.
func runOrder(cmd *cobra.Command, config ConfigUseCase, order application.Order, clear bool, names []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	var write domain.Write

	if clear {
		write, err = config.ClearOrder(dir, order)
	} else {
		write, err = config.SetOrder(dir, order, names)
	}

	if err != nil {
		return err
	}

	return writeResults(cmd.OutOrStdout(), write)
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

// given is a flag's value as an option: a flag left empty was not given.
func given(value string) mo.Option[string] {
	if strings.TrimSpace(value) == "" {
		return mo.None[string]()
	}

	return mo.Some(value)
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
