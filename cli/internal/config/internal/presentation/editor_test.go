package presentation

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

func press(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text, Mod: mod}
}

var (
	enter  = press(tea.KeyEnter, "", 0)
	escape = press(tea.KeyEscape, "", 0)
	down   = press(tea.KeyDown, "", 0)
)

// twoEntries is the configuration every editor test starts from: the default entry and one for Muse
// whose review list reaches for Claude then Codex, posting off, the default persona.
func twoEntries() domain.Configuration {
	return domain.Configuration{
		Entries: []settings.Entry{defaultEntry, museEntry},
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

func TestEditorOpensEachScreenAndComesBack(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoEntries()})

	// Agents is the first item: the active agents, default first.
	e = send(t, e, enter)
	if e.screen != screenEntries {
		t.Fatalf("after enter on Agents: screen = %v, want the entries", e.screen)
	}

	first, ok := e.entries.SelectedItem().(row)
	if !ok || first.value != settings.ActiveDefault {
		t.Fatalf("the first active agent is %+v, want default", first)
	}

	// The fourth row is muse (agy, claude, codex come before it), and it has an entry.
	e = send(t, e, down, down, down, down, enter)
	if e.screen != screenEntry || e.active != "muse" {
		t.Fatalf("after enter on muse: screen = %v, active = %q", e.screen, e.active)
	}

	// Its review list opens with Claude then Codex.
	e = send(t, e, enter)
	if e.screen != screenList || e.feature != settings.FeatureReview {
		t.Fatalf("after enter on Review: screen = %v, feature = %q", e.screen, e.feature)
	}

	if got := e.orderedWorking(); !slices.Equal(got, []settings.Agent{claude, codex}) {
		t.Errorf("the review list shows %+v, want claude then codex", got)
	}

	// Esc walks back one step at a time.
	for _, want := range []screen{screenEntry, screenEntries, screenMenu} {
		e = send(t, e, escape)
		if e.screen != want {
			t.Fatalf("after esc: screen = %v, want %v", e.screen, want)
		}
	}

	// Reviews is the second item and opens the posting form.
	e = send(t, e, down, enter)
	if e.screen != screenReviews || e.form == nil {
		t.Fatalf("after enter on Reviews: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	e = send(t, e, escape)

	// Persona is the third.
	e = send(t, e, down, enter)
	if e.screen != screenPersona || e.form == nil {
		t.Fatalf("after enter on Persona: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	e = send(t, e, escape)
	if e.screen != screenMenu || e.form != nil {
		t.Fatalf("esc from the persona form: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	if len(e.writes) != 0 {
		t.Errorf("walking the screens wrote %d things, want none", len(e.writes))
	}
}

// A list the entry leaves to the default opens showing what a run would use, the default's list, so
// the person edits from there rather than from nothing.
func TestEditorOpensAnAbsentListFromTheDefault(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoEntries()})

	// muse, then its consult list, which it leaves to the default.
	e = send(t, e, enter, down, down, down, down, enter, down, enter)
	if e.screen != screenList || e.feature != settings.FeatureConsult {
		t.Fatalf("screen = %v, feature = %q, want muse's consult list", e.screen, e.feature)
	}

	if got := e.orderedWorking(); !slices.Equal(got, []settings.Agent{current}) {
		t.Errorf("the consult list shows %+v, want the default's, current", got)
	}
}

func TestEditorQuitsFromTheMenuWithoutWriting(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoEntries()})

	if !quits(e, press('q', "q", 0)) {
		t.Error("q on the menu did not quit")
	}

	if !quits(e, press('c', "", tea.ModCtrl)) {
		t.Error("ctrl+c on the menu did not quit")
	}

	// Quit is the last item, after Agents, Reviews, Persona, and Skills.
	e = send(t, e, down, down, down, down)
	if !quits(e, enter) {
		t.Error("enter on Quit did not quit")
	}

	if len(e.writes) != 0 {
		t.Errorf("quitting wrote %d things, want none", len(e.writes))
	}
}

func TestEditorReordersAListAndSavesOnEnter(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), write: domain.Changed("set muse review in .codefall/settings.json: codex:gpt-5-codex, claude")}
	e := newTestEditor(t, config)

	e = send(t, e, enter, down, down, down, down, enter, enter, press(tea.KeyDown, "", tea.ModShift))
	if got := e.orderedWorking(); !slices.Equal(got, []settings.Agent{codex, claude}) {
		t.Fatalf("after moving claude down: %+v", got)
	}

	if config.agents != nil {
		t.Fatal("moving an agent wrote before enter")
	}

	e = send(t, e, enter)

	if config.active != "muse" || config.feature != "review" || !slices.Equal(config.agents, []settings.Agent{codex, claude}) {
		t.Errorf("SetList was given %q %q %+v, want muse review codex, claude", config.active, config.feature, config.agents)
	}

	if e.screen != screenEntry || len(e.writes) != 1 || !strings.Contains(e.status, "set muse review") {
		t.Errorf("after saving: screen = %v, writes = %d, status = %q", e.screen, len(e.writes), e.status)
	}
}

