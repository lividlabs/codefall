package application

import (
	"fmt"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// Show reads the effective configuration: the agents list as the settings module reads it, whether
// review may post, and the persona as the user file module reads it. It reads the way every other
// reader does, so a value Validate would refuse shows as the default it falls back to; doctor is what
// reports it.
func (c *Config) Show(dir string) (domain.Configuration, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return domain.Configuration{}, err
	}

	persona, err := c.Persona(dir)
	if err != nil {
		return domain.Configuration{}, err
	}

	return domain.Configuration{
		Entries: settings.Agents(project.doc),
		Default: !listsAgents(project.doc),
		Posting: settings.PostToPullRequest(project.doc),
		Persona: persona,
	}, nil
}

// Agents reads the agents list in order, the default when the settings write none.
func (c *Config) Agents(dir string) ([]settings.Entry, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return nil, err
	}

	return settings.Agents(project.doc), nil
}

// Persona reads who the person at the keyboard works as, and whether the user file says so. A
// missing file, or one that names no persona, is the default. A file that is there and not valid is
// an error rather than the default the skills fall back to, because a person asking is the one who
// can fix it.
func (c *Config) Persona(dir string) (domain.Persona, error) {
	_, doc, err := c.readUserFile(dir)
	if err != nil {
		return domain.Persona{}, err
	}

	if doc == nil {
		return domain.Persona{Name: userfile.DefaultPersona}, nil
	}

	if problems := userfile.Validate(doc); len(problems) > 0 {
		return domain.Persona{}, fmt.Errorf("%s is not valid: %s", userfile.Name, strings.Join(problems, "; "))
	}

	// Validate has accepted the file, so a persona that is there is a string it knows; a null or an
	// empty one counts as absent, as the module reads it.
	named, _ := doc[userfile.FieldPersona].(string)

	return domain.Persona{Name: userfile.Persona(doc), FromFile: named != ""}, nil
}

// listsAgents reports whether the settings spell out a list, as opposed to leaving the field absent
// or empty so the default applies.
func listsAgents(doc settings.Document) bool {
	entries, ok := doc[settings.FieldAgents].([]any)

	return ok && len(entries) > 0
}
