// Spike: run a terminal program inside a Bubble Tea v2 pane, beside a second
// pane owned by the host.
//
// The host spawns the command on a PTY (x/xpty), feeds the PTY's output into a
// virtual terminal (x/vt), renders the terminal's grid as the right pane, and
// sends the host's key, paste, and mouse events back through the emulator,
// which encodes them for the child. The emulator also answers the child's
// terminal queries (DA1, DSR, CPR) on the same pipe. The left pane is a menu
// the host draws itself, to see what layout and focus switching cost.
//
//	go run . [command [args...]]     default: $SHELL
//
// ctrl+] moves focus between the panes, showing the sidebar if it is hidden;
// ctrl+^ hides and shows the sidebar; a mouse click focuses the pane under it.
// ctrl+q quits the host and kills the child. Everything else goes to the
// focused pane.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"
)

const (
	statusRows = 1
	sidebarPad = 3 // cells before the sidebar's text, the margin included: under the wordmark's left edge; the bar starts a cell before
	borderCols = 2
	borderRows = 2
)

// sidebarWidth is the sidebar's content width; the border adds 2. It is wide
// enough for the wordmark in use with a cell of margin either side, and at
// least 24, so the child is resized when the wordmark changes.
func sidebarWidth() int {
	return max(24, lipgloss.Width(strings.TrimRight(logos[logoIndex].text, "\n"))+2)
}

// --- terminal pane -----------------------------------------------------------------------------

type outputMsg []byte
type exitedMsg struct{ err error }
type startedMsg struct {
	pty   xpty.Pty
	cmd   *exec.Cmd
	term  *vt.Emulator
	out   chan tea.Msg
	modes *childModes
}

// childModes is what the emulator's callbacks report about the child's
// terminal modes. The callbacks run inside Emulator.Write, on Bubble Tea's
// goroutine, so plain fields are safe.
type childModes struct {
	cursorVisible bool
	mouse         bool // the child turned on some mouse reporting mode
	log           *os.File
}

func (c *childModes) set(mode ansi.Mode, on bool) {
	switch mode {
	case ansi.ModeMouseX10, ansi.ModeMouseNormal, ansi.ModeMouseHighlight,
		ansi.ModeMouseButtonEvent, ansi.ModeMouseAnyEvent:
		c.mouse = on
	}
	if c.log != nil {
		state := "reset"
		if on {
			state = "set"
		}
		fmt.Fprintf(c.log, "%s %s %v\n", time.Now().Format("15:04:05.000"), state, mode)
	}
}

type errMsg struct{ err error }

// termPane owns the child process and its emulator. Its size is the inner
// size of the pane, without the border.
type termPane struct {
	argv          []string
	width, height int

	pty    xpty.Pty
	cmd    *exec.Cmd
	term   *vt.Emulator
	out    chan tea.Msg
	modes  *childModes
	exited bool
	err    error
	bytes  int

	// scroll is how many scrollback lines the view is shifted up by. Zero is
	// the live screen.
	scroll int

	// filter guards the emulator against x/ansi issue #848; see filter.go.
	// It also carries the child's window title.
	filter *c1Filter
}

// title is what the child last asked the terminal to show, or the command.
func (p termPane) title() string {
	if p.filter != nil && p.filter.Title != "" {
		return p.filter.Title
	}
	return strings.Join(p.argv, " ")
}

func (p termPane) started() bool { return p.term != nil }

// start spawns the child once the pane's size is known, so the PTY and the
// emulator begin at the right size.
func (p termPane) start() tea.Cmd {
	argv, w, h := p.argv, p.width, p.height
	return func() tea.Msg {
		pt, err := xpty.NewPty(w, h)
		if err != nil {
			return errMsg{fmt.Errorf("pty: %w", err)}
		}
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
		// xpty.Start wires the slave to stdio but does not make it the child's
		// controlling terminal; without this, ctrl+c and job control never reach
		// the child through the line discipline.
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
		if err := pt.Start(cmd); err != nil {
			_ = pt.Close()
			return errMsg{fmt.Errorf("start %s: %w", argv[0], err)}
		}

		term := vt.NewEmulator(w, h)
		modes := &childModes{cursorVisible: true}
		if path := os.Getenv("VTSPIKE_LOG"); path != "" {
			modes.log, _ = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		}
		term.SetCallbacks(vt.Callbacks{
			CursorVisibility: func(v bool) { modes.cursorVisible = v },
			EnableMode:       func(m ansi.Mode) { modes.set(m, true) },
			DisableMode:      func(m ansi.Mode) { modes.set(m, false) },
		})

		out := make(chan tea.Msg, 64)

		// Child output -> channel -> Update -> emulator. The emulator is only
		// touched on Bubble Tea's goroutine. The parent keeps the slave open
		// (xpty holds it), so the master never reports EOF; the exit is seen by
		// waiting on the process instead.
		// VTSPIKE_RAW=<prefix> records both directions: <prefix>.out is what
		// the child wrote, <prefix>.in is what the emulator sent it.
		var rawOut, rawIn *os.File
		if prefix := os.Getenv("VTSPIKE_RAW"); prefix != "" {
			rawOut, _ = os.Create(prefix + ".out")
			rawIn, _ = os.Create(prefix + ".in")
		}
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := pt.Read(buf)
				if n > 0 {
					chunk := make([]byte, n)
					copy(chunk, buf[:n])
					if rawOut != nil {
						_, _ = rawOut.Write(chunk)
					}
					out <- outputMsg(chunk)
				}
				if err != nil {
					return
				}
			}
		}()
		go func() {
			err := xpty.WaitProcess(context.Background(), cmd)
			time.Sleep(50 * time.Millisecond) // let the last output drain
			out <- exitedMsg{err}
		}()

		// Emulator-encoded keys and query replies -> child's stdin.
		go func() {
			var w io.Writer = pt
			if rawIn != nil {
				w = io.MultiWriter(pt, rawIn)
			}
			_, _ = io.Copy(w, term)
		}()

		return startedMsg{pty: pt, cmd: cmd, term: term, out: out, modes: modes}
	}
}

