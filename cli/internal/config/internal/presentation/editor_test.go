package presentation

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

func press(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text, Mod: mod}
}

var (
	enter  = press(tea.KeyEnter, "", 0)
	escape = press(tea.KeyEscape, "", 0)
	down   = press(tea.KeyDown, "", 0)
)

// twoAgents is the configuration every editor test starts from: two agents, no narrower orders, the
// default persona.
func twoAgents() domain.Configuration {
	return domain.Configuration{
		Agents:  []settings.Agent{subagent, architect},
		Persona: domain.Persona{Name: "engineer"},
	}
}

func newTestEditor(t *testing.T, config *fakeConfig) editor {
	t.Helper()

	e, err := newEditor(config, "/project")
	if err != nil {
		t.Fatalf("newEditor: %v", err)
	}

	return e
}

// send feeds messages to the editor one at a time, running every command each update returns and
// feeding what those produce back in, the way the program would. That is what lets a Huh form move
// from field to field in a test: its keys return commands rather than changing the form directly.
func send(t *testing.T, e editor, msgs ...tea.Msg) editor {
	t.Helper()

	var model tea.Model = e

	for _, msg := range msgs {
		model = drain(t, model, msg)
	}

	finished, ok := model.(editor)
	if !ok {
		t.Fatalf("the editor became a %T", model)
	}

	return finished
}

// drain updates the model with one message and then with everything the returned command produces,
// until nothing is left to run. A quit is left where it is, since the program would stop there. A
// cursor blink or a tick is not followed: each would produce another after a delay for as long as the
// program ran, and a test wants the form's state, not its animation.
func drain(t *testing.T, model tea.Model, msg tea.Msg) tea.Model {
	t.Helper()

	model, cmd := model.Update(msg)

	return drainCmd(t, model, cmd, 0)
}

// drainCmd runs one command and feeds what it produces back into the model, following batches and
// the commands each update returns in turn. The depth bound is a guard against a command that keeps
// producing more; nothing the editor does needs more than a handful of rounds.
func drainCmd(t *testing.T, model tea.Model, cmd tea.Cmd, depth int) tea.Model {
	t.Helper()

	if cmd == nil || depth > 32 {
		return model
	}

	produced := cmd()
	if produced == nil || isAnimation(produced) {
		return model
	}

	switch produced := produced.(type) {
	case tea.QuitMsg:
		return model
	case tea.BatchMsg:
		for _, each := range produced {
			model = drainCmd(t, model, each, depth+1)
		}

		return model
	default:
		next, cmd := model.Update(produced)

		return drainCmd(t, next, cmd, depth+1)
	}
}

// isAnimation reports whether a message is a cursor blink or a tick: something a running program
// would keep receiving on a timer, and a test has no use for.
func isAnimation(msg tea.Msg) bool {
	name := strings.ToLower(reflect.TypeOf(msg).String())

	return strings.Contains(name, "blink") || strings.Contains(name, "tick")
}

// quits reports whether the editor's response to a message is the program stopping.
func quits(e editor, msg tea.Msg) bool {
	_, cmd := e.Update(msg)
	if cmd == nil {
		return false
	}

	_, ok := cmd().(tea.QuitMsg)

	return ok
}

func TestEditorRefusesADirectoryItCannotRead(t *testing.T) {
	failure := errors.New(".codefall/settings.json not found; run codefall init first")

	if _, err := newEditor(&fakeConfig{err: failure}, "/project"); !errors.Is(err, failure) {
		t.Fatalf("newEditor error = %v, want %v", err, failure)
	}
}

