package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// OrderEntry is one entry of an OrderList: what the list shows for it, the value the caller reads
// back, and whether it is switched on, which matters only to a list that toggles.
type OrderEntry struct {
	ID    string
	Label string
	On    bool
}

// OrderList is a list a person puts in order: a cursor moves over the entries, an entry moves up or
// down with it, and, when the list toggles, an entry switches on and off. It knows nothing about
// what the entries are — a component binds its own data as labels and reads the IDs back (ADR-012).
// It is a Bubble Tea model in all but the return types: a component embeds it and hands it the key
// presses it does not handle itself.
type OrderList struct {
	entries []OrderEntry
	cursor  int
	toggles bool
	keys    OrderKeys
}

// OrderKeys is what an OrderList answers to. The defaults follow the Bubbles list: arrows or j and k
// move the cursor, the same with shift moves the entry, and space toggles.
type OrderKeys struct {
	Up, Down, MoveUp, MoveDown, Toggle key.Binding
}

// DefaultOrderKeys is the binding every OrderList starts with.
func DefaultOrderKeys() OrderKeys {
	return OrderKeys{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		MoveUp:   key.NewBinding(key.WithKeys("shift+up", "K"), key.WithHelp("shift+↑/K", "move up")),
		MoveDown: key.NewBinding(key.WithKeys("shift+down", "J"), key.WithHelp("shift+↓/J", "move down")),
		Toggle:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
	}
}

// NewOrderList builds a list over the entries, in the order given. A list that toggles shows each
// entry's switch and answers to the toggle key; one that does not shows the entries alone.
func NewOrderList(entries []OrderEntry, toggles bool) OrderList {
	return OrderList{entries: append([]OrderEntry(nil), entries...), toggles: toggles, keys: DefaultOrderKeys()}
}

// Update handles one message. A key press it answers to moves the cursor, moves the entry under it,
// or toggles that entry; anything else leaves the list as it was. It returns the list, so a caller
// can hold it by value the way Bubble Tea models are held.
func (l OrderList) Update(msg tea.Msg) OrderList {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok || len(l.entries) == 0 {
		return l
	}

	switch {
	case key.Matches(press, l.keys.Up):
		l.cursor = max(l.cursor-1, 0)
	case key.Matches(press, l.keys.Down):
		l.cursor = min(l.cursor+1, len(l.entries)-1)
	case key.Matches(press, l.keys.MoveUp):
		if l.cursor > 0 {
			l.entries[l.cursor-1], l.entries[l.cursor] = l.entries[l.cursor], l.entries[l.cursor-1]
			l.cursor--
		}
	case key.Matches(press, l.keys.MoveDown):
		if l.cursor < len(l.entries)-1 {
			l.entries[l.cursor+1], l.entries[l.cursor] = l.entries[l.cursor], l.entries[l.cursor+1]
			l.cursor++
		}
	case l.toggles && key.Matches(press, l.keys.Toggle):
		l.entries[l.cursor].On = !l.entries[l.cursor].On
	}

	return l
}

// View draws the list: one entry per line, the cursor's entry marked, and each entry's switch when
// the list toggles. An empty list says so.
func (l OrderList) View() string {
	if len(l.entries) == 0 {
		return Style(ToneFaint).Render("(nothing here)")
	}

	lines := make([]string, 0, len(l.entries))

	for at, entry := range l.entries {
		cursor := "  "
		if at == l.cursor {
			cursor = Style(TonePrimary).Render("› ")
		}

		toggle := ""
		if l.toggles {
			toggle = "[ ] "
			if entry.On {
				toggle = Style(TonePrimary).Render("[x]") + " "
			}
		}

		label := entry.Label
		if at == l.cursor {
			label = Style(TonePrimary).Render(label)
		}

		lines = append(lines, cursor+toggle+label)
	}

	return strings.Join(lines, "\n")
}

// Help is the one-line key legend for this list, in the order a person reaches for the keys.
func (l OrderList) Help() string {
	bindings := []key.Binding{l.keys.Up, l.keys.Down, l.keys.MoveUp, l.keys.MoveDown}
	if l.toggles {
		bindings = append(bindings, l.keys.Toggle)
	}

	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		parts = append(parts, binding.Help().Key+" "+binding.Help().Desc)
	}

	return strings.Join(parts, " · ")
}

// Entries is the list as it stands, in order, with each entry's switch.
func (l OrderList) Entries() []OrderEntry {
	return append([]OrderEntry(nil), l.entries...)
}

// IDs is every entry's ID in the list's order, or, for a list that toggles, the IDs of the entries
// that are on.
func (l OrderList) IDs() []string {
	ids := make([]string, 0, len(l.entries))

	for _, entry := range l.entries {
		if l.toggles && !entry.On {
			continue
		}

		ids = append(ids, entry.ID)
	}

	return ids
}

// Cursor is the index of the entry under the cursor.
func (l OrderList) Cursor() int {
	return l.cursor
}

// Selected is the entry under the cursor, and false when the list is empty.
func (l OrderList) Selected() (OrderEntry, bool) {
	if len(l.entries) == 0 {
		return OrderEntry{}, false
	}

	return l.entries[l.cursor], true
}

// Len is how many entries the list holds.
func (l OrderList) Len() int {
	return len(l.entries)
}
