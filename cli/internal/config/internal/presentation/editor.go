package presentation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/application"
	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/ui"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// The editor is `codefall config` with no arguments in a terminal: a menu over the sections of the
// two files, each section showing its current value and offering its edits, and Esc bringing the
// person back to the menu. It is the shell that owns arrangement and navigation for this component
// (ADR-012). It decides nothing the scripted subcommands do not: every edit goes through the same
// use case, and the one line each write earns is printed after the program has left the screen.

// screen is which part of the editor has the keyboard.
type screen int

const (
	screenMenu screen = iota
	screenAgents
	screenAdd
	screenOrder
	screenHarness
	screenPersona
	// screenQuit is the menu's last item rather than a screen: choosing it ends the program.
	screenQuit
)

// section is one item of the menu: where it leads, the current value shown under its title, and,
// for a section that is one narrower order, which order and where it sits.
type section struct {
	title  string
	detail string
	leads  screen
	order  mo.Option[application.Order]
	path   string
	// current reads the order as the configuration holds it, for the sections that are an order.
	current func(domain.Configuration) mo.Option[[]string]
}

func (s section) Title() string       { return s.title }
func (s section) Description() string { return s.detail }
func (s section) FilterValue() string { return s.title }

// harnessItem is one item of the per-harness list: a harness name, and the order it carries.
type harnessItem struct {
	name   string
	detail string
}

func (h harnessItem) Title() string       { return h.name }
func (h harnessItem) Description() string { return h.detail }
func (h harnessItem) FilterValue() string { return h.name }

// editorKeys is what the editor answers to beside what its lists and forms answer to themselves.
type editorKeys struct {
	Back, Select, Quit, Add, Remove, Save key.Binding
}

func defaultEditorKeys() editorKeys {
	return editorKeys{
		Back:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Remove: key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "remove")),
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

	menu      list.Model
	agents    ui.OrderList
	order     ui.OrderList
	orderName application.Order
	orderPath string
	harnesses list.Model
	form      *huh.Form
	// fields is where the form on screen writes. It is a pointer because the editor is held by
	// value and copied on every update, and the form binds to an address once, when it is built.
	fields *addFields

	status     string
	statusTone ui.Tone
	writes     []domain.Write

	width, height int
}

// addFields is where a form writes what the person typed or chose: the add-agent form's three
// fields, or the persona form's one.
type addFields struct {
	name, harness, model string
	persona              string
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
	e.harnesses = newHarnessList(shown)

	return e, nil
}

func (e editor) Init() tea.Cmd {
	return nil
}

// Update routes each message to the screen that has the keyboard. Quitting and going back are the
// editor's own; everything else is the section's.
func (e editor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		e.width, e.height = size.Width, size.Height
		e.menu.SetSize(size.Width, e.listHeight())
		e.harnesses.SetSize(size.Width, e.listHeight())

		if e.form != nil {
			e.form = e.form.WithWidth(min(size.Width, 72))
		}
	}

	switch e.screen {
	case screenMenu:
		return e.updateMenu(msg)
	case screenAgents:
		return e.updateAgents(msg)
	case screenAdd, screenPersona:
		return e.updateForm(msg)
	case screenOrder:
		return e.updateOrder(msg)
	case screenHarness:
		return e.updateHarness(msg)
	}

	return e, nil
}

func (e editor) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, e.keys.Quit):
			return e, tea.Quit
		case key.Matches(press, e.keys.Select):
			chosen, ok := e.menu.SelectedItem().(section)
			if !ok {
				return e, nil
			}

			if order, isOrder := chosen.order.Get(); isOrder {
				return e.openOrder(order, chosen.path, chosen.current(e.shown))
			}

			return e.open(chosen.leads)
		}
	}

	var cmd tea.Cmd
	e.menu, cmd = e.menu.Update(msg)

	return e, cmd
}

