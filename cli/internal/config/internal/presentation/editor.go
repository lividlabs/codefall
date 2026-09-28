package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// The editor is `codefall config` with no arguments in a terminal: a menu over what a person
// configures, Agents, Reviews, and Persona, each screen showing its current value and offering its
// edits, and Esc bringing the person back a step. It is the shell that owns arrangement and
// navigation for this component (ADR-012). It decides nothing the scripted subcommands do not: every
// edit goes through the same use case, and the one line each write earns is printed after the program
// has left the screen.

// screen is which part of the editor has the keyboard.
type screen int

const (
	screenMenu screen = iota
	// screenEntries lists the active agents: default, then each harness, with or without an entry.
	screenEntries
	// screenEntry is one active agent's two lists, Review and Consult.
	screenEntry
	// screenList is one list being edited: reorder, add, remove, clear, save.
	screenList
	// screenAdd is the form that adds an agent to the list on screen.
	screenAdd
	// screenReviews holds the review settings that do not vary by harness: posting.
	screenReviews
	screenPersona
	// screenQuit is the menu's last item rather than a screen: choosing it ends the program.
	screenQuit
)

// row is one item of a Bubbles list in this editor: what it says, what it leads to, and the value it
// stands for, an active agent or a feature, when it stands for one.
type row struct {
	title  string
	detail string
	leads  screen
	value  string
}

func (r row) Title() string       { return r.title }
func (r row) Description() string { return r.detail }
func (r row) FilterValue() string { return r.title }

// editorKeys is what the editor answers to beside what its lists and forms answer to themselves.
type editorKeys struct {
	Back, Select, Quit, Add, Remove, Clear, Save key.Binding
}

func defaultEditorKeys() editorKeys {
	return editorKeys{
		Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add an agent")),
		Remove: key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "remove")),
		Clear:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "use the default instead")),
		Save:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
	}
}

// editor is the whole program: the configuration as it stands, the screen that has the keyboard,
// the models behind each screen, the last thing worth saying, and every write the session made.
type editor struct {
	config ConfigUseCase
	dir    string
	keys   editorKeys

	shown  domain.Configuration
	screen screen

	menu    list.Model
	entries list.Model
	entry   list.Model
	// active and feature name the list on screen; working is that list as the person has it so far,
	// in the order of the list model, saved only on enter.
	active  string
	feature string
	working []settings.Agent
	agents  ui.OrderList
	form    *huh.Form
	// fields is where the form on screen writes. It is a pointer because the editor is held by
	// value and copied on every update, and the form binds to an address once, when it is built.
	fields *formFields

	status     string
	statusTone ui.Tone
	writes     []domain.Write

	width, height int
}

// formFields is where a form writes what the person typed or chose: the add-agent form's two fields,
// the posting form's one, or the persona form's one.
type formFields struct {
	harness, model string
	posting        bool
	persona        string
}

// runEditor opens the editor, and, once it has left the screen, prints the writes the session made
// the way the subcommands print theirs. A person quitting with Ctrl-C has stopped editing rather
// than hit an error, so the writes already made are still reported and nothing is returned.
func runEditor(ctx context.Context, config ConfigUseCase, dir string, out io.Writer) error {
	model, err := newEditor(config, dir)
	if err != nil {
		return err
	}

	// The palette asks the terminal a question on stdin; it has to be asked before the program owns
	// stdin, or the answer goes to the program and the question waits for ever.
	ui.Prime()

	final, err := tea.NewProgram(model, tea.WithContext(ctx)).Run()
	if err != nil && !errors.Is(err, tea.ErrInterrupted) {
		return fmt.Errorf("config: %w", err)
	}

	finished, ok := final.(editor)
	if !ok {
		return nil
	}

	if len(finished.writes) == 0 {
		return writeLines(out, []string{"nothing changed"})
	}

	return writeResults(out, finished.writes...)
}

// newEditor reads the configuration once before the program starts, so a directory config cannot
// read is refused in the same words the subcommands use, before anything is drawn.
func newEditor(config ConfigUseCase, dir string) (editor, error) {
	e := editor{config: config, dir: dir, keys: defaultEditorKeys(), width: 80, height: 24}

	shown, err := config.Show(dir)
	if err != nil {
		return editor{}, err
	}

	e.shown = shown
	e.menu = newMenu(shown)
	e.entries = newEntriesList(shown)
	// Every list is built before the program starts, because the first window-size message resizes
	// them all and a Bubbles list that was never built cannot be resized.
	e.active = settings.ActiveDefault
	e.entry = newEntryList(shown, e.active)
	e.agents = newAgentsList(nil)

	return e, nil
}

