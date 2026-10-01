package application

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// harnessConfigDocument is one block in the file's shape. The field order is the one the schema
// lists, and a field the block does not carry is left out rather than written null, so the file
// holds exactly what was given.
type harnessConfigDocument struct {
	Harness   mo.Option[string] `json:"harness,omitzero"`
	ModelFlag mo.Option[string] `json:"modelFlag,omitzero"`
	Provider  mo.Option[string] `json:"provider,omitzero"`
	Args      []string          `json:"args,omitempty"`
	Env       mo.Option[string] `json:"env,omitzero"`
}

// SetHarnessConfig replaces one block with exactly the fields given. A block under a harness's own
// name carries no harness field, since the key already says; any other key carries the harness the
// caller gave, and the settings module refuses the result when it gave none. The block is laid out
// the way the one it replaces was, or the way the object holds its other blocks, and a block already
// equal to the one given is left alone.
func (c *Config) SetHarnessConfig(dir, key string, block settings.HarnessConfig) (domain.Write, error) {
	if emptyHarnessConfig(block) {
		return domain.Write{}, domain.ErrEmptyHarnessConfig
	}

	project, err := c.readSettings(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return domain.Write{}, err
	}

	path := settings.FieldHarnessConfig + "." + key

	// A harness-named key is its own harness, for the comparison below; a harness given for one is
	// written as given, so the settings module refuses it in its own words.
	given := block.Harness
	if given == "" && harness.SkillsDir(key).IsPresent() {
		block.Harness = key
	}

	if existing, has := settings.HarnessConfigs(project.doc)[key]; has && sameHarnessConfig(existing, block) {
		return domain.Unchanged(fmt.Sprintf("%s already sets %s that way", settingsName, path)), nil
	}

	document := harnessConfigDocument{
		ModelFlag: block.ModelFlag,
		Provider:  block.Provider,
		Args:      block.Args,
		Env:       block.Env,
	}

	if given != "" {
		document.Harness = mo.Some(given)
	}

	blocks, err := nestedObject(project.data, settings.FieldHarnessConfig)

	var body []byte

	if err == nil {
		// The object is there: the new block follows the layout of the one it replaces, or of the
		// object around it when it is new.
		multiline := bytes.ContainsRune(project.data[blocks.open:blocks.closing], '\n')
		if existing, held := blocks.last(key); held {
			multiline = bytes.ContainsRune(existing.raw, '\n')
		}

		body, err = withNestedField(project.data, settings.FieldHarnessConfig, key, func(indent string) ([]byte, error) {
			return encode(document, multiline, indent)
		})
	} else {
		// No object yet, or one that is not an object: the settings module accepted the file, so it
		// is absent, and it is created holding the one block.
		body, err = withField(project.data, settings.FieldHarnessConfig, func(indent string) ([]byte, error) {
			return objectWithMember(indent, key, func(inner string) ([]byte, error) {
				return encode(document, true, inner)
			})
		})
	}

	if err != nil {
		return domain.Write{}, fmt.Errorf("encode %s: %w", settingsName, err)
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(fmt.Sprintf("set %s in %s", path, settingsName)), nil
}

// ClearHarnessConfig removes one block, so that harness runs bare again, and the harnessConfig
// object with it when it held nothing else.
func (c *Config) ClearHarnessConfig(dir, key string) (domain.Write, error) {
	project, err := c.readSettings(dir)
	if err != nil {
		return domain.Write{}, err
	}

	if err := validSettings(project.doc, settingsName+" is not valid, so nothing was changed"); err != nil {
		return domain.Write{}, err
	}

	path := settings.FieldHarnessConfig + "." + key

	blocks, err := nestedObject(project.data, settings.FieldHarnessConfig)
	if err != nil {
		// The settings module accepted the file, so the object is absent rather than malformed.
		return domain.Unchanged(fmt.Sprintf("%s has no %s to clear", settingsName, path)), nil
	}

	if _, held := blocks.last(key); !held {
		return domain.Unchanged(fmt.Sprintf("%s has no %s to clear", settingsName, path)), nil
	}

	detail := fmt.Sprintf("removed %s from %s", path, settingsName)

	var body []byte

	if others := slices.ContainsFunc(blocks.members, func(m member) bool { return m.key != key }); others {
		body, err = withoutNestedField(project.data, settings.FieldHarnessConfig, key)
	} else {
		body, err = withoutField(project.data, settings.FieldHarnessConfig)
		detail += ", and " + settings.FieldHarnessConfig + " with it, since it held nothing else"
	}

	if err != nil {
		return domain.Write{}, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if err := c.writeSettings(dir, body); err != nil {
		return domain.Write{}, err
	}

	return domain.Changed(detail), nil
}

// emptyHarnessConfig reports whether a block carries nothing: no field the shared script would read,
// and no harness, which on a variant is the one field it must carry.
func emptyHarnessConfig(block settings.HarnessConfig) bool {
	return block.Harness == "" && block.ModelFlag.IsAbsent() && block.Provider.IsAbsent() &&
		block.Env.IsAbsent() && len(block.Args) == 0
}

// sameHarnessConfig reports whether two blocks say the same thing. An absent args and an empty one
// are the same: neither adds an argument.
func sameHarnessConfig(a, b settings.HarnessConfig) bool {
	return a.Harness == b.Harness && a.ModelFlag == b.ModelFlag && a.Provider == b.Provider &&
		a.Env == b.Env && slices.Equal(a.Args, b.Args)
}
