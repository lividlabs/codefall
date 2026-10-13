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

The sidebar opens with the wordmark (`logo.ans`, CODE over FALL in block glyphs, recolored to the
Material cyan `#89ddff` with `#39adb5` for the shading) above the menu, in a dark glow: cells beside
a glyph take the fill tinted a quarter toward the teal, the ring outside that and the rows above and
below an eighth, a row counting as two columns. The left pane is a menu, Beads/Issues and About,
with a cursor on the arrows; `enter` or `right` confirms. The sidebar draws its own margin; its text
sits three cells in, under the wordmark's left edge, and the selected row's bar starts a cell before
that, inside the margin. Focus shows three ways: the cursor is drawn only while the sidebar has
focus, the text is muted without it, and while it has focus the bottom row carries the key help in
the Charm help bubble's style (`↑↓ move • enter confirm`).

Beads/Issues opens a dialog listing the beads; `↑`/`↓` choose, `enter` opens the bead's own
dialog, and `esc` closes. The bead dialog shows the bead's details and two buttons; `←`/`→`,
`tab`, or `shift+tab` move between them, `enter` confirms, and `esc` or Cancel returns to the
list. Implement types `/codefall-implement <id>` into the child through the emulator, as if at
the keyboard and without `enter`, and moves focus to it so sending is yours. About opens a
dialog naming the spike and the child's command line; `enter` or `esc` closes it. Every dialog
ends with a key help row in the same style as the sidebar's.

A dialog's title row is the title, and under it a hairline of `▔` across the full width of the
next row, so it underlines the title itself: a cell holds a letter or a block, never both, so the
rule is the top eighth of the row below. The rule is a Lip Gloss `Blend1D` gradient, one color per
cell, and the title takes its first stop. Seven gradients are in to compare, and `tab` cycles them
while a dialog is open, the status bar naming the one in use: the gold `#f2c25a` into the teal
`#89ddff`, Material's purple `#c792ea`, violet `#bb80b3`, red `#f07178`, or green `#c3e88d`, or
the wordmark's deep teal `#39adb5`; and the teal into the purple, for comparison. A fade ends
in another hue, never in the fill gray, which passes through desaturated mud and reads as the
same color dimming. While the gradients are under comparison `tab` is taken from the bead
dialog, whose buttons still move on the arrows.

A dialog is 70 cells wide, hairline to hairline, like Crush's command palette, or narrower when the
window leaves less after the ring and a cell of margin on each side; a list shows as many rows as
the window has and scrolls to keep the cursor in view. Text sits two cells in from the hairlines. A
selected row, in a dialog or the sidebar, is a blue-gray bar `#455a64` a step above the dialog fill,
set inside the padding rather than under it, a cell of fill left on each side, with a gold `▎` as
the bar's first cell for the cursor: the gold stays one cell wide and means what it means on a
button, that this is the one `enter` acts on. A gold bar, teal and dark teal bars, a dark gold bar,
and bare gold text were tried against it and set aside. The dialog is composed over a dimmed copy of
the frame on a Lip Gloss canvas: every cell's colors, backgrounds included, move halfway toward
black. The dialog itself is a shaded block with a hairline border of one-eighth blocks in the cyan
`#89ddff`: the sides flush with the edge, the top and bottom a cell in. The dialog sits in a ring
one cell wide, the top and bottom border rows and a blank column outside each side, that is
transparent: those cells take the dimmed frame's background, so the ring looks the same over a child
that paints its own background, as OpenCode does, as over one on the terminal's. Every rule lies
where the fill meets the ring and the corners close without a joining glyph. While the dialog is
open no key, paste, or mouse event reaches either pane. The right pane is the child. `ctrl+]` moves
focus between them, and a mouse click focuses the pane under it. `ctrl+^` (typed as `ctrl+6` on a US
keyboard) hides the sidebar and gives the child its width, or shows it again; hiding it moves focus
to the child, and `ctrl+]` on a hidden sidebar shows it and focuses it. The panes are colored blocks
rather than boxes, in OpenCode's Material palette (`packages/tui/src/theme/assets/ material.json`
there): the sidebar on `#1e272c`, and the status bar, dialog, and selected bead on `#37474f`. In
those, every cell left on the terminal's default colors is painted with the block's background and
`#eeffff`. The one color outside Material is the website's accent, the gold `#f2c25a`: the
complement of the wordmark's teal, it marks what is about to be acted on, the selection bar in a
dialog, the active button, and the focused title, roles the Material blue `#82aaff` had before. The
selected row of a list is not gold: it marks a position, not a commitment, so it is a gray bar with
a one-cell gold cursor. The terminal pane, margin included, is not painted at all: the child chose
its palette against the terminal's own colors, so it keeps them. The child's title is gold while it
has focus and gray otherwise; the sidebar has no title, the wordmark stands for it. `ctrl+q` quits
the host and kills the child. The wheel and `shift+pgup`/`shift+pgdn` scroll (see below), and `esc`
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
- The host keys are `ctrl+]`, `ctrl+^`, and `ctrl+q`, taken from the child unconditionally.
  Claude Code binds `ctrl+]` globally to `app:openArtifact`, so that one already collides; `ctrl+x`
  is a chord prefix in both Claude Code and OpenCode, so it is not available either. `ctrl+^` is
  bound by none of the three harnesses. A real host needs to pick keys no harness uses, or a
  tmux-style prefix. `ctrl+^` is matched by three spellings, `ctrl+^`, `ctrl+6`, and `ctrl+shift+6`,
  because a terminal with the kitty keyboard protocol on reports the key by its digit.
- `x/vt` is experimental with no tagged release; the module pins a commit.