func (e editor) Init() tea.Cmd {
	return nil
}

// Update routes each message to the screen that has the keyboard. Quitting and going back are the
// editor's own; everything else is the screen's.
func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		e.width, e.height = size.Width, size.Height
		e.menu.SetSize(size.Width, e.listHeight())
		e.entries.SetSize(size.Width, e.listHeight())
		e.entry.SetSize(size.Width, e.listHeight())

		if e.form != nil {
			e.form = e.form.WithWidth(min(size.Width, 72))
		}
	}

	switch e.screen {
	case screenMenu:
		return e.updateMenu(msg)
	case screenEntries:
		return e.updateEntries(msg)
	case screenEntry:
		return e.updateEntry(msg)
	case screenList:
		return e.updateList(msg)
	case screenAdd, screenReviews, screenPersona:
		return e.updateForm(msg)
	}

	return e, nil
}

func (e editor) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, e.keys.Quit):
			return e, tea.Quit
		case key.Matches(press, e.keys.Select):
			chosen, ok := e.menu.SelectedItem().(row)
			if !ok {
				return e, nil
			}

			return e.open(chosen.leads)
		}
	}

	var cmd tea.Cmd
	e.menu, cmd = e.menu.Update(msg)

	return e, cmd
}

// open moves the keyboard to a screen, building the model behind it from the configuration as it
// stands now.
func (e editor) open(to screen) (tea.Model, tea.Cmd) {
	e.status = ""

	switch to {
	case screenQuit:
		return e, tea.Quit
	case screenEntries:
		e.entries = newEntriesList(e.shown)
		e.entries.SetSize(e.width, e.listHeight())
	case screenReviews:
		e.fields = &formFields{posting: e.shown.Posting}
		e.form = e.postingForm()
		e.screen = to

		return e, e.form.Init()
	case screenPersona:
		e.fields = &formFields{persona: e.shown.Persona.Name}
		e.form = e.personaForm()
		e.screen = to

		return e, e.form.Init()
	}

	e.screen = to

	return e, nil
}

func (e editor) updateEntries(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, e.keys.Back):
			return e.back()
		case key.Matches(press, e.keys.Select):
			chosen, ok := e.entries.SelectedItem().(row)
			if !ok {
				return e, nil
			}

			e.active = chosen.value
			e.entry = newEntryList(e.shown, e.active)
			e.entry.SetSize(e.width, e.listHeight())
			e.screen = screenEntry
			e.status = ""

			return e, nil
		}
	}

	var cmd tea.Cmd
	e.entries, cmd = e.entries.Update(msg)

	return e, cmd
}

func (e editor) updateEntry(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, e.keys.Back):
			return e.back()
		case key.Matches(press, e.keys.Select):
			chosen, ok := e.entry.SelectedItem().(row)
			if !ok {
				return e, nil
			}

			return e.openList(chosen.value)
		}
	}

	var cmd tea.Cmd
	e.entry, cmd = e.entry.Update(msg)

	return e, cmd
}

// openList moves the keyboard to one list: the agents it names, in order, ready to reorder, add to,
// remove from, clear, or save. A list the entry leaves to the default starts from the default's, so
// a person edits what a run would use rather than an empty screen.
func (e editor) openList(feature string) (tea.Model, tea.Cmd) {
	e.feature = feature
	e.working = e.currentList()
	e.agents = newAgentsList(e.working)
	e.screen = screenList
	e.status = ""

	return e, nil
}

// currentList is the list on screen as the file has it: the entry's own, else what the entry resolves
// to through the default.
func (e editor) currentList() []settings.Agent {
	for _, entry := range e.shown.Entries {
		if entry.ActiveAgent == e.active {
			if agents, has := entry.List(e.feature).Get(); has {
				return slices.Clone(agents)
			}
		}
	}

	for _, entry := range e.shown.Entries {
		if entry.ActiveAgent == settings.ActiveDefault {
			if agents, has := entry.List(e.feature).Get(); has {
				return slices.Clone(agents)
			}
		}
	}

	return []settings.Agent{settings.CurrentAgent()}
}

