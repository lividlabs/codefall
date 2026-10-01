# Spike: a terminal program inside a Bubble Tea pane

A throwaway program that runs another terminal application inside a Bubble Tea v2 view, the way
tmux runs one inside a pane, beside a second pane the host draws itself. It is its own Go module
so nothing here reaches the CLI's dependency set or its lint rules.

```
cd spikes/vt
go run .                 # your $SHELL in a pane
go run . claude          # Claude Code in a pane
go run . codex
```

The left pane is a static list of beads with a cursor (`j`/`k` or the arrows). `enter` opens a
dialog over both panes with the bead's details and two buttons; `tab`, `shift+tab`, or the
arrows move between them, `enter` chooses, and `esc` closes. Implement types
`/codefall-implement <id>` into the child through the emulator, as if at the keyboard and
without `enter`, and moves focus to it so sending is yours. Cancel closes and nothing reaches
the child. The dialog is a second Lip Gloss layer composited over the frame, and while it is
open no key, paste, or mouse event reaches either pane. The right pane is the child.
`ctrl+]` moves focus between them, and a mouse click focuses the pane under it. `ctrl+q` quits the
host and kills the child. The wheel and `shift+pgup`/`shift+pgdn` scroll (see below), and `esc`
leaves scroll mode. Every other key, paste, and mouse event goes to the focused pane.

Two environment variables help when something looks wrong: `VTSPIKE_DUMP=<file>` writes the last
composed frame on every render, and `VTSPIKE_LOG=<file>` appends every terminal mode the child
sets or resets.

## How it works

Four parts, all from the Charm `x` repositories the CLI already draws on:

1. `x/xpty` opens a PTY at the pane's size and starts the command on it.
2. `x/vt` is the virtual terminal. The PTY's output is written into it, and it keeps the grid, the
   alternate screen, colours, cursor, and scrollback. It also answers the child's queries (DA1,
   DSR, CPR) on its own output pipe, which is copied to the PTY.
3. The view is a status bar over `lipgloss.JoinHorizontal` of the two bordered panes. The right
   pane's content is `Emulator.Render()`, and its inner size is the window minus the bar, the
   sidebar, and the borders. The host cursor is `CursorPosition()` offset by the pane's origin,
   and mouse events are offset the other way before they are forwarded; those two offsets are
   the only places the layout and the emulator meet.
4. Key, paste, and mouse messages are converted to ultraviolet events and sent through
   `SendKey`, `SendText`, `Paste`, and `SendMouse`, which encode them for whatever modes the
   child has turned on.

The wiring follows `muhamm-ad/bubble-ssh`, which embeds an SSH session the same way.

The panes are drawn by `box`, a rounded border with the title set into the top edge the way
tmux's pane-border-status does. The terminal pane's title is whatever the child last set with
OSC 0 or 2, captured by the filter below with its original characters, and falling back to the
command. The same title is handed to Bubble Tea as the window title, so the real tab follows the
child.

## Scrolling

The host does what tmux does, and which of three things that is depends on what the child has
turned on. The emulator reports mode changes through its `EnableMode` and `DisableMode` callbacks,
and the pane keeps the two it cares about: whether the child is on the alternate screen, and
whether it has turned on any mouse reporting mode (1000, 1002, 1003).

1. **The child reports the mouse.** Wheel events are forwarded and the child scrolls itself.
   Claude Code is this case: it sets 1049, 1000, 1002, 1003, and 1006 at startup.
2. **The child is on the alternate screen without the mouse.** There is no history to show, so
   each wheel tick becomes three arrow keys, which is what `less`, `vim`, and the like expect.
3. **The child is on the main screen.** Lines that scroll off the top are in the emulator's
   scrollback. The wheel and `shift+pgup`/`shift+pgdn` shift the view into it, the pane is drawn
   from scrollback cells above live cells, the cursor is hidden, the status bar shows the offset,
   and `esc` or any typed key returns to the live screen.

Verified with the driver: `seq 1 300` under `sh` scrolled twelve lines back on four ticks;
`less --mouse` moved two lines on two ticks (it owns the wheel); plain `less` moved six lines on
two ticks (three arrow keys each).

## Verified on 2026-10-01

