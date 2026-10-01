// Spike: run a terminal program inside a Bubble Tea v2 pane, beside a second
// pane owned by the host.
//
// The host spawns the command on a PTY (x/xpty), feeds the PTY's output into a
// virtual terminal (x/vt), renders the terminal's grid as the right pane, and
// sends the host's key, paste, and mouse events back through the emulator,
// which encodes them for the child. The emulator also answers the child's
// terminal queries (DA1, DSR, CPR) on the same pipe. The left pane is a static
// list the host draws itself, to see what layout and focus switching cost.
//
//	go run . [command [args...]]     default: $SHELL
//
// ctrl+] moves focus between the panes; a mouse click focuses the pane under
// it. ctrl+q quits the host and kills the child. Everything else goes to the
// focused pane.
package main

import (
	"context"
	"fmt"
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
	statusRows   = 1
	sidebarWidth = 28 // content; the border adds 2
	borderCols   = 2
	borderRows   = 2
)

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

// sidebar is the host's own pane: a static list with a cursor, standing in for
// whatever a real host would show beside the harness.
type sidebar struct {
	items  []bead
	cursor int
}

// update moves the cursor, and on enter returns the chosen bead.
func (s sidebar) update(msg tea.KeyPressMsg) (sidebar, *bead) {
	switch msg.String() {
	case "up", "k":
		if s.cursor > 0 {
			s.cursor--
		}
	case "down", "j":
		if s.cursor < len(s.items)-1 {
			s.cursor++
		}
	case "enter":
		if s.cursor < len(s.items) {
			b := s.items[s.cursor]
			return s, &b
		}
	}
	return s, nil
}

func (s sidebar) content(width, height int) string {
	var b strings.Builder
	for i, it := range s.items {
		if i >= height {
			break
		}
		mark := "  "
		line := it.id + " " + it.title
		if i == s.cursor {
			mark = "❯ "
			line = lipgloss.NewStyle().Bold(true).Render(line)
		}
		b.WriteString(mark + line)
		if i < len(s.items)-1 {
			b.WriteByte('\n')
		}
	}
	return fitBlock(b.String(), width, height)
}

// --- bead dialog -------------------------------------------------------------------------------

// beadDialog is the modal opened by enter on a bead: its details and two
// buttons. Implement types a slash command into the child; Cancel closes.
type beadDialog struct {
	bead   bead
	button int // 0 implement, 1 cancel
}

const dialogWidth = 60

var (
	dialogLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	buttonStyle  = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("8"))
	buttonActive = lipgloss.NewStyle().Padding(0, 2).Reverse(true).Bold(true)
)

func (d beadDialog) render() string {
	b := d.bead
	w := dialogWidth
	row := func(label, value string) string {
		return dialogLabel.Render(fmt.Sprintf("%-9s", label)) + value
	}
	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Render(b.title),
		"",
		row("status", b.status),
		row("priority", b.priority),
		row("deps", strings.Join(b.deps, ", ")),
		"",
	)
	lines = append(lines, strings.Split(lipgloss.NewStyle().Width(w).Render(b.description), "\n")...)
	impl, cancel := buttonStyle, buttonStyle
	if d.button == 0 {
		impl = buttonActive
	} else {
		cancel = buttonActive
	}
	buttons := impl.Render("Implement") + "  " + cancel.Render("Cancel")
	lines = append(lines, "", lipgloss.PlaceHorizontal(w, lipgloss.Right, buttons))
	body := fitBlock(strings.Join(lines, "\n"), w, len(lines))
	return box(body, w, b.id, true)
}

// update handles a key while the dialog is open. It returns the dialog (nil
// when it closed) and whether Implement was chosen.
func (d *beadDialog) update(msg tea.KeyPressMsg) (*beadDialog, bool) {
	switch msg.String() {
	case "left", "h", "shift+tab":
		d.button = 0
	case "right", "l", "tab":
		d.button = 1
	case "esc", "q":
		return nil, false
	case "enter":
		return nil, d.button == 0
	}
	return d, false
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
	term          termPane
	dialog        *beadDialog // open modal, or nil
}

var (
	statusStyle  = lipgloss.NewStyle().Reverse(true)
	focusedEdge  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	blurredEdge  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	focusedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	blurredTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// box draws a rounded border around a block that is already exactly w by h,
// with the title set into the top edge the way tmux's pane-border-status does.
func box(block string, w int, title string, focused bool) string {
	edge, ttl := blurredEdge, blurredTitle
	if focused {
		edge, ttl = focusedEdge, focusedTitle
	}
	if w < 4 {
		title = ""
	} else {
		title = ansi.Truncate(title, w-4, "…")
	}
	tw := ansi.StringWidth(title)
	top := edge.Render("╭" + strings.Repeat("─", max(w, 0)) + "╮")
	if tw > 0 {
		top = edge.Render("╭─ ") + ttl.Render(title) + edge.Render(" "+strings.Repeat("─", max(w-3-tw, 0))+"╮")
	}
	bottom := edge.Render("╰" + strings.Repeat("─", max(w, 0)) + "╯")
	side := edge.Render("│")
	var b strings.Builder
	b.WriteString(top)
	for _, line := range strings.Split(block, "\n") {
		b.WriteString("\n" + side + line + side)
	}
	b.WriteString("\n" + bottom)
	return b.String()
}

func (m model) Init() tea.Cmd { return nil }

// termInner is the inner size of the terminal pane for the current window.
func (m model) termInner() (int, int) {
	w := m.width - (sidebarWidth + borderCols) - borderCols
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
	return sidebarWidth + borderCols + 1, statusRows + 1
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
			var implement bool
			m.dialog, implement = m.dialog.update(msg)
			if implement && m.term.started() {
				// Type the command into the child as if at the keyboard,
				// without enter, and hand it the focus so the user sends it.
				m.term.term.SendText("/codefall-implement " + m.side.items[m.side.cursor].id)
				m = m.setFocus(focusTerm)
			}
			return m, nil
		}
		if msg.String() == "ctrl+]" {
			return m.setFocus(1 - m.focus), nil
		}
		switch m.focus {
		case focusSidebar:
			var chosen *bead
			m.side, chosen = m.side.update(msg)
			if chosen != nil {
				m.dialog = &beadDialog{bead: *chosen}
			}
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
			} else if ev.Y >= statusRows {
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

	side := box(m.side.content(sidebarWidth, h), sidebarWidth, "beads", m.focus == focusSidebar)
	term := box(fitBlock(m.term.content(), w, h), w, m.term.title(), m.focus == focusTerm)
	body := lipgloss.JoinHorizontal(lipgloss.Top, side, term)

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
	focused := "sidebar"
	if m.focus == focusTerm {
		focused = m.term.title()
	}
	status := fmt.Sprintf(" focus: %s  term %dx%d  %d B%s  ctrl+] switches · shift+pgup/pgdn or wheel scrolls · ctrl+q quits ",
		focused, w, h, m.term.bytes, alt)
	bar := statusStyle.Render(fitBlock(status, m.width, 1))

	frame := bar + "\n" + body
	if m.dialog != nil {
		d := m.dialog.render()
		x := (m.width - lipgloss.Width(d)) / 2
		y := (m.height - lipgloss.Height(d)) / 2
		frame = lipgloss.NewCompositor(
			lipgloss.NewLayer(frame),
			lipgloss.NewLayer(d).X(max(x, 0)).Y(max(y, 0)).Z(1),
		).Render()
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