func (e editor) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return e, nil
	}

	switch {
	case key.Matches(press, e.keys.Back):
		return e.back()
	case key.Matches(press, e.keys.Add):
		e.fields = &formFields{harness: settings.HarnessCurrent}
		e.form = e.addForm()
		e.screen = screenAdd

		return e, e.form.Init()
	case key.Matches(press, e.keys.Remove):
		selected, ok := e.agents.Selected()
		if !ok {
			return e, nil
		}

		e.working = e.orderedWorking()
		at, _ := strconv.Atoi(selected.ID)
		e.working = slices.Delete(e.working, at, at+1)
		e.agents = newAgentsList(e.working)

		return e, nil
	case key.Matches(press, e.keys.Clear):
		active, feature := e.active, e.feature

		return e.apply(func() ([]domain.Write, error) {
			write, err := e.config.ClearList(e.dir, active, feature)

			return []domain.Write{write}, err
		}, screenEntry)
	case key.Matches(press, e.keys.Save):
		active, feature, agents := e.active, e.feature, e.orderedWorking()

		return e.apply(func() ([]domain.Write, error) {
			var (
				write domain.Write
				err   error
			)

			if len(agents) == 0 {
				write, err = e.config.ClearList(e.dir, active, feature)
			} else {
				write, err = e.config.SetList(e.dir, active, feature, agents)
			}

			return []domain.Write{write}, err
		}, screenEntry)
	}

	e.agents = e.agents.Update(msg)

	return e, nil
}

// orderedWorking is the list on screen in the order the person has put it.
func (e editor) orderedWorking() []settings.Agent {
	ids := e.agents.IDs()
	ordered := make([]settings.Agent, 0, len(ids))

	for _, id := range ids {
		at, _ := strconv.Atoi(id)
		ordered = append(ordered, e.working[at])
	}

	return ordered
}

// updateForm drives the Huh form on screen and acts on it once the person has finished or backed out
// of it. Esc is the editor's own back, since a form does not leave on its own.
func (e editor) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok && key.Matches(press, e.keys.Back) {
		e.form = nil

		return e.back()
	}

	model, cmd := e.form.Update(msg)

	form, ok := model.(*huh.Form)
	if !ok {
		return e, cmd
	}

	e.form = form

	switch e.form.State {
	case huh.StateAborted:
		e.form = nil

		return e.back()
	case huh.StateCompleted:
		e.form = nil

		return e.completeForm()
	}

	return e, cmd
}

// completeForm acts on a finished form: an added agent joins the list on screen, unsaved; posting
// and the persona are written at once, since each is one value.
func (e editor) completeForm() (tea.Model, tea.Cmd) {
	switch e.screen {
	case screenAdd:
		e.working = append(e.orderedWorking(), settings.Agent{
			Harness: e.fields.harness, Model: given(e.fields.model),
		})
		e.agents = newAgentsList(e.working)
		e.screen = screenList

		return e, nil
	case screenReviews:
		on := e.fields.posting

		return e.apply(func() ([]domain.Write, error) {
			write, err := e.config.SetPosting(e.dir, on)

			return []domain.Write{write}, err
		}, screenMenu)
	default:
		persona := e.fields.persona

		return e.apply(func() ([]domain.Write, error) {
			return e.config.SetPersona(e.dir, persona)
		}, screenMenu)
	}
}

// apply runs one write through the use case and shows what it said: the write's own line on
// success, the refusal on failure, either way inline and with the file left as the use case left it.
// Then the configuration is read again, so every screen shows what the file now says, and the
// keyboard goes where the caller said.
func (e editor) apply(write func() ([]domain.Write, error), then screen) (tea.Model, tea.Cmd) {
	writes, err := write()
	if err != nil {
		e.status, e.statusTone = err.Error(), ui.ToneFail
		e.screen = then

		return e.refresh(), nil
	}

	details := make([]string, 0, len(writes))

	for _, w := range writes {
		details = append(details, w.Detail)

		if w.Changed {
			e.writes = append(e.writes, w)
		}
	}

	e.status, e.statusTone = strings.Join(details, "; "), ui.TonePrimary
	e.screen = then

	return e.refresh(), nil
}

// refresh reads the configuration again and rebuilds every model that shows it. A read that fails
// keeps what was shown and says so, since the file is what it was a moment ago.
func (e editor) refresh() editor {
	shown, err := e.config.Show(e.dir)
	if err != nil {
		e.status, e.statusTone = err.Error(), ui.ToneFail

		return e
	}

	e.shown = shown
	e.menu = newMenu(shown)
	e.menu.SetSize(e.width, e.listHeight())
	e.entries = newEntriesList(shown)
	e.entries.SetSize(e.width, e.listHeight())

	if e.active != "" {
		e.entry = newEntryList(shown, e.active)
		e.entry.SetSize(e.width, e.listHeight())
	}

	return e
}

