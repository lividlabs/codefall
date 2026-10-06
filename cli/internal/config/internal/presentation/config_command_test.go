package presentation

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

type fakeConfig struct {
	shown   domain.Configuration
	entries []settings.Entry
	persona domain.Persona
	write   domain.Write
	writes  []domain.Write
	err     error

	configs map[string]settings.HarnessConfig

	// What the last write was asked for.
	active, feature string
	agents          []settings.Agent
	cleared         bool
	posting         mo.Option[bool]
	prefix          string
	set             string
	key             string
	block           mo.Option[settings.HarnessConfig]
	blockCleared    bool
}

func (f *fakeConfig) HarnessConfigs(string) (map[string]settings.HarnessConfig, error) {
	return f.configs, f.err
}

func (f *fakeConfig) SetHarnessConfig(_, key string, block settings.HarnessConfig) (domain.Write, error) {
	f.key, f.block = key, mo.Some(block)

	return f.write, f.err
}

func (f *fakeConfig) ClearHarnessConfig(_, key string) (domain.Write, error) {
	f.key, f.blockCleared = key, true

	return f.write, f.err
}

func (f *fakeConfig) Show(string) (domain.Configuration, error) { return f.shown, f.err }

func (f *fakeConfig) Agents(string) ([]settings.Entry, error) { return f.entries, f.err }

func (f *fakeConfig) SetList(_, active, feature string, agents []settings.Agent) (domain.Write, error) {
	f.active, f.feature, f.agents = active, feature, agents

	return f.write, f.err
}

func (f *fakeConfig) ClearList(_, active, feature string) (domain.Write, error) {
	f.active, f.feature, f.cleared = active, feature, true

	return f.write, f.err
}

func (f *fakeConfig) SetPosting(_ string, on bool) (domain.Write, error) {
	f.posting = mo.Some(on)

	return f.write, f.err
}

func (f *fakeConfig) SetSkillPrefix(_, prefix string) (domain.Write, error) {
	f.prefix = prefix

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
	current = settings.Agent{Harness: "current", Model: mo.None[string]()}
	claude  = settings.Agent{Harness: "claude", Model: mo.None[string]()}
	codex   = settings.Agent{Harness: "codex", Model: mo.Some("gpt-5-codex")}

	defaultEntry = settings.Entry{
		ActiveAgent: "default",
		Review:      mo.Some([]settings.Agent{current}),
		Consult:     mo.Some([]settings.Agent{current}),
	}
	museEntry = settings.Entry{
		ActiveAgent: "muse",
		Review:      mo.Some([]settings.Agent{claude, codex}),
		Consult:     mo.None[[]settings.Agent](),
	}
)

func TestShowPrintsTheEffectiveConfiguration(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{
		Entries: []settings.Entry{defaultEntry, museEntry},
		Posting: true,
		Persona: domain.Persona{Name: "product-manager", FromFile: true},
	}}

	out, err := run(t, config, "show")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "agents:\n" +
		"  default (any harness without its own entry):\n" +
		"    review: this harness\n" +
		"    consult: this harness\n" +
		"  when running in muse:\n" +
		"    review: claude, then codex:gpt-5-codex\n" +
		"    consult: same as default\n" +
		"review posting: on\n" +
		"skill prefix: codefall (codefall-design, /codefall-implement)\n" +
		"persona: product-manager (from .codefall/user.json)\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

func TestShowSaysWhenTheListIsTheDefault(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{
		Entries: []settings.Entry{defaultEntry},
		Default: true,
		Persona: domain.Persona{Name: "engineer"},
	}}

	out, err := run(t, config, "show")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "agents (the built-in default; .codefall/settings.json sets none):\n" +
		"  default (any harness without its own entry):\n" +
		"    review: this harness\n" +
		"    consult: this harness\n" +
		"review posting: off\n" +
		"skill prefix: codefall (codefall-design, /codefall-implement)\n" +
		"persona: engineer (default)\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// `agents` alone and `agents list` print the same thing.