func waitOutput(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (p termPane) resize(w, h int) termPane {
	p.width, p.height = w, h
	if p.term != nil && (p.term.Width() != w || p.term.Height() != h) {
		p.term.Resize(w, h)
		_ = p.pty.Resize(w, h)
	}
	return p
}

// lockMods are lock-state flags, not chords.
const lockMods = uv.ModShift | uv.ModCapsLock | uv.ModNumLock | uv.ModScrollLock

func (p termPane) sendKey(msg tea.KeyPressMsg) {
	k := msg.Key()
	// vt.SendKey only emits a printable rune when Mod == 0, so shifted text
	// ("A", "!") goes straight through as text.
	if k.Text != "" && uv.KeyMod(k.Mod)&^lockMods == 0 {
		p.term.SendText(k.Text)
		return
	}
	p.term.SendKey(uv.KeyPressEvent{
		Text:        k.Text,
		Mod:         uv.KeyMod(k.Mod),
		Code:        k.Code,
		ShiftedCode: k.ShiftedCode,
		BaseCode:    k.BaseCode,
		IsRepeat:    k.IsRepeat,
	})
}

// sendMouse forwards a mouse event already translated to pane coordinates.
func (p termPane) sendMouse(msg tea.MouseMsg, x, y int) {
	ev := msg.Mouse()
	mod := uv.KeyMod(ev.Mod)
	switch msg.(type) {
	case tea.MouseClickMsg:
		p.term.SendMouse(uv.MouseClickEvent{X: x, Y: y, Button: ev.Button, Mod: mod})
	case tea.MouseReleaseMsg:
		p.term.SendMouse(uv.MouseReleaseEvent{X: x, Y: y, Button: ev.Button, Mod: mod})
	case tea.MouseMotionMsg:
		p.term.SendMouse(uv.MouseMotionEvent{X: x, Y: y, Button: ev.Button, Mod: mod})
	case tea.MouseWheelMsg:
		p.term.SendMouse(uv.MouseWheelEvent{X: x, Y: y, Button: ev.Button, Mod: mod})
	}
}

// scrollBy shifts the view into history by n lines (negative scrolls back
// toward the live screen). What that means depends on the child, the way it
// does in tmux: a child that reports the mouse gets the wheel itself and never
// reaches here; a child on the alternate screen has no history, so the wheel
// becomes arrow keys; otherwise the host shows the emulator's scrollback.
func (p termPane) scrollBy(n int) termPane {
	if p.term == nil {
		return p
	}
	if p.term.IsAltScreen() {
		code := uv.KeyUp
		if n < 0 {
			code, n = uv.KeyDown, -n
		}
		for i := 0; i < n; i++ {
			p.term.SendKey(uv.KeyPressEvent{Code: code})
		}
		return p
	}
	p.scroll += n
	if max := p.term.ScrollbackLen(); p.scroll > max {
		p.scroll = max
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
	return p
}

func (p termPane) content() string {
	switch {
	case p.err != nil:
		return "error: " + p.err.Error()
	case p.term == nil:
		return "starting…"
	case p.scroll > 0:
		return p.renderScrolled()
	default:
		return p.term.Render()
	}
}

// renderScrolled draws the pane shifted p.scroll lines into the scrollback:
// the top rows come from history and the rest from the live screen.
func (p termPane) renderScrolled() string {
	w, h := p.term.Width(), p.term.Height()
	n := p.term.ScrollbackLen()
	buf := uv.NewBuffer(w, h)
	for y := 0; y < h; y++ {
		idx := n - p.scroll + y
		for x := 0; x < w; x++ {
			var cell *uv.Cell
			if idx < n {
				cell = p.term.ScrollbackCellAt(x, idx)
			} else {
				cell = p.term.CellAt(x, idx-n)
			}
			if cell != nil {
				buf.SetCell(x, y, cell)
			}
		}
	}
	return buf.Render()
}

// cursor is the child's cursor in pane coordinates, or nil when hidden.
func (p termPane) cursor() *tea.Cursor {
	if p.term == nil || p.scroll > 0 || (p.modes != nil && !p.modes.cursorVisible) {
		return nil
	}
	pos := p.term.CursorPosition()
	return &tea.Cursor{Position: tea.Position{X: pos.X, Y: pos.Y}, Shape: tea.CursorBlock, Blink: true}
}

// fitBlock makes s exactly w cells wide and h rows tall: each line is cut or
// padded to w, and rows are added or dropped from the bottom to reach h. It is
// used in place of lipgloss's Width and Height, which word-wrap a line they
// measure as too wide; the emulator's rows must never wrap.
func fitBlock(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i, l := range lines {
		l = ansi.Truncate(l, w, "")
		if pad := w - ansi.StringWidth(l); pad > 0 {
			l += "\x1b[0m" + strings.Repeat(" ", pad)
		}
		lines[i] = l + "\x1b[0m"
	}
	return strings.Join(lines, "\n")
}

// --- sidebar ------------------------------------------------------------------------------------

// bead is a stand-in for a Beads issue, with the fields a dialog would show.
type bead struct {
	id, title, status, priority string
	deps                        []string
	description                 string
}

// menuItems are the rows of the sidebar's menu.
var menuItems = []string{"Beads/Issues", "About"}

// sidebar is the host's own pane: a menu with a cursor, standing in for
// whatever a real host would show beside the harness.
type sidebar struct {
	menu  int // cursor in the menu
	items []bead
}

// update handles a key while the sidebar has focus. It returns the sidebar and
// the dialog to open, if the key opened one.
func (s sidebar) update(msg tea.KeyPressMsg) (sidebar, dialog) {
	switch msg.String() {
	case "up":
		if s.menu > 0 {
			s.menu--
		}
	case "down":
		if s.menu < len(menuItems)-1 {
			s.menu++
		}
	case "enter", "right":
		return s, s.open(s.menu)
	}
	return s, nil
}

// open returns the dialog for the menu item at i: the bead list for
// Beads/Issues, the About dialog for About.
func (s sidebar) open(i int) dialog {
	switch menuItems[i] {
	case "Beads/Issues":
		return &beadList{items: s.items}
	case "About":
		return aboutDialog{}
	}
	return nil
}

// The wordmarks, CODEFALL on one line in block or box-drawing glyphs, each
// uncolored and colored here from the palette. One sits above the menu. While
// the sidebar has focus, ctrl+l enters and leaves logo mode, in which the left
// arrow cycles the marks, the right arrow centers the mark or sets it back
// left, and the down arrow darkens the glow or lightens it.
//
//go:embed logo-thin.ans
var logoThin string

//go:embed logo-thin-stretched.ans
var logoThinStretched string

//go:embed logo-thin-sans.ans
var logoThinSans string

//go:embed logo-thin-stretched-sans.ans
var logoThinStretchedSans string

// logoArt is a wordmark. rows, when set, colors the art's rows in turn from
// the palette, the art carrying no colors of its own. band draws the glow as
// stripes instead of the halo: the inner tint behind the rows edge to edge,
// and a half-row rim of the outer tint above and below, ▄ and ▀ on the
// sidebar's own fill.
type logoArt struct {
	name, text string
	rows       []color.Color
	band       bool
}

// logos is the thin marks, narrowest first, each cyan with its bottom row in
// the deep teal, on the banded glow.
var logos = func() []logoArt {
	marks := []logoArt{
		{name: "thin", text: logoThin},
		{name: "thin sans", text: logoThinSans},
		{name: "thin stretched", text: logoThinStretched},
		{name: "thin stretched sans", text: logoThinStretchedSans},
	}
	for i := range marks {
		marks[i].rows, marks[i].band = []color.Color{colAccent, colAccent, colTealDark}, true
	}
	return marks
}()

// logoIndex is the wordmark in use. logoDark puts the wordmark on the dark
// glow. logoCenter centers the wordmark in the sidebar instead of setting it
// a cell in from the left. Logo mode changes them; thin sans, dark and
// centered, is the default.
var (
	logoIndex  = 1
	logoDark   = true
	logoCenter = true
)

// logoState is the wordmark's name for the status bar, with the glow and the
// placement noted when they are the dark one and centered.
func logoState() string {
	name := logos[logoIndex].name
	if logoDark {
		name += " · dark"
	}
	if logoCenter {
		name += " · centered"
	}
	return name
}

// The glow is the sidebar fill tinted toward the logo's teal: a quarter of the
// way for cells touching a glyph and for every cell between a row's first and
// last glyph, the spacing between letters and the open sides of the letters
// included, an eighth for the ring outside that. The dark pair is a touch
// less, three sixteenths and a sixteenth.
var (
	glowInner     = lipgloss.Color("#25484e")
	glowOuter     = lipgloss.Color("#21373c")
	glowInnerDark = lipgloss.Color("#234046")
	glowOuterDark = lipgloss.Color("#202f35")
)

// logo renders the wordmark in use, a cell in from the left or centered, with a dark glow:
// every cell without a background of its own takes a tint by its distance from
// the nearest glyph, so the glyphs sit in a halo and the half blocks show it in
// their empty half. The art gets a blank row above and below for the glow to
// spill into, or for the bands of a banded wordmark.
func logo(width int) string {
	l := logos[logoIndex]
	art := strings.TrimRight(l.text, "\n")
	logoRows := lipgloss.Height(art) + 2
	canvas := lipgloss.NewCanvas(width, logoRows)
	indent := " "   // the margin
	if logoCenter { // the block as a whole, so ragged rows keep their alignment
		indent = strings.Repeat(" ", max(1, (width-lipgloss.Width(art))/2))
	}
	canvas.Compose(lipgloss.NewLayer("\n" + indent + strings.ReplaceAll(art, "\n", "\n"+indent)))
	for y := 1; len(l.rows) > 0 && y < logoRows-1; y++ {
		for x := 0; x < width; x++ {
			if cell := canvas.CellAt(x, y); cell != nil && strings.TrimSpace(cell.Content) != "" {
				cell.Style.Fg = l.rows[(y-1)%len(l.rows)]
			}
		}
	}
	inner, outer := glowInner, glowOuter
	if logoDark {
		inner, outer = glowInnerDark, glowOuterDark
	}
	if l.band {
		for y := 0; y < logoRows; y++ {
			for x := 0; x < width; x++ {
				cell := canvas.CellAt(x, y)
				if cell == nil || cell.Width == 0 {
					continue
				}
				switch {
				case y == 0:
					cell.Content, cell.Style.Fg = "▄", outer
				case y == logoRows-1:
					cell.Content, cell.Style.Fg = "▀", outer
				default:
					cell.Style.Bg = inner
				}
			}
		}
		return canvas.Render()
	}
	// A glyph is any drawn cell, the shaded blocks included. The map is taken
	// before tinting, or the tint itself would count as drawn.
	drawn := make([][]bool, logoRows)
	for y := range drawn {
		drawn[y] = make([]bool, width)
		for x := range drawn[y] {
			c := canvas.CellAt(x, y)
			drawn[y][x] = c != nil && c.Width > 0 && (strings.TrimSpace(c.Content) != "" || c.Style.Bg != nil)
		}
	}
	glyph := func(x, y int) bool {
		return y >= 0 && y < logoRows && x >= 0 && x < width && drawn[y][x]
	}
	for y := 0; y < logoRows; y++ {
		first, last := width, -1
		for x := range drawn[y] {
			if drawn[y][x] {
				first, last = min(first, x), x
			}
		}
		for x := 0; x < width; x++ {
			cell := canvas.CellAt(x, y)
			if cell == nil || cell.Width == 0 || cell.Style.Bg != nil {
				continue
			}
			switch d := nearest(glyph, x, y, 2); {
			case d <= 1, first <= x && x <= last:
				cell.Style.Bg = inner
			case d == 2:
				cell.Style.Bg = outer
			}
		}
	}
	return canvas.Render()
}

// nearest is the distance from x, y to the closest cell for which glyph is
// true, up to limit, with a row counting as two columns because a cell is
// about twice as tall as it is wide; it returns limit+1 when none is that
// close.
func nearest(glyph func(x, y int) bool, x, y, limit int) int {
	best := limit + 1
	for dy := -limit / 2; dy <= limit/2; dy++ {
		for dx := -limit; dx <= limit; dx++ {
			if d := max(abs(dx), 2*abs(dy)); d < best && glyph(x+dx, y+dy) {
				best = d
			}
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// helpItem is one entry of a key help row.
type helpItem struct{ key, desc string }

var (
	menuHelp  = []helpItem{{"↑↓", "move"}, {"enter", "confirm"}}
	listHelp  = []helpItem{{"↑↓", "choose"}, {"enter", "confirm"}, {"esc", "cancel"}}
	beadHelp  = []helpItem{{"←→", "switch"}, {"enter", "confirm"}, {"esc", "cancel"}}
	aboutHelp = []helpItem{{"enter", "confirm"}, {"esc", "cancel"}}
)

// help renders the items the way the Charm help bubble does: key, a space,
// its description, the pairs separated by a dot.
func help(items []helpItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = helpKey.Render(it.key) + " " + helpDesc.Render(it.desc)
	}
	return strings.Join(parts, helpDesc.Render(" • "))
}

// selected draws a row as the selection in the given style: the text on the
// highlight, padded with plain spaces to the full width first, because
// fitBlock resets the style ahead of its padding and would end the highlight
// at the text.
func selected(line string, width int, style lipgloss.Style) string {
	return style.Render(line + strings.Repeat(" ", max(width-ansi.StringWidth(line), 0)))
}

// content draws the sidebar as a block the full size of its pane, margin
// included, so a selected row runs edge to edge; the text sits sidebarPad
// cells in, under the wordmark's left edge. The top row is blank, the
// wordmark standing for a title. The wordmark's name sits faint on the
// second-to-last row while they are under comparison. While the sidebar has
// focus the cursor is highlighted and the key help sits on the bottom row, or
// an item a row when the sidebar is too narrow for one row; without it the
// text is muted and neither is drawn.
func (s sidebar) content(width, height int, focused bool) string {
	rows := []string{""} // the title row
	rows = append(rows, strings.Split(logo(width), "\n")...)
	rows = append(rows, "") // a blank row between the glow and the list
	text := lipgloss.NewStyle()
	if !focused {
		text = text.Foreground(colMuted)
	}
	lead := strings.Repeat(" ", sidebarPad)
	for i, it := range menuItems {
		line := lead + text.Render(it)
		if focused && i == s.menu {
			line = selectRow(it, width, sidebarPad-1)
		}
		rows = append(rows, line)
	}
	var foot []string
	if focused {
		foot = []string{help(menuHelp)}
		if lipgloss.Width(foot[0]) > width-sidebarPad {
			foot = foot[:0]
			for _, it := range menuHelp {
				foot = append(foot, help([]helpItem{it}))
			}
		}
	}
	foot = append([]string{helpDesc.Render(logos[logoIndex].name)}, foot...)
	block := fitBlock(strings.Join(rows, "\n"), width, height-len(foot))
	for _, f := range foot {
		block += "\n" + lead + f
	}
	return fitBlock(block, width, height)
}

// --- dialogs -----------------------------------------------------------------------------------

// dialog is a modal composed over the dimmed frame. While one is open it takes
// every key.
type dialog interface {
	// render draws the dialog for a window of the given size.
	render(winW, winH int) string
	// update handles a key. It returns the dialog to show next, nil once the
	// dialog has closed, and the bead to implement when Implement was chosen.
	update(msg tea.KeyPressMsg) (dialog, *bead)
}

// dialogWidth is the inner width of a dialog, hairline to hairline: 70 cells
// like Crush's command palette, or what the window leaves after the ring and a
// cell of margin on each side.
func dialogWidth(winW int) int {
	return max(min(70, winW-6), 20)
}

// dialogPad is the cells between a hairline and the text. A selected row's
// bar starts one cell in from the hairline, with the cursor in that cell.
const dialogPad = 2

// hairline is a border of one-eighth blocks, every rule on the inner edge of
// its cell: ▁ across the top, ▔ across the bottom, ▕ on the left, and ▏ on
// the right, with the corner cells blank. The top rule ends at the inner
// corner of the corner cell, exactly where the side rule in the cell below
// begins, so the corners close without a joining glyph and nothing overshoots.
// Every glyph is a standard Block Element; the flush corners (U+1FB7C..7F)
// are missing from some fonts and are not used.
var hairline = lipgloss.Border{
	Top: "▁", Bottom: "▔", Left: "▕", Right: "▏",
	TopLeft: " ", TopRight: " ", BottomLeft: " ", BottomRight: " ",
}

var (
	dialogLabel  = lipgloss.NewStyle().Foreground(colMuted)
	buttonStyle  = lipgloss.NewStyle().Padding(0, 2).Foreground(colMuted)
	buttonActive = lipgloss.NewStyle().Padding(0, 2).Background(colGold).Foreground(colBgDark).Bold(true)
	// The dialog sits in a ring of the terminal's own background: the border
	// cells carry no fill, so each hairline lies on the boundary between the
	// fill and the ring, and a blank column outside each side widens the ring
	// there to about what the border rows give above and below, a cell being
	// twice as tall as it is wide.
	dialogBorder = lipgloss.NewStyle().Border(hairline).BorderForeground(colAccent).Margin(0, 1)
)

// gradientBorder recolors a rendered dialog's hairlines in the title's
// gradient, run from the top-left corner to the bottom-right along both
// ways round: across the top and down the right side, and down the left side
// and along the bottom. The two runs are the same length, so they meet at
// the bottom-right corner on the last stop. The margin columns outside the
// sides are left alone. With the title bar on, the row under the top
// hairline takes the gradient as its background from hairline to hairline
// and its text the dark fill color. The top hairline then runs the bar's
// own gradient, the whole of it across the top, and so does a hairline of
// ▔ on the row under the bar, so the bar is padded an eighth above and
// below: the top hairline sits on the bottom edge of its cell and the ▔ on
// the top edge of its, both flush with the bar, and both meeting the side
// rules, which sit on the inner edge of their cells. The right side takes
// the gradient's last stop throughout, so the frame agrees with the bar at
// both ends; the left side and the bottom run the corner-to-corner path as
// in rule mode.
func gradientBorder(rendered string) string {
	g := gradients[gradientIndex]
	w, h := lipgloss.Width(rendered), lipgloss.Height(rendered)
	left, right := 1, w-2 // the margin takes column 0 and the last
	n := right - left + 1 // cells across the top or bottom
	if n < 2 || h < 2 {
		return rendered
	}
	colors := lipgloss.Blend1D(n+h-1, g.from, g.to)
	canvas := lipgloss.NewCanvas(w, h)
	canvas.Compose(lipgloss.NewLayer(rendered))
	tint := func(x, y, i int) {
		if cell := canvas.CellAt(x, y); cell != nil && cell.Width > 0 {
			cell.Style.Fg = colors[i]
		}
	}
	for x := left; x <= right; x++ {
		tint(x, 0, x-left)       // the top, left to right
		tint(x, h-1, h-1+x-left) // the bottom, left to right, after the left side
	}
	for y := 1; y < h; y++ {
		tint(right, y, n-1+y) // the right side, downward, after the top
		tint(left, y, y)      // the left side, downward
	}
	if titleBar && h > 3 {
		bar := lipgloss.Blend1D(n, g.from, g.to)
		for x := left + 1; x < right; x++ { // inside the side rules
			c := bar[x-left]
			if cell := canvas.CellAt(x, 0); cell != nil && cell.Width > 0 {
				cell.Style.Fg = c
			}
			if cell := canvas.CellAt(x, 1); cell != nil && cell.Width > 0 {
				cell.Style.Bg = c
				if strings.TrimSpace(cell.Content) != "" {
					cell.Style.Fg = colBgDark
				}
			}
			if cell := canvas.CellAt(x, 2); cell != nil && cell.Width > 0 {
				cell.Content, cell.Style.Fg = "▔", c
			}
		}
		for y := 1; y < h-1; y++ {
			if cell := canvas.CellAt(right, y); cell != nil {
				cell.Style.Fg = g.to
			}
		}
	}
	// The canvas trims trailing blanks, which takes the right margin column
	// off; put it back so the dialog keeps its width and its place.
	lines := strings.Split(canvas.Render(), "\n")
	for i, l := range lines {
		lines[i] = l + strings.Repeat(" ", max(0, w-lipgloss.Width(l)))
	}
	return strings.Join(lines, "\n")
}

// --- selection ---------------------------------------------------------------------------------

// selectedRow is the selected row of a list, in the sidebar or a dialog: a
// blue-gray bar a step above the dialog fill, inside the pane's padding rather
// than under it, with a gold ▎ as its first cell for the cursor. The gold
// stays one cell wide and means what it means on a button: this is the one
// enter acts on.
var (
	selectedRow = lipgloss.NewStyle().Background(colBgLighter).Foreground(colText).Bold(true)
	marker      = lipgloss.NewStyle().Foreground(colGold).Background(colBgLighter).Render("▎")
)

// selectRow draws text as the selected row of a list width wide: inset cells
// of the pane's own fill on each side, then the bar, whose first cell is the
// marker and whose text starts where the other rows' text does.
func selectRow(text string, width, inset int) string {
	side := strings.Repeat(" ", inset)
	return side + marker + selected(text, width-2*inset-1, selectedRow) + side
}

// --- dialog title bar --------------------------------------------------------------------------

// gradient is the pair of stops for the rule under a dialog's title, which
// Blend1D spreads across the row one color per cell; the title takes the
// first stop so the rule continues from it, and the border runs the same
// stops corner to corner. Several are in to compare, cycled
// with tab while a dialog is open. A fade ends in another hue, never in the
// fill gray: a blend into the neutral passes through desaturated mud and
// reads as the same color dimming.
type gradient struct {
	name     string
	from, to color.Color
}

var gradients = []gradient{
	{"gold → teal", colGold, colAccent},
	{"gold → purple", colGold, colPurple},
	{"gold → violet", colGold, colViolet},
	{"gold → red", colGold, colPink},
	{"gold → green", colGold, colGreen},
	{"gold → deep teal", colGold, colTealDark},
	{"teal → purple", colAccent, colPurple},
}

// gradientIndex is the gradient in use; tab cycles it while a dialog is open.
var gradientIndex int

// titleBar sets the title as dark text on a bar of the gradient, flush with
// the hairlines, in place of the title and its rule; shift+tab toggles it
// while a dialog is open.
var titleBar bool

// titleRows draws a dialog's title, and under it a hairline of ▔ across the
// full width in the gradient. The rule is the top eighth of the next row
// because a cell holds a letter or a block, never both, so a rule under the
// title's own letters cannot share its row.
func titleRows(title string, w int) []string {
	g := gradients[gradientIndex]
	text := lipgloss.NewStyle().Foreground(g.from).Bold(true).Render(title)
	n := w - 2*dialogPad
	if n < 2 {
		return []string{pad(text)}
	}
	colors := lipgloss.Blend1D(n, g.from, g.to)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(lipgloss.NewStyle().Foreground(colors[i]).Render("▔"))
	}
	return []string{pad(text), pad(b.String())}
}

// dialogFrame sets the title row, then the lines, already w wide at most, in
// the shaded pane with its border around it. The pane's own title row is left
// blank so the top is padded like the bottom, and the help row is the last
// line.
func dialogFrame(title string, lines []string, w int, items []helpItem) string {
	paneTitle := ""
	if titleBar { // the bar takes the pane's title row, then a row for its lower hairline
		paneTitle = pad(lipgloss.NewStyle().Bold(true).Render(title))
		lines = append([]string{""}, lines...)
	} else {
		lines = append(titleRows(title, w), lines...)
	}
	lines = append(lines, "", pad(help(items)))
	body := fitBlock(strings.Join(lines, "\n"), w, len(lines))
	return gradientBorder(dialogBorder.Render(pane(body, w, paneTitle, true, colBgLight)))
}

// pad indents a line by the dialog's padding.
func pad(line string) string { return strings.Repeat(" ", dialogPad) + line }

// wrap word-wraps text to the width inside the padding and pads each line.
func wrap(text string, w int) []string {
	lines := strings.Split(lipgloss.NewStyle().Width(w-2*dialogPad).Render(text), "\n")
	for i := range lines {
		lines[i] = pad(lines[i])
	}
	return lines
}

// --- bead list ---------------------------------------------------------------------------------

// beadList is the dialog Beads/Issues opens: the beads to choose from, with a
// cursor. enter opens the bead's own dialog, which comes back here on cancel.
type beadList struct {
	items  []bead
	cursor int
}

func (d *beadList) render(winW, winH int) string {
	w := dialogWidth(winW)
	// The rows the window leaves for the list: the ring, the pane's top and
	// bottom rows, the title and its blank row, the help row and its blank.
	visible := max(min(len(d.items), winH-8), 1)
	start := 0
	if d.cursor >= visible {
		start = d.cursor - visible + 1
	}
	lines := []string{""}
	for i := start; i < len(d.items) && i < start+visible; i++ {
		it := d.items[i]
		line := pad(it.id + "  " + it.title)
		if i == d.cursor {
			line = selectRow(it.id+"  "+it.title, w, dialogPad-1)
		}
		lines = append(lines, line)
	}
	return dialogFrame("Beads/Issues", lines, w, listHelp)
}

func (d *beadList) update(msg tea.KeyPressMsg) (dialog, *bead) {
	switch msg.String() {
	case "up":
		if d.cursor > 0 {
			d.cursor--
		}
	case "down":
		if d.cursor < len(d.items)-1 {
			d.cursor++
		}
	case "esc", "q":
		return nil, nil
	case "enter":
		if d.cursor < len(d.items) {
			return &beadDialog{bead: d.items[d.cursor], back: d}, nil
		}
	}
	return d, nil
}

// --- bead dialog -------------------------------------------------------------------------------

// beadDialog is the modal opened on a bead: its details and two buttons.
// Implement types a slash command into the child; Cancel returns to the list
// it was opened from.
type beadDialog struct {
	bead   bead
	button int    // 0 implement, 1 cancel
	back   dialog // what cancel returns to, or nil
}

func (d beadDialog) render(winW, winH int) string {
	b := d.bead
	w := dialogWidth(winW)
	row := func(label, value string) string {
		return pad(dialogLabel.Render(fmt.Sprintf("%-9s", label)) + value)
	}
	lines := []string{
		pad(lipgloss.NewStyle().Bold(true).Render(b.title)),
		"",
		row("status", b.status),
		row("priority", b.priority),
		row("deps", strings.Join(b.deps, ", ")),
		"",
	}
	lines = append(lines, wrap(b.description, w)...)
	impl, cancel := buttonStyle, buttonStyle
	if d.button == 0 {
		impl = buttonActive
	} else {
		cancel = buttonActive
	}
	buttons := impl.Render("Implement") + "  " + cancel.Render("Cancel")
	lines = append(lines, "", lipgloss.PlaceHorizontal(w-dialogPad, lipgloss.Right, buttons))
	return dialogFrame(b.id, lines, w, beadHelp)
}

func (d *beadDialog) update(msg tea.KeyPressMsg) (dialog, *bead) {
	switch msg.String() {
	case "left", "shift+tab":
		d.button = 0
	case "right", "tab":
		d.button = 1
	case "esc", "q":
		return d.back, nil
	case "enter":
		if d.button == 0 {
			return nil, &d.bead
		}
		return d.back, nil
	}
	return d, nil
}

// --- about dialog ------------------------------------------------------------------------------

// aboutDialog is the modal the About item opens: what the spike is, and the
// command it is running, over an OK button.
type aboutDialog struct{}

// childCommand is the command line the child was started with, for About.
var childCommand string

func (aboutDialog) render(winW, winH int) string {
	w := dialogWidth(winW)
	lines := []string{
		pad(lipgloss.NewStyle().Bold(true).Render("VT spike")),
		"",
	}
	lines = append(lines, wrap("A terminal program inside a Bubble Tea pane, beside a pane the host draws itself: "+
		"x/xpty for the PTY, x/vt for the terminal, keys and mouse re-encoded through the emulator.", w)...)
	lines = append(lines, "")
	lines = append(lines, wrap(dialogLabel.Render("child    ")+childCommand, w)...)
	lines = append(lines, "", lipgloss.PlaceHorizontal(w-dialogPad, lipgloss.Right, buttonActive.Render("OK")))
	return dialogFrame("Codefall", lines, w, aboutHelp)
}

func (d aboutDialog) update(msg tea.KeyPressMsg) (dialog, *bead) {
	switch msg.String() {
	case "esc", "q", "enter":
		return nil, nil
	}
	return d, nil
}

// dimAmount is how far the frame's colors move toward black under an open
// dialog: 0 leaves them alone, 1 blacks them out.
const dimAmount = 0.5

// dimmedDefault stands in for the terminal's default foreground, which has no
// value to darken.
var dimmedDefault = lipgloss.Color("8")

// dim darkens every cell on the canvas so a layer composed over it afterwards
// stands out. Colors the frame set, pane backgrounds included, move toward
// black; a cell still on the default foreground becomes gray, and one still on
// the default background is left to the terminal.
func dim(c *lipgloss.Canvas) {
	b := c.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := c.CellAt(x, y)
			if cell == nil {
				continue
			}
			if cell.Style.Fg == nil {
				cell.Style.Fg = dimmedDefault
			} else {
				cell.Style.Fg = lipgloss.Darken(cell.Style.Fg, dimAmount)
			}
			if cell.Style.Bg != nil {
				cell.Style.Bg = lipgloss.Darken(cell.Style.Bg, dimAmount)
			}
		}
	}
}

// overlay centers the dialog over a dimmed copy of the frame. The frame is a
// plain layer so it lands at the origin; the dialog goes through a compositor,
// which is what honors a layer's X and Y. Dialog cells without a background
// of their own, the blank ring and the top and bottom rules, are transparent:
// they take the dimmed cell's background, so the ring looks the same over a
// child that paints its own background as over one that does not.
func overlay(frame, dialog string, width, height int) string {
	dw, dh := lipgloss.Width(dialog), lipgloss.Height(dialog)
	x, y := max((width-dw)/2, 0), max((height-dh)/2, 0)
	canvas := lipgloss.NewCanvas(width, height)
	canvas.Compose(lipgloss.NewLayer(frame))
	dim(canvas)
	under := make([][]color.Color, dh) // the dimmed backgrounds the dialog will cover
	for cy := range under {
		under[cy] = make([]color.Color, dw)
		for cx := range under[cy] {
			if c := canvas.CellAt(x+cx, y+cy); c != nil {
				under[cy][cx] = c.Style.Bg
			}
		}
	}
	canvas.Compose(lipgloss.NewCompositor(
		lipgloss.NewLayer(dialog).X(x).Y(y),
	))
	for cy := 0; cy < dh; cy++ {
		for cx := 0; cx < dw; cx++ {
			cell := canvas.CellAt(x+cx, y+cy)
			if cell == nil || cell.Width == 0 || cell.Style.Bg != nil {
				continue
			}
			cell.Style.Bg = under[cy][cx]
		}
	}
	return canvas.Render()
}

// --- host --------------------------------------------------------------------------------------

type focus int

const (
	focusSidebar focus = iota
	focusTerm
)

type model struct {
	width, height int
	focus         focus
	side          sidebar
	sideHidden    bool // the sidebar is hidden and the child has the width
	term          termPane
	dialog        dialog // open modal, or nil
	logoMode      bool   // the arrows change the wordmark instead of the menu
}

// The palette is OpenCode's Material theme (dark), whose panes are blocks of
// three blue-grays rather than lines: packages/tui/src/theme/assets/material.json.
var (
	colBgDark    = lipgloss.Color("#1e272c") // backgroundPanel: the sidebar
	colBg        = lipgloss.Color("#263238") // background: unused, the terminal pane stays on the terminal's own
	colBgLight   = lipgloss.Color("#37474f") // backgroundElement: status bar, dialog, selection
	colText      = lipgloss.Color("#eeffff")
	colMuted     = lipgloss.Color("#546e7a")
	colMutedDark = lipgloss.Color("#415a64") // the help descriptions, a step under colMuted
	colPrimary   = lipgloss.Color("#82aaff")
	colAccent    = lipgloss.Color("#89ddff")
	// The rest of the Material accents, for the title gradients.
	colPurple   = lipgloss.Color("#c792ea")
	colPink     = lipgloss.Color("#f07178")
	colOrange   = lipgloss.Color("#f78c6c")
	colTealDark = lipgloss.Color("#39adb5") // the wordmark's shading
	colViolet   = lipgloss.Color("#bb80b3")
	colGreen    = lipgloss.Color("#c3e88d")
	// The website's accent: the one warm color, for what is about to be acted
	// on. It is the complement of the wordmark's teal, and it takes the roles
	// the Material blue had.
	colGold      = lipgloss.Color("#f2c25a")
	colBgLighter = lipgloss.Color("#455a64") // the selected row, a step above the dialog fill
)

var (
	statusStyle  = lipgloss.NewStyle().Background(colBgLight).Foreground(colText)
	focusedTitle = lipgloss.NewStyle().Foreground(colGold).Bold(true)
	blurredTitle = lipgloss.NewStyle().Foreground(colMuted)
	helpKey      = lipgloss.NewStyle().Foreground(colMuted)
	helpDesc     = lipgloss.NewStyle().Foreground(colMutedDark)
)

// paint fills in what a block leaves to the terminal's defaults: every cell
// without a foreground gets fg, every cell without a background gets bg, and
// the canvas pads the block to w by h so the whole rectangle is one color.
// Colors the block set itself are kept. A nil bg paints nothing: the whole
// rectangle is left to the terminal.
func paint(block string, w, h int, fg, bg color.Color) string {
	canvas := lipgloss.NewCanvas(w, h)
	canvas.Compose(lipgloss.NewLayer(block))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cell := canvas.CellAt(x, y)
			if cell == nil || cell.Width == 0 || bg == nil { // nil, the tail of a wide character, or unpainted
				continue
			}
			if cell.Style.Fg == nil {
				cell.Style.Fg = fg
			}
			if cell.Style.Bg == nil {
				cell.Style.Bg = bg
			}
		}
	}
	return canvas.Render()
}

