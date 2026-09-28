package domain

import (
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Configuration is the effective configuration `codefall config show` prints: what every reader of
// the two files acts on, which is not always what the files spell out.
type Configuration struct {
	// Entries is the agents list, one entry per active agent, in the file's order; Default says the
	// settings write none so the format's default applies.
	Entries []settings.Entry
	Default bool
	// Posting is whether codefall-review may post its findings to a pull request.
	Posting bool
	Persona Persona
}

// Persona is who the person at the keyboard works as, and whether the user file said so or the
// default applies.
type Persona struct {
	Name     string
	FromFile bool
}