// open moves the keyboard to a section, building the model behind it from the configuration as it
// stands now.
func (e editor) open(to screen) (tea.Model, tea.Cmd) {
	e.status = ""

	switch to {
	case screenQuit:
		return e, tea.Quit
	case screenAgents:
		e.agents = ui.NewOrderList(agentEntries(e.shown.Agents), false)
	case screenHarness:
		e.harnesses = newHarnessList(e.shown)
	case screenPersona:
		e.fields = &addFields{persona: e.shown.Persona.Name}
		e.form = e.personaForm()
		e.screen = to

		return e, e.form.Init()
	}

	e.screen = to

	return e, nil
}

// openOrder moves the keyboard to one narrower order: every agent in the list, the ones the order
// names switched on and in the order's own sequence first.
func (e editor) openOrder(order application.Order, path string, current mo.Option[[]string]) (tea.Model, tea.Cmd) {
	e.orderName, e.orderPath = order, path
	e.order = ui.NewOrderList(orderEntries(e.shown.Agents, current), true)
	e.screen = screenOrder
	e.status = ""

	return e, nil
}

func (e editor) updateAgents(msg tea.Msg) (tea.Model, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return e, nil
	}

	switch {
	case key.Matches(press, e.keys.Back):
		return e.back()
	case key.Matches(press, e.keys.Add):
		e.fields = &addFields{harness: settings.HarnessCurrent}
		e.form = e.addForm()
		e.screen = screenAdd

		return e, e.form.Init()
	case key.Matches(press, e.keys.Remove):
		selected, ok := e.agents.Selected()
		if !ok {
			return e, nil
		}

		return e.apply(func() ([]domain.Write, error) {
			write, err := e.config.RemoveAgent(e.dir, selected.ID)

			return []domain.Write{write}, err
		}, screenAgents)
	case key.Matches(press, e.keys.Save):
		ids := e.agents.IDs()

		return e.apply(func() ([]domain.Write, error) {
			write, err := e.config.OrderAgents(e.dir, ids)

			return []domain.Write{write}, err
		}, screenAgents)
	}

	e.agents = e.agents.Update(msg)

	return e, nil
}

func (e editor) updateOrder(msg tea.Msg) (tea.Model, tea.Cmd) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return e, nil
	}

	switch {
	case key.Matches(press, e.keys.Back):
		return e.back()
	case key.Matches(press, e.keys.Save):
		ids := e.order.IDs()
		order := e.orderName

		return e.apply(func() ([]domain.Write, error) {
			var (
				write domain.Write
				err   error
			)

			if len(ids) == 0 {
				write, err = e.config.ClearOrder(e.dir, order)
			} else {
				write, err = e.config.SetOrder(e.dir, order, ids)
			}

			return []domain.Write{write}, err
		}, e.after(order))
	}

	e.order = e.order.Update(msg)

	return e, nil
}

// after is where the keyboard goes once an order is saved: back to the per-harness list for a
// harness's order, and to the menu for review's or consult's.
func (e editor) after(order application.Order) screen {
	if _, isHarness := order.Harness().Get(); isHarness {
		return screenHarness
	}

	return screenMenu
}

func (e editor) updateHarness(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, e.keys.Back):
			return e.back()
		case key.Matches(press, e.keys.Select):
			chosen, ok := e.harnesses.SelectedItem().(harnessItem)
			if !ok {
				return e, nil
			}

			current := mo.None[[]string]()
			if order, has := e.shown.ByHarness[chosen.name]; has {
				current = mo.Some(order)
			}

			return e.openOrder(application.HarnessOrder(chosen.name), settings.FieldAgentsByHarness+"."+chosen.name, current)
		}
	}

	var cmd tea.Cmd
	e.harnesses, cmd = e.harnesses.Update(msg)

	return e, cmd
}