// pane sets a block that is already exactly w wide into a colored rectangle one
// cell larger on each side, the way OpenCode's panes are blocks rather than
// boxes: the title sits in the top row, the margin is the pane's color. A nil
// bg leaves the pane on the terminal's default colors, for the child, whose
// palette was chosen against them.
func pane(block string, w int, title string, focused bool, bg color.Color) string {
	ttl := blurredTitle
	if focused {
		ttl = focusedTitle
	}
	if w < 2 {
		title = ""
	} else {
		title = ansi.Truncate(title, w-1, "…")
	}
	var b strings.Builder
	b.WriteString(" " + ttl.Render(title))
	rows := 1
	for _, line := range strings.Split(block, "\n") {
		b.WriteString("\n " + line)
		rows++
	}
	return paint(b.String(), w+borderCols, rows+1, colText, bg)
}

func (m model) Init() tea.Cmd { return nil }

// sideCols is the width the sidebar takes from the window: its content and
// its margin, or nothing while it is hidden.
func (m model) sideCols() int {
	if m.sideHidden {
		return 0
	}
	return sidebarWidth() + borderCols
}

// termInner is the inner size of the terminal pane for the current window.
func (m model) termInner() (int, int) {
	w := m.width - m.sideCols() - borderCols
	h := m.height - statusRows - borderRows
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// termOrigin is where the terminal pane's inner top-left cell sits on screen.
func (m model) termOrigin() (int, int) {
	return m.sideCols() + 1, statusRows + 1
}

// isToggleKey reports whether the key hides or shows the sidebar: ctrl+^,
// which no harness binds. Terminals send it for ctrl+6 as well, and one with
// the kitty protocol on names that key by its digit, so both spellings count.
func isToggleKey(key string) bool {
	switch key {
	case "ctrl+^", "ctrl+6", "ctrl+shift+6":
		return true
	}
	return false
}

// toggleSidebar hides or shows the sidebar, gives the child the new width, and
// moves focus to the child when the pane it was on is gone.
func (m model) toggleSidebar() model {
	m.sideHidden = !m.sideHidden
	w, h := m.termInner()
	m.term = m.term.resize(w, h)
	if m.sideHidden {
		m = m.setFocus(focusTerm)
	}
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w, h := m.termInner()
		m.term = m.term.resize(w, h)
		if !m.term.started() && m.term.pty == nil && m.term.err == nil {
			return m, m.term.start()
		}
		return m, nil

	case startedMsg:
		m.term.pty, m.term.cmd, m.term.term, m.term.out, m.term.modes = msg.pty, msg.cmd, msg.term, msg.out, msg.modes
		// The window may have changed between start and now.
		w, h := m.termInner()
		m.term = m.term.resize(w, h)
		if m.focus == focusTerm {
			m.term.term.Focus()
		}
		return m, waitOutput(m.term.out)

	case outputMsg:
		m.term.bytes += len(msg)
		if m.term.filter == nil {
			m.term.filter = &c1Filter{}
		}
		_, _ = m.term.term.Write(m.term.filter.Push(msg))
		return m, waitOutput(m.term.out)

	case exitedMsg:
		m.term.exited, m.term.err = true, msg.err
		return m, tea.Quit

	case errMsg:
		m.term.err = msg.err
		return m, tea.Quit

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}
		if m.dialog != nil {
			switch msg.String() {
			case "tab": // try the next gradient
				gradientIndex = (gradientIndex + 1) % len(gradients)
				return m, nil
			case "shift+tab": // try the title on a bar
				titleBar = !titleBar
				return m, nil
			}
			var implement *bead
			m.dialog, implement = m.dialog.update(msg)
			if implement != nil && m.term.started() {
				// Type the command into the child as if at the keyboard,
				// without enter, and hand it the focus so the user sends it.
				m.term.term.SendText("/codefall-implement " + implement.id)
				m = m.setFocus(focusTerm)
			}
			return m, nil
		}
		if isToggleKey(msg.String()) {
			return m.toggleSidebar(), nil
		}
		if msg.String() == "ctrl+l" && m.focus == focusSidebar {
			m.logoMode = !m.logoMode
			return m, nil
		}
		if m.logoMode && m.focus == focusSidebar {
			switch msg.String() {
			case "left": // try the next wordmark
				logoIndex = (logoIndex + 1) % len(logos)
				w, h := m.termInner() // the sidebar is sized to the wordmark
				m.term = m.term.resize(w, h)
			case "right": // try the other placement
				logoCenter = !logoCenter
			case "down": // try the other glow
				logoDark = !logoDark
			}
			return m, nil
		}
		if msg.String() == "ctrl+]" {
			if m.sideHidden {
				return m.toggleSidebar().setFocus(focusSidebar), nil
			}
			return m.setFocus(1 - m.focus), nil
		}
		switch m.focus {
		case focusSidebar:
			m.side, m.dialog = m.side.update(msg)
		case focusTerm:
			if !m.term.started() {
				break
			}
			_, h := m.termInner()
			switch msg.String() {
			case "shift+pgup":
				m.term = m.term.scrollBy(h - 1)
			case "shift+pgdown":
				m.term = m.term.scrollBy(-(h - 1))
			case "esc":
				if m.term.scroll > 0 {
					m.term.scroll = 0 // leave scroll mode; the child does not see this esc
				} else {
					m.term.sendKey(msg)
				}
			default:
				m.term.scroll = 0 // typing returns to the live screen
				m.term.sendKey(msg)
			}
		}
		return m, nil

	case tea.PasteMsg:
		if m.dialog == nil && m.focus == focusTerm && m.term.started() {
			m.term.term.Paste(msg.Content)
		}
		return m, nil

	case tea.MouseMsg:
		if m.dialog != nil {
			return m, nil
		}
		ev := msg.Mouse()
		ox, oy := m.termOrigin()
		w, h := m.termInner()
		x, y := ev.X-ox, ev.Y-oy
		inTerm := x >= 0 && x < w && y >= 0 && y < h
		if _, click := msg.(tea.MouseClickMsg); click {
			if inTerm {
				m = m.setFocus(focusTerm)
			} else if ev.Y >= statusRows && !m.sideHidden {
				m = m.setFocus(focusSidebar)
			}
		}
		if inTerm && m.term.started() {
			wheel, isWheel := msg.(tea.MouseWheelMsg)
			if isWheel && !m.term.modes.mouse {
				switch wheel.Button {
				case tea.MouseWheelUp:
					m.term = m.term.scrollBy(3)
				case tea.MouseWheelDown:
					m.term = m.term.scrollBy(-3)
				}
				return m, nil
			}
			m.term.sendMouse(msg, x, y)
		}
		return m, nil
	}
	return m, nil
}

