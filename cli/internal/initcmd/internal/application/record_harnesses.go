package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// recordHarnesses adds to .codefall/settings.json's harnesses list every name in wanted that it does
// not carry, and returns what it added, sorted. It runs after respell, so the list is read under the
// names the harnesses have now and a former spelling is not counted as missing.
//
// The list is spliced into the file's text rather than the document being decoded and encoded again,
// for the reason respell splices its names: re-encoding would reorder every key and drop whatever a
// project had added. Only the bytes of the list change, laid out the way the file already lays it
// out — one name per line when the list spans lines, on one line when it does not.
func (i *Initialize) recordHarnesses(dir string, wanted []string) ([]string, error) {
	data, err := i.files.ReadFile(settingsPath(dir))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", settingsName, err)
	}

	body, added, err := settingsWithHarnesses(data, wanted)
	if err != nil || len(added) == 0 {
		return nil, err
	}

	if err := i.files.WriteFile(settingsPath(dir), body); err != nil {
		return nil, fmt.Errorf("write %s: %w", settingsName, err)
	}

	return added, nil
}

// settingsWithHarnesses is the settings file's text with every name in wanted that its harnesses
// list does not carry appended to that list, and the names it appended, sorted. A file with no
// harnesses list, or one that is not a list of names, is returned as it was: validation is what says
// so.
func settingsWithHarnesses(data []byte, wanted []string) ([]byte, []string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	if start, err := decoder.Token(); err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return data, nil, nil
	}

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		if key != settings.FieldHarnesses {
			continue
		}

		var recorded []string
		if err := json.Unmarshal(value, &recorded); err != nil {
			return data, nil, nil
		}

		var added []string

		for _, name := range wanted {
			if !slices.Contains(recorded, name) && !slices.Contains(added, name) {
				added = append(added, name)
			}
		}

		if len(added) == 0 {
			return data, nil, nil
		}

		slices.Sort(added)

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		end := int(decoder.InputOffset())
		from := end - len(value)

		list, err := renderHarnesses(append(recorded, added...), value, indentBefore(data, from))
		if err != nil {
			return nil, nil, err
		}

		return slices.Concat(data[:from], list, data[end:]), added, nil
	}

	return data, nil, nil
}

// renderHarnesses lays the list out the way the file laid the old one out: one name per line, one
// level in from the key, when the old list spanned lines, and on one line otherwise.
func renderHarnesses(names []string, previous json.RawMessage, indent string) ([]byte, error) {
	quoted := make([]string, 0, len(names))

	for _, name := range names {
		q, err := json.Marshal(name)
		if err != nil {
			return nil, fmt.Errorf("encode %q: %w", name, err)
		}

		quoted = append(quoted, string(q))
	}

	if !bytes.ContainsRune(previous, '\n') {
		return []byte("[" + strings.Join(quoted, ", ") + "]"), nil
	}

	inner := indent + "  "

	return []byte("[\n" + inner + strings.Join(quoted, ",\n"+inner) + "\n" + indent + "]"), nil
}

// indentBefore is the whitespace that opens the line offset sits on, which is the indentation of the
// key whose value starts there.
func indentBefore(data []byte, offset int) string {
	start := bytes.LastIndexByte(data[:offset], '\n') + 1

	line := data[start:offset]
	trimmed := bytes.TrimLeft(line, " \t")

	return string(line[:len(line)-len(trimmed)])
}
