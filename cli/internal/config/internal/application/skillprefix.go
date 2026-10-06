package application

import (
	"fmt"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// SetSkillPrefix records what the installed skills are to be called (ADR-015). It writes the field
// and nothing else: the skills on disk keep their names until `codefall upgrade` runs, which reads
// the field, installs under it, and removes the old names, so the sentence a write reports says
// so. A prefix outside the format's closed set is refused in the format's words, and a value already
// as asked is left alone.
func (c *Config) SetSkillPrefix(dir, prefix string) (domain.Write, error) {
	chosen, err := settings.ParseSkillPrefix(prefix)
	if err != nil {
		return domain.Write{}, err
	}

	project, err := c.readSettings(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return domain.Write{}, err
	}

	if current, present := project.doc[settings.FieldSkillPrefix].(string); present && current == chosen {
		return domain.Unchanged(fmt.Sprintf("%s already has skill prefix %s", settingsName, chosen)), nil
	}

	body, err := withField(project.data, settings.FieldSkillPrefix, func(string) ([]byte, error) {
		return encode(chosen, false, "")
	})
	if err != nil {
		return domain.Write{}, fmt.Errorf("encode %s: %w", settingsName, err)
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("set skill prefix %s in %s; run codefall upgrade to rename the installed skills", chosen, settingsName)), nil
}