func TestAgentsPrintsTheEntries(t *testing.T) {
	config := &fakeConfig{entries: []settings.Entry{museEntry}}

	want := "  when running in muse:\n    review: claude, then codex:gpt-5-codex\n    consult: same as default\n"

	for _, args := range [][]string{{"agents"}, {"agents", "list"}} {
		out, err := run(t, config, args...)
		if err != nil {
			t.Fatalf("Execute %q: %v", args, err)
		}

		if out != want {
			t.Errorf("output of %q = %q, want %q", args, out, want)
		}
	}
}

// Every agent reaches the use case parsed, in the order given, with a model when one was typed.
func TestAgentsSetsAListAndPrintsTheWrite(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("set muse review in .codefall/settings.json: claude, codex:gpt-5-codex")}

	out, err := run(t, config, "agents", "muse", "review", "claude", "codex:gpt-5-codex")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if config.active != "muse" || config.feature != "review" || !slices.Equal(config.agents, []settings.Agent{claude, codex}) {
		t.Errorf("SetList was given %q %q %+v, want muse review claude, codex:gpt-5-codex", config.active, config.feature, config.agents)
	}

	if want := "✓ set muse review in .codefall/settings.json: claude, codex:gpt-5-codex\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestAgentsClearsAList(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("removed muse consult from .codefall/settings.json")}

	if _, err := run(t, config, "agents", "muse", "consult", "--clear"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !config.cleared || config.active != "muse" || config.feature != "consult" {
		t.Errorf("ClearList was given %q %q, cleared = %v; want muse consult cleared", config.active, config.feature, config.cleared)
	}
}

// The command refuses what it can see is wrong before the use case is reached, and hands a bad agent
// back in the settings module's words.
func TestAgentsRefusesTheWrongArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "an active agent alone", args: []string{"agents", "muse"}, want: "name the list too"},
		{name: "no agents and no --clear", args: []string{"agents", "muse", "review"}, want: "name at least one agent"},
		{name: "agents with --clear", args: []string{"agents", "muse", "review", "claude", "--clear"}, want: "--clear takes no agents"},
		{name: "a harness written no way codefall reads", args: []string{"agents", "muse", "review", "Cursor"}, want: `agent "Cursor": harness "Cursor" is not a harness`},
		{name: "an empty model", args: []string{"agents", "muse", "review", "codex:"}, want: "names an empty model"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &fakeConfig{}

			_, err := run(t, config, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to say %q", err, tc.want)
			}

			if config.agents != nil || config.cleared {
				t.Error("the use case was reached with arguments the command should have refused")
			}
		})
	}
}

// A refusal from the use case is returned as it is, so the settings module's words reach the person.
func TestAgentsReturnsTheRefusalAsItIs(t *testing.T) {
	refusal := errors.New(`.codefall/settings.json is not valid, so nothing was changed: tracker: missing`)

	_, err := run(t, &fakeConfig{err: refusal}, "agents", "default", "review", "current")
	if !errors.Is(err, refusal) {
		t.Errorf("error = %v, want %v", err, refusal)
	}
}

// harnessBlocks is Codex called through a provider, and a variant that calls it directly.
var harnessBlocks = map[string]settings.HarnessConfig{
	"codex": {
		Harness:   "codex",
		ModelFlag: mo.Some("--model"),
		Provider:  mo.Some("amazon-bedrock-runtime"),
		Env:       mo.Some("aws configure export-credentials --format env"),
		Args:      []string{"-c", "model_reasoning_effort=high"},
	},
	"codex-direct": {
		Harness: "codex", ModelFlag: mo.None[string](), Provider: mo.None[string](), Env: mo.None[string](),
		Args: []string{"-c", "model_reasoning_effort=high"},
	},
}

// The blocks print after the agents, in key order, one field per line, and a block under a harness
// name does not repeat the harness.
const harnessBlockLines = "harnessConfig.codex:\n" +
	"  modelFlag: --model\n" +
	"  provider: amazon-bedrock-runtime\n" +
	"  args: -c, model_reasoning_effort=high\n" +
	"  env: aws configure export-credentials --format env\n" +
	"harnessConfig.codex-direct:\n" +
	"  harness: codex\n" +
	"  args: -c, model_reasoning_effort=high\n"