// updateForm drives the Huh form on screen and acts on it once the person has finished or backed out
// of it. Esc is the editor's own back, since a form does not leave on its own.
func (e editor) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if press, ok := msg.(tea.KeyPressMsg); ok && key.Matches(press, e.keys.Back) {
		e.form = nil

		if e.screen == screenAdd {
			e.screen = screenAgents

			return e, nil
		}

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

		if e.screen == screenAdd {
			agent := application.NewAgent{
				Name:    strings.TrimSpace(e.fields.name),
				Harness: e.fields.harness,
				Model:   given(e.fields.model),
			}

			return e.apply(func() ([]domain.Write, error) {
				write, err := e.config.AddAgent(e.dir, agent)

				return []domain.Write{write}, err
			}, screenAgents)
		}

		persona := e.fields.persona

		return e.apply(func() ([]domain.Write, error) {
			return e.config.SetPersona(e.dir, persona)
		}, screenMenu)
	}

	return e, cmd
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
	e.harnesses = newHarnessList(shown)
	e.harnesses.SetSize(e.width, e.listHeight())

	if e.screen == screenAgents {
		e.agents = ui.NewOrderList(agentEntries(shown.Agents), false)
	}

	return e
}

// back returns the keyboard to the menu from any section, with the menu showing the file as it now
// stands. From a harness's order it returns to the harness list instead.
func (e editor) back() (tea.Model, tea.Cmd) {
	if e.screen == screenOrder {
		if _, isHarness := e.orderName.Harness().Get(); isHarness {
			e.screen = screenHarness

			return e, nil
		}
	}

	e.screen = screenMenu

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
	case screenAgents:
		body.WriteString(ui.Style(ui.TonePrimary).Render("Agents") + "\n")
		body.WriteString(ui.Style(ui.ToneFaint).Render("The list every use walks, in order.") + "\n\n")
		body.WriteString(e.agents.View())
	case screenOrder:
		body.WriteString(ui.Style(ui.TonePrimary).Render(e.orderPath) + "\n")
		body.WriteString(ui.Style(ui.ToneFaint).Render(
			"Switch on the agents this order walks, in sequence. None switched on clears the order.") + "\n\n")
		body.WriteString(e.order.View())
	case screenHarness:
		body.WriteString(e.harnesses.View())
	case screenAdd, screenPersona:
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
	case screenAgents:
		return e.agents.Help() + " · a add · d remove · enter save order · esc back"
	case screenOrder:
		return e.order.Help() + " · enter save · esc back"
	case screenHarness:
		return "↑/↓ move · enter open · esc back"
	default:
		return "enter next · esc back"
	}
}

// listHeight is what is left for a list once the heading, the status, and the help have their lines.
func (e editor) listHeight() int {
	return max(e.height-8, 6)
}

// addForm is the form that adds an agent: its name, the harness that runs it, and optionally the
// model that harness is asked for.
func (e editor) addForm() *huh.Form {
	harnesses := settings.AgentHarnesses()
	options := make([]huh.Option[string], 0, len(harnesses))

	for _, name := range harnesses {
		label := name
		if name == settings.HarnessCurrent {
			label = name + " (whichever harness is running the session)"
		}

		options = append(options, huh.NewOption(label, name))
	}

	return huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Name").Description("A short lower-case name a person types after via=").
			Value(&e.fields.name).Validate(func(name string) error {
			if strings.TrimSpace(name) == "" {
				return errors.New("an agent needs a name")
			}

			return nil
		}),
		huh.NewSelect[string]().Title("Harness").Options(options...).Value(&e.fields.harness),
		huh.NewInput().Title("Model").Description("Optional; passed to the harness as written").
			Value(&e.fields.model),
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
			Description("Who you work as; it changes how the verbs talk to you, never what they may do.").
			Options(options...).Value(&e.fields.persona),
	)).WithShowHelp(true).WithWidth(min(e.width, 72))
}

