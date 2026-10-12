package main

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func ansiStrip(s string) string { return ansi.Strip(s) }

func rgb(c color.Color) (r, g, b uint8) {
	r16, g16, b16, _ := c.RGBA()
	return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)
}

func TestDimDarkensSetColorsAndGraysTheDefault(t *testing.T) {
	canvas := lipgloss.NewCanvas(3, 1)
	canvas.Compose(uv.NewStyledString("\x1b[38;2;200;100;50;48;2;20;40;60mA\x1b[mB"))
	dim(canvas)

	a := canvas.CellAt(0, 0)
	if r, g, b := rgb(a.Style.Fg); r != 100 || g != 50 || b != 25 {
		t.Errorf("fg = %d,%d,%d want 100,50,25", r, g, b)
	}
	if r, g, b := rgb(a.Style.Bg); r != 10 || g != 20 || b != 30 {
		t.Errorf("bg = %d,%d,%d want 10,20,30", r, g, b)
	}

	plain := canvas.CellAt(1, 0)
	if plain.Style.Fg != dimmedDefault {
		t.Errorf("default fg = %v want %v", plain.Style.Fg, dimmedDefault)
	}
	if plain.Style.Bg != nil {
		t.Errorf("default bg = %v want nil", plain.Style.Bg)
	}
}

func TestOverlayCentersTheDialogOverTheDimmedFrame(t *testing.T) {
	frame := strings.TrimSuffix(strings.Repeat("xxxxxxxxx\n", 5), "\n")
	dialog := "+-+\n|D|\n+-+"
	got := overlay(frame, dialog, 9, 5)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines want 5:\n%s", len(lines), got)
	}
	// The dialog's middle row sits on row 2, columns 3-5.
	if !strings.Contains(lines[2], "|D|") {
		t.Errorf("row 2 = %q, want the dialog in it", lines[2])
	}
	// The frame around it is drawn in the dimmed default color, the dialog is not.
	if !strings.Contains(lines[0], "\x1b[90m") {
		t.Errorf("row 0 = %q, want the gray SGR", lines[0])
	}
	if strings.Contains(lines[2], "\x1b[90m|D|") {
		t.Errorf("row 2 = %q, dialog should not be gray", lines[2])
	}
}

func TestPanePaintsTheMarginAndKeepsTheChildsColors(t *testing.T) {
	// A 4-wide block: a cell colored by the child, then default cells.
	block := "\x1b[48;2;1;2;3mA\x1b[mbcd"
	got := pane(block, 4, "t", false, colBg)
	canvas := lipgloss.NewCanvas(6, 3)
	canvas.Compose(uv.NewStyledString(got))

	if r, g, b := rgb(canvas.CellAt(1, 1).Style.Bg); r != 1 || g != 2 || b != 3 {
		t.Errorf("child bg = %d,%d,%d want 1,2,3", r, g, b)
	}
	for _, pos := range [][2]int{{0, 0}, {5, 0}, {0, 1}, {2, 1}, {5, 1}, {0, 2}, {5, 2}} {
		c := canvas.CellAt(pos[0], pos[1])
		if r, g, b := rgb(c.Style.Bg); r != 0x26 || g != 0x32 || b != 0x38 {
			t.Errorf("cell %v bg = %x,%x,%x want 26,32,38", pos, r, g, b)
		}
	}
	if canvas.CellAt(1, 0).Content != "t" {
		t.Errorf("title cell = %q want t", canvas.CellAt(1, 0).Content)
	}
}

func TestPaneWithNilBackgroundLeavesEverythingToTheTerminal(t *testing.T) {
	got := pane("abcd", 4, "t", false, nil)
	canvas := lipgloss.NewCanvas(6, 3)
	canvas.Compose(uv.NewStyledString(got))

	for y := 0; y < 3; y++ {
		for x := 0; x < 6; x++ {
			c := canvas.CellAt(x, y)
			if c.Style.Bg != nil {
				t.Errorf("cell %d,%d bg = %v, want nil", x, y, c.Style.Bg)
			}
			if c.Style.Fg != nil && !(y == 0 && x == 1) { // only the title is colored
				t.Errorf("cell %d,%d fg = %v, want nil", x, y, c.Style.Fg)
			}
		}
	}
}

func TestDialogIsBorderedOnItsFill(t *testing.T) {
	d := beadDialog{bead: bead{id: "cf-1", title: "T", description: "d"}}
	got := d.render()
	lines := strings.Split(got, "\n")
	first := strings.TrimSpace(ansiStrip(lines[0]))
	if !strings.HasPrefix(first, "▁") || !strings.HasSuffix(first, "▁") {
		t.Errorf("top = %q, want a hairline border", first)
	}
	canvas := lipgloss.NewCanvas(lipgloss.Width(got), len(lines))
	canvas.Compose(uv.NewStyledString(got))
	if bg := canvas.CellAt(1, 0).Style.Bg; bg != nil {
		t.Errorf("top border bg = %v want nil", bg)
	}
	if c := canvas.CellAt(0, 1); c.Style.Bg != nil || strings.TrimSpace(c.Content) != "" {
		t.Errorf("ring cell = %q bg %v, want blank on the terminal background", c.Content, c.Style.Bg)
	}
	if r, g, b := rgb(canvas.CellAt(1, 1).Style.Bg); r != 0x37 || g != 0x47 || b != 0x4f {
		t.Errorf("side border bg = %x,%x,%x want 37,47,4f", r, g, b)
	}
	if r, g, b := rgb(canvas.CellAt(2, 1).Style.Bg); r != 0x37 || g != 0x47 || b != 0x4f {
		t.Errorf("inner bg = %x,%x,%x want 37,47,4f", r, g, b)
	}
}

func TestOverlayRingTakesTheDimmedBackground(t *testing.T) {
	// A frame painted in its own color everywhere, the way OpenCode draws.
	row := strings.Repeat("\x1b[48;2;100;100;100mx", 70) + "\x1b[m"
	frame := strings.TrimSuffix(strings.Repeat(row+"\n", 14), "\n")
	d := beadDialog{bead: bead{id: "cf-1", title: "T", description: "d"}}
	out := overlay(frame, d.render(), 70, 14)
	canvas := lipgloss.NewCanvas(70, 14)
	canvas.Compose(uv.NewStyledString(out))

	dw, dh := lipgloss.Width(d.render()), lipgloss.Height(d.render())
	x, y := (70-dw)/2, (14-dh)/2
	for _, pos := range [][2]int{{x, y + 1}, {x + 1, y}, {x + 1, y + dh - 1}, {x + dw - 1, y + 1}} {
		c := canvas.CellAt(pos[0], pos[1])
		if r, g, b := rgb(c.Style.Bg); r != 50 || g != 50 || b != 50 {
			t.Errorf("ring cell %v bg = %d,%d,%d want 50,50,50 (the frame dimmed), content %q", pos, r, g, b, c.Content)
		}
	}
	if r, g, b := rgb(canvas.CellAt(x+2, y+2).Style.Bg); r != 0x37 || g != 0x47 || b != 0x4f {
		t.Errorf("fill = %x,%x,%x want 37,47,4f", r, g, b)
	}
}