func TestEditorRemovesAnAgentAndSavesAnEmptyListAsAClear(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), write: domain.Changed("removed muse review from .codefall/settings.json")}
	e := newTestEditor(t, config)

	e = send(t, e, enter, down, down, down, down, enter, enter, press('d', "d", 0), press('d', "d", 0))
	if got := e.orderedWorking(); len(got) != 0 {
		t.Fatalf("after removing both agents: %+v, want none", got)
	}

	send(t, e, enter)

	if !config.cleared || config.active != "muse" || config.feature != "review" {
		t.Errorf("saving an empty list did not clear muse review: cleared = %v, %q %q", config.cleared, config.active, config.feature)
	}
}

func TestEditorClearsAListWithC(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), write: domain.Changed("removed muse review from .codefall/settings.json")}
	e := newTestEditor(t, config)

	e = send(t, e, enter, down, down, down, down, enter, enter, press('c', "c", 0))

	if !config.cleared || config.active != "muse" || config.feature != "review" {
		t.Errorf("c did not clear muse review: cleared = %v, %q %q", config.cleared, config.active, config.feature)
	}

	if e.screen != screenEntry || len(e.writes) != 1 {
		t.Errorf("after clearing: screen = %v, writes = %d", e.screen, len(e.writes))
	}
}

func TestEditorShowsARefusalInlineAndStays(t *testing.T) {
	config := &fakeConfig{shown: twoEntries()}
	e := send(t, newTestEditor(t, config), enter, enter, enter)

	// The fake answers every call with one error, so it is set once the editor is on the list.
	config.err = errors.New("that change would leave .codefall/settings.json invalid, so nothing was changed")

	e = send(t, e, enter)

	if e.status != config.err.Error() {
		t.Errorf("status = %q, want the refusal", e.status)
	}

	if e.screen != screenEntry || len(e.writes) != 0 {
		t.Errorf("after a refusal: screen = %v, writes = %d, want the entry screen and nothing written", e.screen, len(e.writes))
	}
}

