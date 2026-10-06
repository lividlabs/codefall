// Package presentation builds `codefall config`: the interactive editor a person opens with no
// arguments in a terminal, and the subcommands a script runs. Both are thin over the one use case:
// the subcommands read their arguments, call it, and print what came back, and the editor collects
// the same calls from key presses and prints the same lines once it has left the screen (ADR-012).
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

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
	"github.com/lividlabs/codefall/cli/internal/shared/ui"
	"github.com/lividlabs/codefall/cli/internal/shared/userfile"
)

// ConfigUseCase is what the commands need from the application layer, declared by their consumer.
type ConfigUseCase interface {
	Show(dir string) (domain.Configuration, error)
	Agents(dir string) ([]settings.Entry, error)
	SetList(dir, active, feature string, agents []settings.Agent) (domain.Write, error)
	ClearList(dir, active, feature string) (domain.Write, error)
	SetPosting(dir string, on bool) (domain.Write, error)
	SetSkillPrefix(dir, prefix string) (domain.Write, error)
	HarnessConfigs(dir string) (map[string]settings.HarnessConfig, error)
	SetHarnessConfig(dir, key string, block settings.HarnessConfig) (domain.Write, error)
	ClearHarnessConfig(dir, key string) (domain.Write, error)
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
		Short: "Edit who reviews and consults for this project, how each harness is called, what the skills are called, and your persona",
		Long: "With no arguments in a terminal, opens an editor over the agents, reviews, the skill prefix, and your persona. " +
			"The agents are who codefall asks to review or for a second opinion, chosen by the harness you " +
			"are running in; they, how each harness is called, the review settings, and the prefix the " +
			"installed skills are named with live in " + settingsName + ", which is checked in. " +
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
		newHarnessCommand(config),
		newReviewCommand(config),
		newSkillPrefixCommand(config),
		newPersonaCommand(config),
	)

	return cmd
}

func newShowCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print who reviews and consults, how each harness is called, whether review posts, what the skills are called, and your persona",
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
	lines = append(lines, harnessConfigLines(shown.HarnessConfigs)...)

	posting := "off"
	if shown.Posting {
		posting = "on"
	}

	lines = append(lines, "review posting: "+posting)
	lines = append(lines, "skill prefix: "+skillPrefixLine(shown.SkillPrefix))

	return append(lines, "persona: "+personaLine(shown.Persona))
}

// skillPrefixLine is the prefix as show prints it, with an example of a name under it, since the
// value on its own is a word and the reader wants to know what it does.
func skillPrefixLine(prefix string) string {
	if prefix == "" {
		prefix = settings.DefaultSkillPrefix
	}

	return prefix + " (" + prefix + "-design, /" + prefix + "-implement)"
}