func TestShowPrintsHowEachHarnessIsCalled(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{
		Entries:        []settings.Entry{defaultEntry},
		HarnessConfigs: harnessBlocks,
		Persona:        domain.Persona{Name: "engineer"},
	}}

	out, err := run(t, config, "show")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := "agents:\n" +
		"  default (any harness without its own entry):\n" +
		"    review: this harness\n" +
		"    consult: this harness\n" +
		harnessBlockLines +
		"review posting: off\n" +
		"skill prefix: codefall (codefall-design, /codefall-implement)\n" +
		"persona: engineer (default)\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
}

// `harness` alone and `harness list` print the same thing, and say so when there is nothing.
func TestHarnessPrintsTheBlocks(t *testing.T) {
	config := &fakeConfig{configs: harnessBlocks}

	for _, args := range [][]string{{"harness"}, {"harness", "list"}} {
		out, err := run(t, config, args...)
		if err != nil {
			t.Fatalf("Execute %q: %v", args, err)
		}

		if out != harnessBlockLines {
			t.Errorf("output of %q = %q, want %q", args, out, harnessBlockLines)
		}
	}

	out, err := run(t, &fakeConfig{}, "harness")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "harnessConfig: none set in .codefall/settings.json; every harness runs bare\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

// Every flag reaches the use case as the field it stands for, and a flag not given is absent.
func TestHarnessSetsABlockAndPrintsTheWrite(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("set harnessConfig.codex-direct in .codefall/settings.json")}

	out, err := run(t, config, "harness", "codex-direct", "--harness", "codex",
		"--arg", "-c", "--arg", "model_reasoning_effort=high", "--env", "aws configure export-credentials --format env")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	want := settings.HarnessConfig{
		Harness:   "codex",
		ModelFlag: mo.None[string](),
		Provider:  mo.None[string](),
		Env:       mo.Some("aws configure export-credentials --format env"),
		Args:      []string{"-c", "model_reasoning_effort=high"},
	}

	if block, ok := config.block.Get(); config.key != "codex-direct" || !ok || block.Harness != want.Harness ||
		block.ModelFlag != want.ModelFlag || block.Provider != want.Provider || block.Env != want.Env ||
		!slices.Equal(block.Args, want.Args) {
		t.Errorf("SetHarnessConfig was given %q %+v, want codex-direct %+v", config.key, config.block, want)
	}

	if want := "✓ set harnessConfig.codex-direct in .codefall/settings.json\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	config = &fakeConfig{write: domain.Changed("set harnessConfig.codex in .codefall/settings.json")}

	if _, err := run(t, config, "harness", "codex", "--model-flag", "--model", "--provider", "amazon-bedrock-runtime"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if block, ok := config.block.Get(); !ok || block.Harness != "" || block.ModelFlag != mo.Some("--model") ||
		block.Provider != mo.Some("amazon-bedrock-runtime") || block.Env.IsPresent() || block.Args != nil {
		t.Errorf("SetHarnessConfig was given %+v, want only the model flag and the provider", config.block)
	}
}

func TestHarnessClearsABlock(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("removed harnessConfig.codex from .codefall/settings.json")}

	if _, err := run(t, config, "harness", "codex", "--clear"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !config.blockCleared || config.key != "codex" {
		t.Errorf("ClearHarnessConfig was given %q, cleared = %v; want codex cleared", config.key, config.blockCleared)
	}
}

// The command refuses what it can see is wrong before the use case is reached, naming the flags.
func TestHarnessRefusesTheWrongArguments(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "a key and no flags", args: []string{"harness", "codex"}, want: "pass at least one of --harness, --model-flag, --provider, --arg, or --env, or --clear"},
		{name: "flags and no key", args: []string{"harness", "--provider", "x"}, want: "name the key the flags are for"},
		{name: "--clear and no key", args: []string{"harness", "--clear"}, want: "name the key the flags are for"},
		{name: "two keys", args: []string{"harness", "codex", "muse", "--provider", "x"}, want: "name one key"},
		{name: "--clear with a flag", args: []string{"harness", "codex", "--clear", "--provider", "x"}, want: "--clear takes no other flags"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := &fakeConfig{}

			_, err := run(t, config, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to say %q", err, tc.want)
			}

			if config.block.IsPresent() || config.blockCleared {
				t.Error("the use case was reached with arguments the command should have refused")
			}
		})
	}
}