// newMenu builds the menu over the sections, each showing its current value.
func newMenu(shown domain.Configuration) list.Model {
	items := []list.Item{
		section{title: "Agents", detail: agentsDetail(shown), leads: screenAgents},
		section{
			title: "Review order", detail: orderDetail(shown.Review), leads: screenOrder,
			order: mo.Some(application.ReviewOrder()), path: settings.BlockReview + "." + settings.FieldReviewAgents,
			current: func(c domain.Configuration) mo.Option[[]string] { return c.Review },
		},
		section{
			title: "Consult order", detail: orderDetail(shown.Consult), leads: screenOrder,
			order: mo.Some(application.ConsultOrder()), path: settings.BlockConsult + "." + settings.FieldConsultAgents,
			current: func(c domain.Configuration) mo.Option[[]string] { return c.Consult },
		},
		section{title: "Per-harness orders", detail: byHarnessDetail(shown.ByHarness), leads: screenHarness},
		section{title: "Persona", detail: personaLine(shown.Persona), leads: screenPersona},
		section{title: "Quit", detail: "Leave the editor", leads: screenQuit},
	}

	return newList(items, "Sections")
}

// newHarnessList builds the list of harnesses a per-harness order may be set for.
func newHarnessList(shown domain.Configuration) list.Model {
	names := harness.All()
	items := make([]list.Item, 0, len(names))

	for _, name := range names {
		detail := "walks the wider order"
		if order, has := shown.ByHarness[name]; has {
			detail = strings.Join(order, ", ")
		}

		items = append(items, harnessItem{name: name, detail: detail})
	}

	return newList(items, "Per-harness orders")
}

// newList is a Bubbles list with what this editor does not want switched off: filtering, the status
// bar, and the list's own help and quit keys, since the editor draws its own help and owns quitting.
func newList(items []list.Item, title string) list.Model {
	delegate := list.NewDefaultDelegate()

	l := list.New(items, delegate, 80, 16)
	l.Title = title
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	return l
}

// agentEntries is the agents list as order entries, each labelled the way show describes it.
func agentEntries(agents []settings.Agent) []ui.OrderEntry {
	entries := make([]ui.OrderEntry, 0, len(agents))
	for _, agent := range agents {
		entries = append(entries, ui.OrderEntry{ID: agent.Name, Label: settings.DescribeAgent(agent)})
	}

	return entries
}

// orderEntries is every agent as a toggling entry: the ones a narrower order names first, in that
// order and switched on, then the rest switched off.
func orderEntries(agents []settings.Agent, current mo.Option[[]string]) []ui.OrderEntry {
	byName := map[string]settings.Agent{}
	for _, agent := range agents {
		byName[agent.Name] = agent
	}

	named := current.OrElse(nil)
	entries := make([]ui.OrderEntry, 0, len(agents))

	for _, name := range named {
		if agent, ok := byName[name]; ok {
			entries = append(entries, ui.OrderEntry{ID: name, Label: settings.DescribeAgent(agent), On: true})
		}
	}

	for _, agent := range agents {
		if !slices.Contains(named, agent.Name) {
			entries = append(entries, ui.OrderEntry{ID: agent.Name, Label: settings.DescribeAgent(agent)})
		}
	}

	return entries
}

func agentsDetail(shown domain.Configuration) string {
	names := settings.AgentNames(shown.Agents)
	detail := strings.Join(names, ", ")

	if shown.Default {
		detail += " (the default; the settings list none)"
	}

	return detail
}

func orderDetail(order mo.Option[[]string]) string {
	if names, ok := order.Get(); ok {
		return strings.Join(names, ", ")
	}

	return "not set; walks the wider order"
}

func byHarnessDetail(byHarness map[string][]string) string {
	if len(byHarness) == 0 {
		return "none set; every harness walks the wider order"
	}

	parts := make([]string, 0, len(byHarness))
	for _, name := range slices.Sorted(maps.Keys(byHarness)) {
		parts = append(parts, name+": "+strings.Join(byHarness[name], ", "))
	}

	return strings.Join(parts, "; ")
}