// An added agent joins the list on screen and is written with the rest on enter, not on its own.
func TestEditorAddsAnAgentThroughTheForm(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), write: domain.Changed("set default review in .codefall/settings.json: current, codex")}
	e := newTestEditor(t, config)

	// default, review, add.
	e = send(t, e, enter, enter, enter, press('a', "a", 0))
	if e.screen != screenAdd || e.form == nil {
		t.Fatalf("after a: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	// The harness select opens on current; codex is the option above it (agy, claude, codex, current, ...).
	e = send(t, e, press(tea.KeyUp, "", 0), enter, enter)

	if e.screen != screenList || e.form != nil {
		t.Fatalf("after the form: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	if got := e.orderedWorking(); !slices.Equal(got, []settings.Agent{current, {Harness: "codex", Model: mo.None[string]()}}) {
		t.Fatalf("the list shows %+v, want current then codex", got)
	}

	if config.agents != nil {
		t.Fatal("adding wrote before enter")
	}

	e = send(t, e, enter)

	if config.active != "default" || config.feature != "review" || len(config.agents) != 2 || config.agents[1].Harness != "codex" {
		t.Errorf("SetList was given %q %q %+v, want default review current, codex", config.active, config.feature, config.agents)
	}
}

// The add form tells a person who does not know the model string where to get it, and offers a
// harnessConfig variant beside the harness names.
func TestEditorAddFormNamesTheEquipSkillAndTheVariants(t *testing.T) {
	want := "Not sure of the model string? Run /codefall-equip agents in any harness; " +
		"it finds the model and writes the entry for you."
	if equipHint != want {
		t.Fatalf("equipHint = %q, want %q", equipHint, want)
	}

	shown := twoEntries()
	shown.HarnessConfigs = harnessBlocks

	config := &fakeConfig{shown: shown, write: domain.Changed("set default review in .codefall/settings.json: current, codex-direct")}
	e := newTestEditor(t, config)

	// default, review, add; the model field is the second, so its description is drawn once the
	// select has been answered.
	e = send(t, e, enter, enter, enter, press('a', "a", 0))
	if e.screen != screenAdd || e.form == nil {
		t.Fatalf("after a: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	view := e.View().Content
	for _, fragment := range []string{"codefall-equip", "codex-direct"} {
		if !strings.Contains(view, fragment) {
			t.Errorf("add form view = %q, want it to hold %q", view, fragment)
		}
	}

	// The variant is the last option: below current, muse, and opencode.
	e = send(t, e, press(tea.KeyDown, "", 0), press(tea.KeyDown, "", 0), press(tea.KeyDown, "", 0), enter, enter)
	if e.screen != screenList {
		t.Fatalf("after the form: screen = %v, want the list", e.screen)
	}

	if got := e.orderedWorking(); !slices.Equal(got, []settings.Agent{current, {Harness: "codex-direct", Model: mo.None[string]()}}) {
		t.Fatalf("the list shows %+v, want current then codex-direct", got)
	}
}

func TestEditorSetsPosting(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), write: domain.Changed("set posting on in .codefall/settings.json")}
	e := newTestEditor(t, config)

	// Reviews, then the select opens on off; down is on.
	e = send(t, e, down, enter, down, enter)

	if got, ok := config.posting.Get(); !ok || !got {
		t.Errorf("SetPosting was given %v, want on", config.posting)
	}

	if e.screen != screenMenu || len(e.writes) != 1 {
		t.Errorf("after setting posting: screen = %v, writes = %d", e.screen, len(e.writes))
	}
}

func TestEditorSetsThePersona(t *testing.T) {
	config := &fakeConfig{shown: twoEntries(), writes: []domain.Write{domain.Changed("set persona to engineer in .codefall/user.json")}}
	e := newTestEditor(t, config)

	e = send(t, e, down, down, enter, enter)

	if config.set != "engineer" {
		t.Errorf("SetPersona was given %q, want the option under the cursor, engineer", config.set)
	}

	if e.screen != screenMenu || len(e.writes) != 1 {
		t.Errorf("after setting the persona: screen = %v, writes = %d", e.screen, len(e.writes))
	}
}

// Every visible string is in plain words: nothing about walking or orders, and current is "this
// harness".
func TestEditorViewSpeaksPlainly(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoEntries()})

	if view := e.View().Content; !strings.Contains(view, "codefall config") || !strings.Contains(view, "q quit") {
		t.Errorf("menu view = %q, want the heading and the quit key", view)
	}

	e = send(t, e, enter)
	if view := e.View().Content; !strings.Contains(view, "default (any harness without its own entry)") || !strings.Contains(view, "this harness") {
		t.Errorf("entries view = %q, want the default row in plain words", view)
	}

	e = send(t, e, down, down, down, down, enter, enter)

	view := e.View().Content
	for _, want := range []string{"when running in muse", "who reviews", "a add an agent", "d remove", "c use the default instead", "enter save"} {
		if !strings.Contains(view, want) {
			t.Errorf("list view = %q, want it to hold %q", view, want)
		}
	}

	for _, banned := range []string{"walk", "wider", "resolved"} {
		if strings.Contains(strings.ToLower(view), banned) {
			t.Errorf("list view holds %q, want plain words", banned)
		}
	}
}

// Skills is the fourth item and opens the prefix form on the prefix the settings hold; the option
// chosen is handed to the use case and the editor returns to the menu with the write recorded.
func TestEditorSetsTheSkillPrefix(t *testing.T) {
	shown := twoEntries()
	shown.SkillPrefix = settings.DefaultSkillPrefix

	config := &fakeConfig{shown: shown, write: domain.Changed("set skill prefix cf in .codefall/settings.json; run codefall upgrade to rename the installed skills")}
	e := newTestEditor(t, config)

	e = send(t, e, down, down, down, enter)
	if e.screen != screenSkillPrefix || e.form == nil {
		t.Fatalf("after enter on Skills: screen = %v, form nil = %v", e.screen, e.form == nil)
	}

	// The options are the closed set in order, cf, cfall, codefall, opened on codefall; up twice is cf.
	e = send(t, e, press(tea.KeyUp, "", 0), press(tea.KeyUp, "", 0), enter)

	if config.prefix != settings.SkillPrefixCf {
		t.Errorf("SetSkillPrefix was given %q, want cf", config.prefix)
	}

	if e.screen != screenMenu || len(e.writes) != 1 {
		t.Errorf("after setting the prefix: screen = %v, writes = %d", e.screen, len(e.writes))
	}
}
