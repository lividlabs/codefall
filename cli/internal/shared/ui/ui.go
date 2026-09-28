// Package ui is the shared module every command's presentation layer draws through: the palette, the
// marks, the colour-profile writer, and the spinner a slow job runs under. It knows nothing about
// what a command reports — a component maps its own statuses onto a Tone and binds its own data
// (ADR-GO-02 rule 4: this module never imports a component).
package ui

import (
	"sync"

	"charm.land/lipgloss/v2"
)

// Tone is how much attention a line is asking for. It is the one vocabulary the commands share:
// doctor maps a check's status onto it, initcmd maps a step's outcome, and neither one owns a
// colour.
type Tone int

const (
	// ToneNone is the zero value and carries no styling, which is what an unrecognised status or
	// outcome falls back to.
	ToneNone Tone = iota
	// TonePrimary is something that is as it should be — a passing check, a finished step, the
	// spinner itself.
	TonePrimary
	// ToneWarn is something worth knowing about that is not a failure.
	ToneWarn
	// ToneFail is something that is wrong.
	ToneFail
	// ToneFaint is secondary text: versions, accounts, remedies, the label under a spinner.
	ToneFaint
)

// The palette is the teal trail from the brand image — desaturated teals for what is right, muted
// amber for a warning, muted coral for a failure — picked light/dark-aware the way Fang's own colour
// scheme picks its. Warn and fail are warm hues so they do not collapse into the teal. Secondary
// text is dimmed and tinted teal-grey.
//
// The scheme is built once, on first use, because deciding it means asking the terminal a question.
var styles = sync.OnceValue(func() map[Tone]lipgloss.Style {
	c := lipgloss.LightDark(HasDarkBackground())

	return map[Tone]lipgloss.Style{
		TonePrimary: lipgloss.NewStyle().Bold(true).
			Foreground(c(lipgloss.Color("#2F6B6B"), lipgloss.Color("#6AB3B3"))),
		ToneWarn: lipgloss.NewStyle().Bold(true).
			Foreground(c(lipgloss.Color("#7E6217"), lipgloss.Color("#D9B44A"))),
		ToneFail: lipgloss.NewStyle().Bold(true).
			Foreground(c(lipgloss.Color("#9E3A3A"), lipgloss.Color("#E08878"))),
		ToneFaint: lipgloss.NewStyle().Faint(true).
			Foreground(c(lipgloss.Color("#526D71"), lipgloss.Color("#8AA3A8"))),
	}
})

// Prime settles the palette now rather than on first use. Deciding it asks the terminal for its
// background colour and reads the answer from stdin, so a program that will own stdin — a Bubble Tea
// screen — calls this before it starts; a first use from inside the program's View would race the
// program's own reader for the reply and could wait on it for ever. The spinner is safe without it
// because its first style is built before its program starts.
func Prime() {
	_ = styles()
}

// Style is the one place a tone becomes a colour, so no two commands can drift apart. An
// unrecognised tone is left unstyled.
func Style(tone Tone) lipgloss.Style {
	if style, ok := styles()[tone]; ok {
		return style
	}

	return lipgloss.NewStyle()
}

// HeadingStyle is a chip drawn the way Fang draws its ERROR header: the same foreground-on-background
// pair, padding, and weight, in the colour Fang gives a title. The colorprofile writer strips the
// styling on a non-terminal stdout and under NO_COLOR, leaving the word itself.
var HeadingStyle = sync.OnceValue(func() lipgloss.Style {
	c := lipgloss.LightDark(HasDarkBackground())

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(c(lipgloss.Color("#C8DADA"), lipgloss.Color("#C8DADA"))).
		Background(c(lipgloss.Color("#143337"), lipgloss.Color("#3A7680"))).
		Padding(0, 1)
})

// The glyph each tone prints when it is standing in for a status. A command whose own vocabulary
// needs a different mark — initcmd draws a skipped step as a dash, because nothing is wrong — keeps
// that glyph to itself and takes only the colour from here.
var marks = map[Tone]string{
	TonePrimary: "✓",
	ToneWarn:    "!",
	ToneFail:    "✗",
}

// Mark is the bare glyph for a tone, unstyled. A tone with no mark of its own answers "?".
func Mark(tone Tone) string {
	if mark, ok := marks[tone]; ok {
		return mark
	}

	return "?"
}
