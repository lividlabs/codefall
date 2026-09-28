package domain

import (
	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Configuration is the effective configuration `codefall config show` prints: what every reader of
// the two files acts on, which is not always what the files spell out.
type Configuration struct {
	// Agents is the list in order, and Default says the settings list none so the format's default
	// applies.
	Agents  []settings.Agent
	Default bool
	// Review and Consult are each use's own order, when the settings give one.
	Review  mo.Option[[]string]
	Consult mo.Option[[]string]
	// ByHarness is the per-harness override, keyed by harness; empty when the settings give none.
	ByHarness map[string][]string
	Persona   Persona
}

// Persona is who the person at the keyboard works as, and whether the user file said so or the
// default applies.
type Persona struct {
	Name     string
	FromFile bool
}