// harnessConfigLines is how each harness is called, one block per heading in key order and one
// field per line under it. A block under a harness's own name does not repeat the harness.
func harnessConfigLines(configs map[string]settings.HarnessConfig) []string {
	var lines []string

	for _, key := range slices.Sorted(maps.Keys(configs)) {
		block := configs[key]
		lines = append(lines, settings.FieldHarnessConfig+"."+key+":")

		if block.Harness != key {
			lines = append(lines, "  "+settings.FieldConfigHarness+": "+block.Harness)
		}

		if flag, ok := block.ModelFlag.Get(); ok {
			lines = append(lines, "  "+settings.FieldModelFlag+": "+flag)
		}

		if provider, ok := block.Provider.Get(); ok {
			lines = append(lines, "  "+settings.FieldProvider+": "+provider)
		}

		if len(block.Args) > 0 {
			lines = append(lines, "  "+settings.FieldArgs+": "+strings.Join(block.Args, ", "))
		}

		if env, ok := block.Env.Get(); ok {
			lines = append(lines, "  "+settings.FieldEnv+": "+env)
		}
	}

	return lines
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

// harnessFlags is what `config harness <key>` takes: every field a block may carry, each as a flag,
// and --clear in place of them all.
type harnessFlags struct {
	harness, modelFlag, provider, env string
	args                              []string
	clear                             bool
}

// The flags, in the order the block writes their fields, for the message that names them.
const harnessFlagNames = "--harness, --model-flag, --provider, --arg, or --env"

func newHarnessCommand(config ConfigUseCase) *cobra.Command {
	flags := &harnessFlags{}

	cmd := &cobra.Command{
		Use: "harness [<key> [--harness <" + strings.Join(harness.All(), "|") + ">] [--model-flag <flag>] " +
			"[--provider <name>] [--arg <argument>]... [--env <command>] | <key> --clear]",
		Short: "Print how each harness is called, or set how one is called",
		Long: "With no arguments, prints the " + settings.FieldHarnessConfig + " blocks in " + settingsName +
			". With a key, replaces that block with exactly the flags given. A key that is a harness name (" +
			strings.Join(harness.All(), ", ") + ") configures that harness; any other key is a variant, and " +
			"--harness names the binary it runs. --model-flag is the flag the model is passed with, when the " +
			"harness's usual one is not it; --provider is the harness's own provider name; --arg is an extra " +
			"argument, appended as written, repeated for several; --env is a shell command whose output is " +
			"evaluated before the harness starts, so its export lines take effect. A harness with no block " +
			"runs bare. --clear removes the block. An agent in a list names a block by its key, so " +
			"codefall config agents claude review codex-direct:gpt-6-astra reviews on the codex-direct block.",
		Args: harnessArgs(flags),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(args) == 0 {
				return runHarnessList(cmd, config, dir)
			}

			var write domain.Write

			if flags.clear {
				write, err = config.ClearHarnessConfig(dir, args[0])
			} else {
				write, err = config.SetHarnessConfig(dir, args[0], settings.HarnessConfig{
					Harness:   flags.harness,
					ModelFlag: given(flags.modelFlag),
					Provider:  given(flags.provider),
					Env:       given(flags.env),
					Args:      flags.args,
				})
			}

			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}

	cmd.Flags().StringVar(&flags.harness, "harness", "", "the binary a variant runs ("+strings.Join(harness.All(), ", ")+")")
	cmd.Flags().StringVar(&flags.modelFlag, "model-flag", "", "the flag the model is passed with")
	cmd.Flags().StringVar(&flags.provider, "provider", "", "the harness's own provider name")
	cmd.Flags().StringArrayVar(&flags.args, "arg", nil, "an extra argument, appended as written; repeat for several")
	cmd.Flags().StringVar(&flags.env, "env", "", "a shell command whose output is evaluated before the harness starts")
	cmd.Flags().BoolVar(&flags.clear, "clear", false, "remove the block, so the harness runs bare")

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Print how each harness is called, one block at a time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			return runHarnessList(cmd, config, dir)
		},
	})

	return cmd
}

// harnessArgs checks the harness command's arguments: nothing, to list; or a key with at least one
// field flag, or with --clear alone.
func harnessArgs(flags *harnessFlags) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		fields := 0

		for _, name := range []string{"harness", "model-flag", "provider", "arg", "env"} {
			if cmd.Flags().Changed(name) {
				fields++
			}
		}

		switch {
		case len(args) > 1:
			return errors.New("name one key")
		case len(args) == 0 && (fields > 0 || flags.clear):
			return errors.New("name the key the flags are for")
		case len(args) == 0:
			return nil
		case flags.clear && fields > 0:
			return errors.New("--clear takes no other flags")
		case !flags.clear && fields == 0:
			return errors.New("pass at least one of " + harnessFlagNames + ", or --clear")
		}

		return nil
	}
}

func runHarnessList(cmd *cobra.Command, config ConfigUseCase, dir string) error {
	configs, err := config.HarnessConfigs(dir)
	if err != nil {
		return err
	}

	lines := harnessConfigLines(configs)
	if len(lines) == 0 {
		lines = []string{settings.FieldHarnessConfig + ": none set in " + settingsName + "; every harness runs bare"}
	}

	return writeLines(cmd.OutOrStdout(), lines)
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

// newSkillPrefixCommand is `codefall config skill-prefix`: print what the installed skills are called,
// or set it. Setting it writes the settings alone; the rename on disk is upgrade's, and the write's
// own sentence says so (ADR-015).
func newSkillPrefixCommand(config ConfigUseCase) *cobra.Command {
	return &cobra.Command{
		Use:   "skill-prefix [" + strings.Join(settings.SkillPrefixes(), "|") + "]",
		Short: "Print what the installed skills are called, or set the prefix they are installed under",
		Long: "With no argument, prints the prefix the installed skills are named with, as in codefall-design " +
			"and /codefall-implement. With one, records it in " + settingsName + "; the skills on disk keep " +
			"their names until codefall upgrade runs, which installs them under the new prefix and removes " +
			"the old ones.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: settings.SkillPrefixes(),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}

			if len(args) == 0 {
				shown, err := config.Show(dir)
				if err != nil {
					return err
				}

				return writeLines(cmd.OutOrStdout(), []string{"skill prefix: " + skillPrefixLine(shown.SkillPrefix)})
			}

			write, err := config.SetSkillPrefix(dir, args[0])
			if err != nil {
				return err
			}

			return writeResults(cmd.OutOrStdout(), write)
		},
	}
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