// setFocus moves focus and tells the child, which sees a focus event if it
// has turned focus reporting on.
func (m model) setFocus(f focus) model {
	if m.focus == f {
		return m
	}
	m.focus = f
	if m.term.started() {
		if f == focusTerm {
			m.term.term.Focus()
		} else {
			m.term.term.Blur()
		}
	}
	return m
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("") // no size yet
	}
	w, h := m.termInner()

	body := pane(fitBlock(m.term.content(), w, h), w, m.term.title(), m.focus == focusTerm, nil)
	if !m.sideHidden {
		sw, sh := sidebarWidth()+borderCols, h+borderRows
		side := paint(m.side.content(sw, sh, m.focus == focusSidebar), sw, sh, colText, colBgDark)
		body = lipgloss.JoinHorizontal(lipgloss.Top, side, body)
	}

	alt := ""
	if m.term.started() {
		if m.term.term.IsAltScreen() {
			alt += "  [alt]"
		}
		if m.term.modes.mouse {
			alt += "  [mouse]"
		}
		if m.term.scroll > 0 {
			alt += fmt.Sprintf("  [scroll %d/%d · esc]", m.term.scroll, m.term.term.ScrollbackLen())
		}
	}
	if m.dialog != nil {
		title := "rule"
		if titleBar {
			title = "bar"
		}
		alt += fmt.Sprintf("  [gradient: %s · tab · title: %s · shift+tab]", gradients[gradientIndex].name, title)
	} else if m.logoMode && m.focus == focusSidebar {
		alt += fmt.Sprintf("  [logo: %s · ← mark · → center · ↓ glow · ctrl+l done]", logoState())
	} else if m.focus == focusSidebar {
		alt += "  [ctrl+l logo]"
	}
	focused := "sidebar"
	if m.focus == focusTerm {
		focused = m.term.title()
	}
	status := fmt.Sprintf(" focus: %s  term %dx%d  %d B%s  ctrl+] switches · ctrl+^ sidebar · shift+pgup/pgdn or wheel scrolls · ctrl+q quits ",
		focused, w, h, m.term.bytes, alt)
	bar := statusStyle.Render(fitBlock(status, m.width, 1))

	frame := bar + "\n" + body
	if m.dialog != nil {
		frame = overlay(frame, m.dialog.render(m.width, m.height), m.width, m.height)
	}
	if dump := os.Getenv("VTSPIKE_DUMP"); dump != "" {
		_ = os.WriteFile(dump, []byte(frame), 0o644)
	}
	v := tea.NewView(frame)
	v.WindowTitle = m.term.title() // the child's title reaches the real tab too
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if m.dialog == nil && m.focus == focusTerm {
		if c := m.term.cursor(); c != nil {
			ox, oy := m.termOrigin()
			c.X += ox
			c.Y += oy
			v.Cursor = c
		}
	}
	return v
}

