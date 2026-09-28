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
