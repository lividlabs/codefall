package presentation

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

type fakeConfig struct {
	shown   domain.Configuration
	agents  []settings.Agent
	persona domain.Persona
	write   domain.Write
	writes  []domain.Write
	err     error

	added   application.NewAgent
	removed string
	ordered []string
	set     string

	order   mo.Option[application.Order]
	names   []string
	cleared bool
}

func (f *fakeConfig) SetOrder(_ string, order application.Order, names []string) (domain.Write, error) {
	f.order, f.names = mo.Some(order), names

	return f.write, f.err
}

func (f *fakeConfig) ClearOrder(_ string, order application.Order) (domain.Write, error) {
	f.order, f.cleared = mo.Some(order), true

	return f.write, f.err
}

func (f *fakeConfig) Show(string) (domain.Configuration, error) { return f.shown, f.err }

func (f *fakeConfig) Agents(string) ([]settings.Agent, error) { return f.agents, f.err }

func (f *fakeConfig) AddAgent(_ string, agent application.NewAgent) (domain.Write, error) {
	f.added = agent

	return f.write, f.err
}

func (f *fakeConfig) RemoveAgent(_, name string) (domain.Write, error) {
	f.removed = name

	return f.write, f.err
}

func (f *fakeConfig) OrderAgents(_ string, order []string) (domain.Write, error) {
	f.ordered = order

	return f.write, f.err
}

func (f *fakeConfig) Persona(string) (domain.Persona, error) { return f.persona, f.err }

func (f *fakeConfig) SetPersona(_, name string) ([]domain.Write, error) {
	f.set = name

	return f.writes, f.err
}

// run executes `codefall config` with the arguments against a fake use case and returns everything
// it wrote. CLICOLOR_FORCE and TTY_FORCE are cleared so a forced-colour environment cannot leak ANSI
// into the buffer.
func run(t *testing.T, config ConfigUseCase, args ...string) (string, error) {
	t.Helper()
	t.Setenv("CLICOLOR_FORCE", "0")
	t.Setenv("TTY_FORCE", "0")

	var out bytes.Buffer

	cmd := NewConfigCommand(config)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()

	return out.String(), err
}

var (
	subagent  = settings.Agent{Name: "subagent", Harness: "current", Model: mo.None[string]()}
	architect = settings.Agent{Name: "architect", Harness: "codex", Model: mo.Some("gpt-5-codex")}
)

func TestShowPrintsTheEffectiveConfiguration(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{
		Agents:    []settings.Agent{architect, subagent},
		Review:    mo.Some([]string{"architect", "subagent"}),
		Consult:   mo.None[[]string](),
		ByHarness: map[string][]string{"codex": {"subagent"}, "claude": {"architect"}},
		Persona:   domain.Persona{Name: "product-manager", FromFile: true},
	}}

	out, err := run(t, config, "show")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "agents:\n" +
		"  1. architect (codex, gpt-5-codex)\n" +
		"  2. subagent (current)\n" +
		"review.agents: architect, subagent\n" +
		"agentsByHarness.claude: architect\n" +
		"agentsByHarness.codex: subagent\n" +
		"persona: product-manager (from .codefall/user.json)\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

func TestShowSaysWhenTheListIsTheDefault(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{
		Agents:  []settings.Agent{subagent},
		Default: true,
		Persona: domain.Persona{Name: "engineer"},
	}}

	out, err := run(t, config, "show")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "agents (the default; .codefall/settings.json lists none):\n" +
		"  1. subagent (current)\n" +
		"persona: engineer (default)\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

func TestAgentsListPrintsOneAgentPerLine(t *testing.T) {
	out, err := run(t, &fakeConfig{agents: []settings.Agent{architect, subagent}}, "agents", "list")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "architect (codex, gpt-5-codex)\nsubagent (current)\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// Every flag reaches the use case, and one left out arrives as None rather than as an empty string.
func TestAgentsAddHandsOverTheAgentAndPrintsTheWrite(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("added agent second (muse) to .codefall/settings.json, before architect")}

	out, err := run(t, config, "agents", "add", "second", "--harness", "muse", "--before", "architect")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := application.NewAgent{
		Name: "second", Harness: "muse",
		Model: mo.None[string](), Before: mo.Some("architect"), After: mo.None[string](),
	}
	if config.added != want {
		t.Errorf("added = %+v, want %+v", config.added, want)
	}

	if want := "✓ added agent second (muse) to .codefall/settings.json, before architect\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// A command that never prompts has to be told everything it needs: a missing --harness is refused by
// name, and so is being told two places at once.
func TestAgentsAddRefusesMissingOrConflictingFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "no harness", args: []string{"agents", "add", "second"}, want: `"harness" not set`},
		{
			name: "before and after",
			args: []string{"agents", "add", "second", "--harness", "muse", "--before", "a", "--after", "b"},
			want: "[after before] were all set",
		},
		{name: "no name", args: []string{"agents", "add", "--harness", "muse"}, want: "accepts 1 arg"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &fakeConfig{}

			_, err := run(t, config, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Execute error = %v, want it to say %q", err, tc.want)
			}

			if config.added.Name != "" {
				t.Error("the use case was asked, want the command to stop at the flags")
			}
		})
	}
}

