package presentation

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The first message a running program receives is its window size, before any screen but the menu
// has been opened; every list has to survive being resized then, including to a width of zero.
func TestEditorSurvivesAWindowSizeBeforeAnyScreenOpens(t *testing.T) {
	e := newTestEditor(t, &fakeConfig{shown: twoEntries()})

	e = send(t, e, tea.WindowSizeMsg{Width: 0, Height: 6})
	e = send(t, e, tea.WindowSizeMsg{Width: 100, Height: 30})

	if e.width != 100 || e.height != 30 {
		t.Errorf("size = %dx%d, want 100x30", e.width, e.height)
	}
}