// stopChild ends the child the way a terminal does when its window closes:
// SIGHUP to the child's process group, a grace period for it to clean up,
// then SIGKILL for whatever is left. Claude Code in particular records a
// fullscreen boot as failed when the process dies before it is healthy, and a
// hard kill gives it no chance to finish.
func stopChild(cmd *exec.Cmd) {
	pgid := -cmd.Process.Pid // Setsid made the child a group leader
	_ = syscall.Kill(pgid, syscall.SIGHUP)
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = syscall.Kill(pgid, syscall.SIGKILL)
		<-done
	}
}

func main() {
	argv := os.Args[1:]
	if len(argv) == 0 {
		sh := os.Getenv("SHELL")
		if sh == "" {
			sh = "/bin/sh"
		}
		argv = []string{sh}
	}

	childCommand = strings.Join(argv, " ")
	m := model{
		focus: focusTerm,
		side: sidebar{items: []bead{
			{"cf-101", "settings schema", "open", "P1", nil,
				"Add the local block to .codefall/settings.json declaring the start and update entry points, and teach doctor to check that both exist and are runnable."},
			{"cf-102", "init writes section", "open", "P2", []string{"cf-101"},
				"init writes the refresh rule into AGENTS.md through the marker mechanism, and names equip in its report when nothing is declared."},
			{"cf-103", "preflight stamp", "open", "P2", []string{"cf-101"},
				"preflight.sh fetches and reports commits behind the default branch, a dirty tree, and the refresh stamp against HEAD; every verb reports it."},
			{"cf-104", "equip", "blocked", "P2", []string{"cf-101", "cf-103"},
				"Build and rebuild the project's local scripts, start and update, searching first on an existing repo and confirming with evidence."},
			{"cf-105", "refresh", "blocked", "P3", []string{"cf-104"},
				"Bring the checkout and the local environment current: start what is down, run update, record the stamp."},
			{"cf-106", "scaffold at code depth", "open", "P3", []string{"cf-104"},
				"scaffold emits both local scripts when it runs at code depth."},
		}},
		term: termPane{argv: argv},
	}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fm := final.(model)
	if fm.term.cmd != nil && fm.term.cmd.Process != nil && !fm.term.exited {
		stopChild(fm.term.cmd)
	}
	if fm.term.pty != nil {
		_ = fm.term.pty.Close()
	}
	if fm.term.err != nil {
		fmt.Fprintln(os.Stderr, fm.term.err)
		os.Exit(1)
	}
}