// A refusal is the use case's own sentence, returned as it is so Fang renders it without a prefix.
func TestAgentsRemoveReturnsTheRefusalAsItIs(t *testing.T) {
	refusal := errors.New(`agent "architect" is still named by review.agents; remove it there first`)

	_, err := run(t, &fakeConfig{err: refusal}, "agents", "remove", "architect")
	if !errors.Is(err, refusal) || err.Error() != refusal.Error() {
		t.Errorf("Execute error = %v, want %v as it is", err, refusal)
	}
}

func TestAgentsOrderHandsOverEveryName(t *testing.T) {
	config := &fakeConfig{write: domain.Unchanged(".codefall/settings.json already lists the agents in that order")}

	out, err := run(t, config, "agents", "order", "second", "architect", "subagent")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := strings.Join(config.ordered, " "); got != "second architect subagent" {
		t.Errorf("ordered = %q, want every name in the order given", got)
	}

	if want := "- .codefall/settings.json already lists the agents in that order\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// The order command hands the use case the order its target names and the agents in the order
// given, or asks it to clear the order, and prints the one line that comes back.
func TestOrderCommandHandsOverTheOrder(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		order   application.Order
		names   []string
		cleared bool
	}{
		{
			name: "review", args: []string{"order", "review", "architect", "subagent"},
			order: application.ReviewOrder(), names: []string{"architect", "subagent"},
		},
		{name: "review --clear", args: []string{"order", "review", "--clear"}, order: application.ReviewOrder(), cleared: true},
		{
			name: "consult", args: []string{"order", "consult", "subagent"},
			order: application.ConsultOrder(), names: []string{"subagent"},
		},
		{name: "consult --clear", args: []string{"order", "consult", "--clear"}, order: application.ConsultOrder(), cleared: true},
		{
			name: "a harness", args: []string{"order", "claude", "architect", "subagent"},
			order: application.HarnessOrder("claude"), names: []string{"architect", "subagent"},
		},
		{
			name: "a harness --clear", args: []string{"order", "claude", "--clear"},
			order: application.HarnessOrder("claude"), cleared: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &fakeConfig{write: domain.Changed("set the order")}

			out, err := run(t, config, tc.args...)
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if got, ok := config.order.Get(); !ok || got != tc.order {
				t.Errorf("order = %+v, want %+v", config.order, tc.order)
			}

			if strings.Join(config.names, " ") != strings.Join(tc.names, " ") || config.cleared != tc.cleared {
				t.Errorf("names = %q, cleared = %v, want %q, %v", config.names, config.cleared, tc.names, tc.cleared)
			}

			if want := "✓ set the order\n"; out != want {
				t.Errorf("output = %q, want %q", out, want)
			}
		})
	}
}

// The order command is told a target first, then either the agents or --clear, never both and never
// neither. A refusal here never reaches the use case.
func TestOrderCommandRefusesTheWrongArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "review with nothing", args: []string{"order", "review"}, want: "name at least one agent, or pass --clear"},
		{name: "consult with both", args: []string{"order", "consult", "subagent", "--clear"}, want: "--clear takes no agent names"},
		{name: "no target", args: []string{"order"}, want: "name what the order is for first"},
		{name: "a harness with no agents", args: []string{"order", "claude"}, want: "name at least one agent, or pass --clear"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &fakeConfig{}

			_, err := run(t, config, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Execute error = %v, want it to say %q", err, tc.want)
			}

			if config.order.IsPresent() {
				t.Error("the use case was asked, want the command to stop at the arguments")
			}
		})
	}
}

func TestPersonaPrintsTheValueAndWhereItCameFrom(t *testing.T) {
	out, err := run(t, &fakeConfig{persona: domain.Persona{Name: "engineer"}}, "persona")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "engineer (default)\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestPersonaWithAValueWritesItAndPrintsEachWrite(t *testing.T) {
	config := &fakeConfig{writes: []domain.Write{
		domain.Changed("wrote .codefall/user.json (persona: product-manager)"),
		domain.Changed("added .codefall/user.json to .gitignore"),
	}}

	out, err := run(t, config, "persona", "product-manager")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if config.set != "product-manager" {
		t.Errorf("set = %q, want product-manager", config.set)
	}

	want := "✓ wrote .codefall/user.json (persona: product-manager)\n✓ added .codefall/user.json to .gitignore\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// The bare command with no terminal to draw on prints what show prints and says where the editor
// is, so a script or a pipe gets the facts and a person learns the way in.
func TestBareCommandWithoutATerminalPrintsTheConfigurationAndAHint(t *testing.T) {
	stdoutIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdoutIsTerminal = func() bool { return false } })

	config := &fakeConfig{shown: domain.Configuration{
		Agents:  []settings.Agent{subagent},
		Persona: domain.Persona{Name: "engineer"},
	}}

	out, err := run(t, config)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, want := range []string{"agents:", "1. subagent (current)", "persona: engineer (default)", "Run codefall config in a terminal"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to hold %q", out, want)
		}
	}
}