Driven from a Python script that forks the host on a 120x36 PTY and reads what it paints.

- A short `sh` child's output is painted; the host exits by itself when the child exits.
- `claude` (2.1.286) starts, draws its full screen, including the trust dialog when started in an
  untrusted directory, and text typed through the emulator appears in its prompt. Claude Code
  switches the emulator to the alternate screen.
- OpenCode and Codex run in their fullscreen modes with working scrollback (checked by hand).
- With the sidebar beside it, Claude Code lays itself out at the pane's 88x33, typed text still
  lands in its prompt, `ctrl+]` moves focus to the sidebar, and `j` moves the sidebar's cursor
  while the child receives nothing. Focus changes are passed to the child through
  `Emulator.Focus()` and `Blur()`, so a child with focus reporting on sees them.

## One thing lipgloss must not do

`Style.Width` and `Style.Height` word-wrap any line they measure as wider than the width. Applied
to the emulator's output they fold rows, and the grid drifts until the pane is unreadable, which
is what the first two-pane build did. The panes are fitted by hand instead: each line is cut with
`ansi.Truncate`, padded to the width, and the row count is forced. A border style with no width
set is then safe to wrap around the block. The frame can be checked with `VTSPIKE_DUMP=<file>`,
which writes the last composed frame on every render.

## A parser bug, worked around in `filter.go`

The `x/ansi` parser treats a raw `0x9C` byte inside an OSC, DCS, APC, PM, or SOS string as the
8-bit String Terminator even when it is a UTF-8 continuation byte, so the string ends there and
the rest prints to the screen. Every character in the Dingbats block has `0x9C` as its second
byte, and Claude Code's window titles begin with U+2733 `✳`, so each title update leaked
" Claude Code" into the pane at the cursor, which is the input box. Ordinary screen text is not
affected; only string payloads are. It is charmbracelet/x issue #848, with fix PRs #946 and #976
open as of 2026-10-01. Until one merges, `filter.go` sits between the PTY and the emulator and
replaces any character containing `0x9C` inside a string with `?`, tracking state across chunk
boundaries. `filter_test.go` covers the title, clean input, every split point, and the emulator
end to end. Delete both files when upstream fixes it.

## Ending the child

The host ends the child the way a terminal does when its window closes: SIGHUP to the child's
process group, three seconds of grace, then SIGKILL. The first version used SIGKILL alone, and
that cost a day: Claude Code records a fullscreen boot in `~/.claude.json` and clears the record
only after the first frame plus ten seconds of uptime, or a clean exit via `/exit`, Ctrl+C, or
Ctrl+D. A process that dies before that counts as a failed start; two failed starts on one Claude
Code version turn fullscreen off on the machine until `/tui fullscreen` is run. Scripted runs that
killed Claude Code at eight seconds did exactly this. Anything that hosts Claude Code has to
either let it exit or keep it alive past ten seconds. `CLAUDE_CODE_NO_FLICKER=1` makes a session
fullscreen without counting it, which is the right setting for a scripted check.

## Two things xpty does not do

- `Start` wires the slave to the child's stdio but does not make it the controlling terminal. The
  spike sets `Setsid` and `Setctty` itself; without them `ctrl+c` and job control never reach the
  child through the line discipline.
- The parent keeps the slave open, so the master never reports end of file. The child's exit is
  observed by waiting on the process, not by the read loop ending.

## Known gaps

- `x/vt`'s `SendKey` has a TODO for the kitty keyboard protocol, CSI u, and modifyOtherKeys. A
  child that enables kitty keys (Claude Code does, for shift+enter) gets legacy encodings instead.
  Untested whether that breaks anything beyond shift+enter.
- OSC 10/11 colour queries and the kitty keyboard query are not answered. Claude Code started
  anyway, so it falls back cleanly.
- Scroll mode has no selection or copy. tmux's copy mode is the reference if that is wanted.
- New output while scrolled keeps the offset, which is what tmux does; the pane does not
  pin to the history line the way a terminal with a scrollbar would.
- The host keys are `ctrl+]` and `ctrl+q`, taken from the child unconditionally. A real host
  needs to pick keys no harness uses, or a tmux-style prefix.
- `x/vt` is experimental with no tagged release; the module pins a commit.