// back returns the keyboard one step: from a list to its entry, from an entry to the entries, from
// an add form to its list, and from anywhere else to the menu.
func (e editor) back() (tea.Model, tea.Cmd) {
	switch e.screen {
	case screenAdd:
		e.screen = screenList
	case screenList:
		e.screen = screenEntry
	case screenEntry:
		e.screen = screenEntries
	default:
		e.screen = screenMenu
	}

	return e, nil
}

// View draws the heading, the screen that has the keyboard, the last thing worth saying, and the
// keys that screen answers to.
func (e editor) View() tea.View {
	var body strings.Builder

	body.WriteString(ui.HeadingStyle().Render("codefall config"))
	body.WriteString("\n\n")

	switch e.screen {
	case screenMenu:
		body.WriteString(e.menu.View())
	case screenEntries:
		body.WriteString(e.entries.View())
	case screenEntry:
		body.WriteString(e.entry.View())
	case screenList:
		body.WriteString(ui.Style(ui.TonePrimary).Render(listTitle(e.active, e.feature)) + "\n")
		body.WriteString(ui.Style(ui.ToneFaint).Render(
			"Tried in this order until one answers. Move an agent with shift and the arrows; enter saves.") + "\n\n")
		body.WriteString(e.agents.View())
	case screenAdd, screenReviews, screenPersona:
		if e.form != nil {
			body.WriteString(e.form.View())
		}
	}

	body.WriteString("\n")

	if e.status != "" {
		body.WriteString("\n" + ui.Style(e.statusTone).Render(ui.Mark(e.statusTone)) + " " + e.status + "\n")
	}

	body.WriteString("\n" + ui.Style(ui.ToneFaint).Render(e.help()))

	return tea.NewView(body.String())
}

// help is the key legend for the screen that has the keyboard.
func (e editor) help() string {
	switch e.screen {
	case screenMenu:
		return "↑/↓ move · enter open · q quit"
	case screenEntries, screenEntry:
		return "↑/↓ move · enter open · esc back"
	case screenList:
		return e.agents.Help() + " · a add an agent · d remove · c use the default instead · enter save · esc back"
	default:
		return "enter next · esc back"
	}
}

// listHeight is what is left for a list once the heading, the status, and the help have their lines.
func (e editor) listHeight() int {
	return max(e.height-8, 6)
}

// equipHint is what the add form says about the model: the skill that finds it and writes the
// entry, for whoever does not know the model string by heart.
const equipHint = "Not sure of the model string? Run /codefall-equip agents in any harness; it finds the " +
	"model and writes the entry for you."

// addForm is the form that adds an agent to the list on screen: the harness that runs it, or a
// harnessConfig variant that names one, and optionally the model that harness is asked for.
func (e editor) addForm() *huh.Form {
	harnesses := settings.AgentHarnesses()
	options := make([]huh.Option[string], 0, len(harnesses))

	for _, name := range harnesses {
		label := name
		if name == settings.HarnessCurrent {
			label = "this harness (a subagent of whichever harness you are running in)"
		}

		options = append(options, huh.NewOption(label, name))
	}

	for _, key := range slices.Sorted(maps.Keys(e.shown.HarnessConfigs)) {
		if block := e.shown.HarnessConfigs[key]; block.Harness != key {
			options = append(options, huh.NewOption(key+" ("+block.Harness+", called as "+
				settings.FieldHarnessConfig+"."+key+" says)", key))
		}
	}

	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Which harness runs it?").Options(options...).Value(&e.fields.harness),
		huh.NewInput().Title("Model").Description("Optional; passed to the harness as written. "+equipHint).
			Value(&e.fields.model),
	)).WithShowHelp(true).WithWidth(min(e.width, 72))
}

// postingForm is the form that turns posting review findings to the pull request on or off.
func (e editor) postingForm() *huh.Form {
	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[bool]().Title("Post review findings to the pull request?").
			Description("On, review comments its findings on the pull request it reviewed. Off, they stay in the findings file.").
			Options(huh.NewOption("off", false), huh.NewOption("on", true)).Value(&e.fields.posting),
	)).WithShowHelp(true).WithWidth(min(e.width, 72))
}