// A refusal from the use case is returned as it is, so the settings module's words reach the person.
func TestHarnessReturnsTheRefusalAsItIs(t *testing.T) {
	refusal := errors.New(`that change would leave .codefall/settings.json invalid, so nothing was changed: harnessConfig.bedrock.harness: missing`)

	_, err := run(t, &fakeConfig{err: refusal}, "harness", "bedrock", "--provider", "amazon-bedrock-runtime")
	if !errors.Is(err, refusal) {
		t.Errorf("error = %v, want %v", err, refusal)
	}
}

// A variant key is an agent like any harness name; the file decides whether it is there.
func TestAgentsAcceptsAVariantKey(t *testing.T) {
	config := &fakeConfig{write: domain.Changed("set claude consult in .codefall/settings.json: codex-direct:gpt-6-astra")}

	if _, err := run(t, config, "agents", "claude", "consult", "codex-direct:gpt-6-astra"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := []settings.Agent{{Harness: "codex-direct", Model: mo.Some("gpt-6-astra")}}; !slices.Equal(config.agents, want) {
		t.Errorf("SetList was given %+v, want %+v", config.agents, want)
	}
}

func TestReviewPostingHandsOverOnOrOff(t *testing.T) {
	for _, tc := range []struct {
		arg  string
		want bool
	}{{"on", true}, {"off", false}} {
		config := &fakeConfig{write: domain.Changed("set posting " + tc.arg + " in .codefall/settings.json")}

		out, err := run(t, config, "review", "posting", tc.arg)
		if err != nil {
			t.Fatalf("Execute %s: %v", tc.arg, err)
		}

		if got, ok := config.posting.Get(); !ok || got != tc.want {
			t.Errorf("SetPosting was given %v, want %v", config.posting, tc.want)
		}

		if want := "✓ set posting " + tc.arg + " in .codefall/settings.json\n"; out != want {
			t.Errorf("output = %q, want %q", out, want)
		}
	}

	if _, err := run(t, &fakeConfig{}, "review", "posting", "maybe"); err == nil {
		t.Error("posting maybe was accepted, want a refusal")
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
		Entries: []settings.Entry{defaultEntry},
		Persona: domain.Persona{Name: "engineer"},
	}}

	out, err := run(t, config)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, want := range []string{"agents:", "review: this harness", "persona: engineer (default)", "Run codefall config in a terminal"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to hold %q", out, want)
		}
	}
}

// The prefix prints with no argument and is handed over with one; the sentence the write reports is
// the use case's, which names the command that renames the install.
func TestSkillPrefixPrintsOrSets(t *testing.T) {
	config := &fakeConfig{shown: domain.Configuration{SkillPrefix: settings.SkillPrefixCf, Persona: domain.Persona{Name: "engineer"}}}

	out, err := run(t, config, "skill-prefix")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "skill prefix: cf (cf-design, /cf-implement)\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	config = &fakeConfig{write: domain.Changed("set skill prefix cfall in .codefall/settings.json; run codefall upgrade to rename the installed skills")}

	out, err = run(t, config, "skill-prefix", "cfall")
	if err != nil {
		t.Fatalf("Execute cfall: %v", err)
	}

	if config.prefix != settings.SkillPrefixCfall {
		t.Errorf("SetSkillPrefix was given %q, want %q", config.prefix, settings.SkillPrefixCfall)
	}

	if want := "✓ set skill prefix cfall in .codefall/settings.json; run codefall upgrade to rename the installed skills\n"; out != want {
		t.Errorf("output = %q, want %q", out, want)
	}

	if _, err := run(t, &fakeConfig{}, "skill-prefix", "cf", "cfall"); err == nil {
		t.Error("two prefixes were accepted, want a refusal")
	}
}
