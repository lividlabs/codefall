package ui

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// styling is what Lip Gloss wraps text in; the view is judged on its words.
var styling = regexp.MustCompile("\x1b\\[[0-9;]*m")

func three() []OrderEntry {
	return []OrderEntry{{ID: "a", Label: "A"}, {ID: "b", Label: "B", On: true}, {ID: "c", Label: "C"}}
}

func press(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text, Mod: mod}
}

func TestOrderListMovesTheCursorAndStopsAtTheEnds(t *testing.T) {
	l := NewOrderList(three(), false)

	l = l.Update(press(tea.KeyUp, "", 0))
	if l.Cursor() != 0 {
		t.Errorf("cursor after up at the top = %d, want 0", l.Cursor())
	}

	l = l.Update(press(tea.KeyDown, "", 0)).Update(press('j', "j", 0))
	if l.Cursor() != 2 {
		t.Errorf("cursor after two downs = %d, want 2", l.Cursor())
	}

	l = l.Update(press(tea.KeyDown, "", 0))
	if l.Cursor() != 2 {
		t.Errorf("cursor after down at the bottom = %d, want 2", l.Cursor())
	}

	if got, _ := l.Selected(); got.ID != "c" {
		t.Errorf("Selected = %+v, want c", got)
	}
}

func TestOrderListMovesTheEntryUnderTheCursor(t *testing.T) {
	l := NewOrderList(three(), false)

	l = l.Update(press(tea.KeyDown, "", 0)).Update(press(tea.KeyDown, "", tea.ModShift))

	if got := l.IDs(); !slices.Equal(got, []string{"a", "c", "b"}) {
		t.Errorf("IDs after moving b down = %q, want a c b", got)
	}

	if l.Cursor() != 2 {
		t.Errorf("cursor followed the entry to %d, want 2", l.Cursor())
	}

	l = l.Update(press('K', "K", tea.ModShift)).Update(press('K', "K", tea.ModShift))

	if got := l.IDs(); !slices.Equal(got, []string{"b", "a", "c"}) {
		t.Errorf("IDs after moving b to the top = %q, want b a c", got)
	}

	// A move past either end is ignored, and so is the cursor.
	l = l.Update(press(tea.KeyUp, "", tea.ModShift))

	if got := l.IDs(); !slices.Equal(got, []string{"b", "a", "c"}) || l.Cursor() != 0 {
		t.Errorf("after a move at the top: IDs = %q, cursor = %d, want b a c and 0", got, l.Cursor())
	}
}

func TestOrderListTogglesOnlyWhenAskedTo(t *testing.T) {
	plain := NewOrderList(three(), false).Update(press(tea.KeySpace, " ", 0))
	if got := plain.IDs(); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Errorf("a list that does not toggle answered space: IDs = %q", got)
	}

	toggling := NewOrderList(three(), true)

	if got := toggling.IDs(); !slices.Equal(got, []string{"b"}) {
		t.Errorf("IDs of a toggling list = %q, want only the entries that are on: b", got)
	}

	toggling = toggling.Update(press(tea.KeySpace, " ", 0)).Update(press(tea.KeyDown, "", 0)).Update(press(tea.KeySpace, " ", 0))

	if got := toggling.IDs(); !slices.Equal(got, []string{"a"}) {
		t.Errorf("IDs after toggling a on and b off = %q, want a", got)
	}
}

func TestOrderListViewMarksTheCursorAndTheSwitches(t *testing.T) {
	view := styling.ReplaceAllString(NewOrderList(three(), true).View(), "")

	for _, want := range []string{"› [ ] A", "  [x] B", "  [ ] C"} {
		if !strings.Contains(view, want) {
			t.Errorf("View() = %q, want it to hold %q", view, want)
		}
	}

	if got := NewOrderList(nil, false).View(); !strings.Contains(got, "nothing here") {
		t.Errorf("View() of an empty list = %q, want it to say nothing is here", got)
	}

	if help := NewOrderList(three(), true).Help(); !strings.Contains(help, "space toggle") || !strings.Contains(help, "move up") {
		t.Errorf("Help() = %q, want the move and toggle keys named", help)
	}

	if help := NewOrderList(three(), false).Help(); strings.Contains(help, "toggle") {
		t.Errorf("Help() of a list that does not toggle = %q, want no toggle key", help)
	}
}

func TestOrderListIgnoresAnEmptyListAndOtherMessages(t *testing.T) {
	l := NewOrderList(nil, true).Update(press(tea.KeyDown, "", 0)).Update(tea.WindowSizeMsg{Width: 80})

	if l.Len() != 0 || l.Cursor() != 0 {
		t.Errorf("an empty list moved: len %d, cursor %d", l.Len(), l.Cursor())
	}

	if _, ok := l.Selected(); ok {
		t.Error("Selected on an empty list reported an entry")
	}
}