// personaForm is the form that picks the persona.
func (e editor) personaForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(userfile.Personas()))
	for _, name := range userfile.Personas() {
		options = append(options, huh.NewOption(name, name))
	}

	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Persona").
			Description("Who you are to the skills. It changes how they talk to you, never what they may do.").
			Options(options...).Value(&e.fields.persona),
	)).WithShowHelp(true).WithWidth(min(e.width, 72))
}

// newMenu builds the menu over what a person configures, each item showing its current value.
func newMenu(shown domain.Configuration) list.Model {
	posting := "off"
	if shown.Posting {
		posting = "on"
	}

	items := []list.Item{
		row{title: "Agents", detail: "Who reviews and consults, by the harness you run codefall in", leads: screenEntries},
		row{title: "Reviews", detail: "Posting findings to the pull request: " + posting, leads: screenReviews},
		row{title: "Persona", detail: personaLine(shown.Persona), leads: screenPersona},
		row{title: "Quit", detail: "Leave the editor", leads: screenQuit},
	}

	return newList(items, "")
}

// newEntriesList builds the list of active agents: default first, then every harness, each saying
// what it has or that it uses the default.
func newEntriesList(shown domain.Configuration) list.Model {
	names := append([]string{settings.ActiveDefault}, harnessNames()...)
	items := make([]list.Item, 0, len(names))

	for _, name := range names {
		items = append(items, row{title: activeAgentLabel(name), detail: entryDetail(shown, name), value: name})
	}

	return newList(items, "Agents")
}

// harnessNames is every active agent but default, sorted.
func harnessNames() []string {
	return slices.DeleteFunc(settings.ActiveAgents(), func(name string) bool { return name == settings.ActiveDefault })
}

// entryDetail is one active agent's row: its two lists, or that it has no entry of its own.
func entryDetail(shown domain.Configuration, active string) string {
	for _, entry := range shown.Entries {
		if entry.ActiveAgent != active {
			continue
		}

		parts := make([]string, 0, 2)
		for _, feature := range settings.Features() {
			parts = append(parts, feature+": "+listLabel(entry.List(feature)))
		}

		return strings.Join(parts, " · ")
	}

	if active == settings.ActiveDefault {
		return "review: this harness · consult: this harness (built in; nothing set)"
	}

	return "no entry of its own; uses the default"
}

// newEntryList builds one active agent's screen: a row per list.
func newEntryList(shown domain.Configuration, active string) list.Model {
	items := make([]list.Item, 0, len(settings.Features()))

	for _, feature := range settings.Features() {
		items = append(items, row{title: featureTitle(feature), detail: featureDetail(shown, active, feature), value: feature})
	}

	return newList(items, activeAgentLabel(active))
}

// featureTitle is what a list is for, as a person reads it.
func featureTitle(feature string) string {
	if feature == settings.FeatureReview {
		return "Review: who reviews"
	}

	return "Consult: who is asked when a run is stuck"
}

// featureDetail is one list's row: the agents in order, or what it falls back to.
func featureDetail(shown domain.Configuration, active, feature string) string {
	for _, entry := range shown.Entries {
		if entry.ActiveAgent == active {
			if agents, has := entry.List(feature).Get(); has {
				return listLabel(mo.Some(agents))
			}
		}
	}

	if active == settings.ActiveDefault {
		return "this harness (built in; nothing set)"
	}

	return "same as default"
}

// listTitle names the list on screen.
func listTitle(active, feature string) string {
	return activeAgentLabel(active) + " · " + featureTitle(feature)
}

// newAgentsList is the list on screen as an order list, each agent labelled as a person reads it and
// identified by its position, since two agents may share a harness.
func newAgentsList(agents []settings.Agent) ui.OrderList {
	entries := make([]ui.OrderEntry, 0, len(agents))
	for at, agent := range agents {
		entries = append(entries, ui.OrderEntry{ID: strconv.Itoa(at), Label: agentLabel(agent)})
	}

	return ui.NewOrderList(entries, false)
}

// newList is a Bubbles list with what this editor does not want switched off: filtering, the status
// bar, and the list's own help and quit keys, since the editor draws its own help and owns quitting.
func newList(items []list.Item, title string) list.Model {
	delegate := list.NewDefaultDelegate()

	l := list.New(items, delegate, 80, 16)
	l.Title = title
	l.SetShowTitle(title != "")
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return l
}
