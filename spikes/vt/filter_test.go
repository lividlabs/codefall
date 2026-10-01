package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

func TestC1FilterTitleWithStar(t *testing.T) {
	in := "A\x1b]0;✳ Claude Code\x07B"
	var f c1Filter
	got := string(f.Push([]byte(in)))
	want := "A\x1b]0;? Claude Code\x07B"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestC1FilterLeavesCleanInputAlone(t *testing.T) {
	cases := []string{
		"plain text ✳ with the glyph on screen",
		"A\x1b]0;café \U0001F600\x07B",
		"\x1b[?2026h\x1b[H\x1b[38;2;1;2;3m✓ done\x1b[m",
		"\x1b]8;;https://example.com\x07x\x1b]8;;\x07",
		"\x1bPq data\x1b\\after",
	}
	for _, in := range cases {
		var f c1Filter
		if got := string(f.Push([]byte(in))); got != in {
			t.Errorf("changed %q to %q", in, got)
		}
	}
}

func TestC1FilterAcrossChunks(t *testing.T) {
	in := "A\x1b]0;✳ Claude Code\x1b\\B\x1b]2;x✳\x07C"
	want := "A\x1b]0;? Claude Code\x1b\\B\x1b]2;x?\x07C"
	for cut := 1; cut < len(in); cut++ {
		var f c1Filter
		got := string(f.Push([]byte(in[:cut]))) + string(f.Push([]byte(in[cut:])))
		if got != want {
			t.Errorf("cut at %d: got %q want %q", cut, got, want)
		}
	}
}

func TestC1FilterKeepsTheRealTitle(t *testing.T) {
	in := "A\x1b]0;\u2733 Claude Code\x07B\x1b]2;second \u2713\x1b\\C\x1b]8;;https://x\x07D"
	for cut := 1; cut < len(in); cut++ {
		var f c1Filter
		f.Push([]byte(in[:cut]))
		f.Push([]byte(in[cut:]))
		if f.Title != "second \u2713" || f.TitleSeq != 2 {
			t.Errorf("cut at %d: title %q seq %d", cut, f.Title, f.TitleSeq)
		}
	}
}

func TestC1FilterThroughEmulator(t *testing.T) {
	e := vt.NewEmulator(40, 2)
	var title string
	e.SetCallbacks(vt.Callbacks{Title: func(s string) { title = s }})
	var f c1Filter
	e.Write(f.Push([]byte("A\x1b]0;✳ Claude Code\x07B")))
	screen := strings.TrimRight(strings.Split(ansi.Strip(e.Render()), "\n")[0], " ")
	if screen != "AB" {
		t.Errorf("screen %q, want AB", screen)
	}
	if title != "? Claude Code" {
		t.Errorf("title %q", title)
	}
}
