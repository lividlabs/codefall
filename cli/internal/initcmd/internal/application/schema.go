package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// currentSchema points .codefall/settings.json's $schema at the schema URL codefall publishes now
// when the file names one an earlier release wrote, and reports whether it did. A project set up
// before 0.26.0 carries a URL that no longer resolves, and an editor that follows it validates
// nothing.
//
// The URL is spliced into the file's text for the reason recordHarnesses splices its list: re-encoding
// would reorder every key and drop whatever a project had added. Only the bytes of the value change.
func (i *Initialize) currentSchema(dir string) (bool, error) {
	data, err := i.files.ReadFile(settingsPath(dir))
	if err != nil {
		return false, fmt.Errorf("read %s: %w", settingsName, err)
	}

	body, changed, err := settingsWithCurrentSchema(data)
	if err != nil || !changed {
		return false, err
	}

	if err := i.files.WriteFile(settingsPath(dir), body); err != nil {
		return false, fmt.Errorf("write %s: %w", settingsName, err)
	}

	return true, nil
}

// settingsWithCurrentSchema is the settings file's text with a former schema URL replaced by
// settings.SchemaID, and whether it replaced one. A file with no $schema, with the current URL, or
// with a URL codefall never wrote is returned as it was: the first two need nothing, and the third is
// the project's own choice.
func settingsWithCurrentSchema(data []byte) ([]byte, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	if start, err := decoder.Token(); err != nil {
		return nil, false, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return data, false, nil
	}

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, false, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		if key != settings.FieldSchema {
			continue
		}

		var recorded string
		if err := json.Unmarshal(value, &recorded); err != nil || !slices.Contains(settings.FormerSchemaIDs, recorded) {
			return data, false, nil
		}

		current, err := json.Marshal(settings.SchemaID)
		if err != nil {
			return nil, false, fmt.Errorf("encode %s: %w", settings.FieldSchema, err)
		}

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		end := int(decoder.InputOffset())
		from := end - len(value)

		return slices.Concat(data[:from], current, data[end:]), true, nil
	}

	return data, false, nil
}
