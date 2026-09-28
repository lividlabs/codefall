package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
)

// errNotObject is what a file whose top level is not a JSON object is refused with.
var errNotObject = errors.New("the top level must be a JSON object")

// member is one top-level key of a JSON object's text, and where its value sits in that text.
type member struct {
	key  string
	raw  json.RawMessage
	from int
	end  int
}

// members reads a JSON object's top-level keys in the order the text holds them, with the byte range
// each value occupies, and the offset of the closing brace. It is what lets a write change one value
// and leave every other byte of a checked-in file as its author laid it out: decoding into a map and
// encoding it again would reorder every key.
func members(data []byte) ([]member, int, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	start, err := decoder.Token()
	if err != nil {
		return nil, 0, err
	}

	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return nil, 0, errNotObject
	}

	var found []member

	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, 0, err
		}

		key, _ := token.(string)

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, 0, err
		}

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		end := int(decoder.InputOffset())
		found = append(found, member{key: key, raw: raw, from: end - len(raw), end: end})
	}

	if _, err := decoder.Token(); err != nil {
		return nil, 0, err
	}

	return found, int(decoder.InputOffset()) - 1, nil
}

// field returns one top-level value's text, and false when the object has no such key. A key
// written twice is read the way encoding/json reads it: the last one counts.
func field(data []byte, key string) (json.RawMessage, bool, error) {
	found, _, err := members(data)
	if err != nil {
		return nil, false, err
	}

	for _, m := range slices.Backward(found) {
		if m.key == key {
			return m.raw, true, nil
		}
	}

	return nil, false, nil
}

// withField returns a JSON object's text with key set to what render produces: the value's bytes
// replaced where the key already is, and the key added as the object's last member where it is not.
// Nothing else in the text changes. render is handed the indentation of the key's line, so a value
// that spans lines can lay itself out one level in from it.
func withField(data []byte, key string, render func(indent string) ([]byte, error)) ([]byte, error) {
	found, closing, err := members(data)
	if err != nil {
		return nil, err
	}

	for _, m := range slices.Backward(found) {
		if m.key != key {
			continue
		}

		value, err := render(indentBefore(data, m.from))
		if err != nil {
			return nil, err
		}

		return slices.Concat(data[:m.from], value, data[m.end:]), nil
	}

	quoted, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("encode %q: %w", key, err)
	}

	if len(found) == 0 {
		const indent = "  "

		value, err := render(indent)
		if err != nil {
			return nil, err
		}

		open := bytes.IndexByte(data, '{')
		inside := slices.Concat([]byte("\n"+indent), quoted, []byte(": "), value, []byte("\n"))

		return slices.Concat(data[:open+1], inside, data[closing:]), nil
	}

	last := found[len(found)-1]
	indent := indentBefore(data, last.from)

	value, err := render(indent)
	if err != nil {
		return nil, err
	}

	added := slices.Concat([]byte(",\n"+indent), quoted, []byte(": "), value)

	return slices.Concat(data[:last.end], added, data[last.end:]), nil
}

// indentBefore is the whitespace that opens the line offset sits on, which is the indentation of the
// key whose value starts there.
func indentBefore(data []byte, offset int) string {
	start := bytes.LastIndexByte(data[:offset], '\n') + 1

	line := data[start:offset]
	trimmed := bytes.TrimLeft(line, " \t")

	return string(line[:len(line)-len(trimmed)])
}

// encode renders a value as JSON: compact when indented is false, and otherwise indented two spaces per
// level with every line after the first opening with prefix. HTML is not escaped, because the text is
// read by people and by JSON parsers, never by a browser.
func encode(value any, indented bool, prefix string) ([]byte, error) {
	var out bytes.Buffer

	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)

	if indented {
		encoder.SetIndent(prefix, "  ")
	}

	if err := encoder.Encode(value); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}