func TestEditorOpensEachSectionAndComesBack(t *testing.T) {
	config := &fakeConfig{shown: twoAgents()}
	config.shown.ByHarness = map[string][]string{"claude": {"architect"}}

	e := newTestEditor(t, config)

	// Agents is the first item.
	e = send(t, e, enter)
	if e.screen != screenAgents {
		t.Fatalf("after enter on Agents: screen = %v, want agents", e.screen)
	}

	if got := e.agents.IDs(); !slices.Equal(got, []string{"subagent", "architect"}) {
		t.Errorf("the agents screen lists %q, want the list in its order", got)
	}

	e = send(t, e, escape)
	if e.screen != screenMenu {
		t.Fatalf("after esc: screen = %v, want the menu", e.screen)
	}

	// Review order is the second.
	e = send(t, e, down, enter)
	if e.screen != screenOrder || e.orderPath != "review.agents" {
		t.Fatalf("after enter on Review order: screen = %v, path = %q", e.screen, e.orderPath)
	}

	if got := e.order.IDs(); len(got) != 0 {
		t.Errorf("an order that is not set has %q switched on, want none", got)
	}

	e = send(t, e, escape)

	// Per-harness orders is the fourth; agy is the first harness, claude the second.
	e = send(t, e, down, down, enter)
	if e.screen != screenHarness {
		t.Fatalf("after enter on Per-harness orders: screen = %v, want the harness list", e.screen)
	}

	e = send(t, e, down, enter)
	if e.screen != screenOrder || e.orderPath != "agentsByHarness.claude" {
		t.Fatalf("after enter on claude: screen = %v, path = %q", e.screen, e.orderPath)
	}

	if got := e.order.IDs(); !slices.Equal(got, []string{"architect"}) {
		t.Errorf("claude's order shows %q switched on, want architect", got)
	}

	e = send(t, e, escape)
	if e.screen != screenHarness {
		t.Fatalf("esc from a harness's order: screen = %v, want the harness list", e.screen)
	}

	e = send(t, e, escape)
	if e.screen != screenMenu {
		t.Fatalf("esc from the harness list: screen = %v, want the menu", e.screen)
	}

	// Persona is the fifth.
	e = send(t, e, down, enter)
	if e.screen != screenPersona || e.form == nil {
		t.Fatalf("after enter on Persona: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	e = send(t, e, escape)
	if e.screen != screenMenu || e.form != nil {
		t.Fatalf("esc from the persona form: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	if len(e.writes) != 0 {
		t.Errorf("walking the sections wrote %d things, want none", len(e.writes))
	}
}

func TestEditorQuitsFromTheMenuWithoutWriting(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoAgents()})

	if !quits(e, press('q', "q", 0)) {
		t.Error("q on the menu did not quit")
	}

	if !quits(e, press('c', "", tea.ModCtrl)) {
		t.Error("ctrl+c on the menu did not quit")
	}

	// Quit is the last item.
	e = send(t, e, down, down, down, down, down)
	if !quits(e, enter) {
		t.Error("enter on Quit did not quit")
	}

	if len(e.writes) != 0 {
		t.Errorf("quitting wrote %d things, want none", len(e.writes))
	}
}

func TestEditorReordersTheAgentsAndSavesOnEnter(t *testing.T) {
	config := &fakeConfig{shown: twoAgents(), write: domain.Changed("ordered the agents in .codefall/settings.json: architect, subagent")}
	e := newTestEditor(t, config)

	e = send(t, e, enter, press(tea.KeyDown, "", tea.ModShift))
	if got := e.agents.IDs(); !slices.Equal(got, []string{"architect", "subagent"}) {
		t.Fatalf("after moving subagent down: %q", got)
	}

	if config.ordered != nil {
		t.Fatal("moving an entry wrote before enter")
	}

	e = send(t, e, enter)

	if !slices.Equal(config.ordered, []string{"architect", "subagent"}) {
		t.Errorf("OrderAgents was given %q, want architect, subagent", config.ordered)
	}

	if e.screen != screenAgents || len(e.writes) != 1 || !strings.Contains(e.status, "ordered the agents") {
		t.Errorf("after saving: screen = %v, writes = %d, status = %q", e.screen, len(e.writes), e.status)
	}
}

func TestEditorSavesAnOrderFromTheSwitchedOnAgentsAndClearsWhenNoneAre(t *testing.T) {
	config := &fakeConfig{shown: twoAgents(), write: domain.Changed("set review.agents")}
	e := newTestEditor(t, config)

	// Review order: switch architect on, then subagent, so the order reads architect, subagent.
	e = send(t, e, down, enter, down, press(tea.KeySpace, " ", 0), press(tea.KeyUp, "", 0), press(tea.KeySpace, " ", 0))

	if got := e.order.IDs(); !slices.Equal(got, []string{"subagent", "architect"}) {
		t.Fatalf("switched on in the list's order: %q", got)
	}

	e = send(t, e, press(tea.KeyDown, "", tea.ModShift), enter)

	if got, ok := config.order.Get(); !ok || got.Harness().IsPresent() || !slices.Equal(config.names, []string{"architect", "subagent"}) {
		t.Errorf("SetOrder was given %+v with %q, want review's order with architect, subagent", config.order, config.names)
	}

	if e.screen != screenMenu {
		t.Errorf("after saving review's order: screen = %v, want the menu", e.screen)
	}

	// Nothing switched on clears the order instead.
	config.cleared = false
	e = send(t, e, down, enter, enter)

	if !config.cleared {
		t.Error("saving an order with nothing switched on did not clear it")
	}
}

func TestEditorShowsARefusalInlineAndStays(t *testing.T) {
	config := &fakeConfig{shown: twoAgents()}
	e := send(t, newTestEditor(t, config), enter)

	// The fake answers every call with one error, so it is set once the editor has read the
	// configuration and is on the agents screen.
	config.err = domain.StillNamed("subagent", []string{"review.agents"})

	e = send(t, e, press('d', "d", 0))

	if config.removed != "subagent" {
		t.Errorf("RemoveAgent was asked for %q, want the agent under the cursor", config.removed)
	}

	want := "subagent is named in review.agents. Change that order first, in codefall config, then remove it"
	if e.status != want {
		t.Errorf("status = %q, want the refusal %q", e.status, want)
	}

	if e.screen != screenAgents || len(e.writes) != 0 {
		t.Errorf("after a refusal: screen = %v, writes = %d, want the agents screen and nothing written", e.screen, len(e.writes))
	}
}

func TestEditorAddsAnAgentThroughTheForm(t *testing.T) {
	config := &fakeConfig{shown: twoAgents(), write: domain.Changed("added agent reviewer (muse) to .codefall/settings.json")}
	e := newTestEditor(t, config)

	e = send(t, e, enter, press('a', "a", 0))
	if e.screen != screenAdd || e.form == nil {
		t.Fatalf("after a: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	// The name, then the harness (the default, current, is fine), then no model.
	e = send(t, e, press('r', "r", 0), press('e', "e", 0), press('v', "v", 0), enter, enter, enter)

	if config.added.Name != "rev" || config.added.Harness != settings.HarnessCurrent || config.added.Model.IsPresent() {
		t.Errorf("AddAgent was given %+v, want rev on current with no model", config.added)
	}

	if e.screen != screenAgents || e.form != nil || len(e.writes) != 1 {
		t.Errorf("after adding: screen = %v, form nil = %v, writes = %d", e.screen, e.form == nil, len(e.writes))
	}
}

func TestEditorRefusesAnEmptyNameInTheForm(t *testing.T) {
	config := &fakeConfig{shown: twoAgents()}
	e := newTestEditor(t, config)

	e = send(t, e, enter, press('a', "a", 0), enter)

	if e.screen != screenAdd {
		t.Errorf("enter on an empty name left the form: screen = %v", e.screen)
	}

	if config.added.Name != "" {
		t.Errorf("AddAgent was asked for %q, want the form to hold the person at the name", config.added.Name)
	}
}

func TestEditorSetsThePersona(t *testing.T) {
	config := &fakeConfig{shown: twoAgents(), writes: []domain.Write{domain.Changed("set persona to engineer in .codefall/user.json")}}
	e := newTestEditor(t, config)

	e = send(t, e, down, down, down, down, enter, enter)

	if config.set != "engineer" {
		t.Errorf("SetPersona was given %q, want the option under the cursor, engineer", config.set)
	}

	if e.screen != screenMenu || len(e.writes) != 1 {
		t.Errorf("after setting the persona: screen = %v, writes = %d", e.screen, len(e.writes))
	}
}

func TestEditorViewNamesTheScreenAndItsKeys(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoAgents()})

	if view := e.View().Content; !strings.Contains(view, "codefall config") || !strings.Contains(view, "q quit") {
		t.Errorf("menu view = %q, want the heading and the quit key", view)
	}

	e = send(t, e, enter)
	if view := e.View().Content; !strings.Contains(view, "a add") || !strings.Contains(view, "enter save order") {
		t.Errorf("agents view = %q, want its keys", view)
	}
}

// The consult section's current order reaches the order screen the same way review's does.
func TestEditorShowsTheConsultOrderSwitchedOn(t *testing.T) {
	config := &fakeConfig{shown: twoAgents()}
	config.shown.Consult = mo.Some([]string{"architect"})

	e := send(t, newTestEditor(t, config), down, down, enter)

	if e.orderPath != "consult.agents" || !slices.Equal(e.order.IDs(), []string{"architect"}) {
		t.Errorf("consult order: path = %q, on = %q", e.orderPath, e.order.IDs())
	}
}
