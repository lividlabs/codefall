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

// member is one key of a JSON object's text, and where its key and its value sit in the whole text.
type member struct {
	key     string
	raw     json.RawMessage
	keyFrom int
	from    int
	end     int
}

// object is a JSON object's members in the order the text holds them, and the offsets of its braces,
// all counted from the start of the whole text rather than of the object.
type object struct {
	members []member
	open    int
	closing int
}

// last returns the member written last under key: the one encoding/json reads when a key is written
// twice.
func (o object) last(key string) (member, bool) {
	for _, m := range slices.Backward(o.members) {
		if m.key == key {
			return m, true
		}
	}

	return member{}, false
}

// parseObject reads the JSON object that data[from:end] holds: its keys in the order the text holds
// them, with the byte range each key and value occupies, and where its braces are. It is what lets a
// write change one value and leave every other byte of a checked-in file as its author laid it out:
// decoding into a map and encoding it again would reorder every key.
func parseObject(data []byte, from, end int) (object, error) {
	text := data[from:end]
	decoder := json.NewDecoder(bytes.NewReader(text))

	start, err := decoder.Token()
	if err != nil {
		return object{}, err
	}

	if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return object{}, errNotObject
	}

	found := object{open: from + int(decoder.InputOffset()) - 1}

	for decoder.More() {
		// Only a comma and whitespace sit between the previous token and the key's opening quote.
		after := int(decoder.InputOffset())
		keyFrom := from + after + bytes.IndexByte(text[after:], '"')

		token, err := decoder.Token()
		if err != nil {
			return object{}, err
		}

		key, _ := token.(string)

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return object{}, err
		}

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		valueEnd := from + int(decoder.InputOffset())
		found.members = append(found.members, member{
			key: key, raw: raw, keyFrom: keyFrom, from: valueEnd - len(raw), end: valueEnd,
		})
	}

	if _, err := decoder.Token(); err != nil {
		return object{}, err
	}

	found.closing = from + int(decoder.InputOffset()) - 1

	return found, nil
}

// field returns one top-level value's text, and false when the object has no such key. A key
// written twice is read the way encoding/json reads it: the last one counts.
func field(data []byte, key string) (json.RawMessage, bool, error) {
	top, err := parseObject(data, 0, len(data))
	if err != nil {
		return nil, false, err
	}

	m, ok := top.last(key)

	return m.raw, ok, nil
}

// withField returns a JSON object's text with key set to what render produces: the value's bytes
// replaced where the key already is, and the key added as the object's last member where it is not.
// Nothing else in the text changes. render is handed the indentation of the key's line, so a value
// that spans lines can lay itself out one level in from it.
func withField(data []byte, key string, render func(indent string) ([]byte, error)) ([]byte, error) {
	top, err := parseObject(data, 0, len(data))
	if err != nil {
		return nil, err
	}

	return setMember(data, top, key, render)
}

// withNestedField is withField for a key inside the object a top-level key holds. The block must be
// there and be an object; the caller decides what to do when it is not.
func withNestedField(data []byte, block, key string, render func(indent string) ([]byte, error)) ([]byte, error) {
	inner, err := nestedObject(data, block)
	if err != nil {
		return nil, err
	}

	return setMember(data, inner, key, render)
}

// withoutField returns a JSON object's text with key gone, every copy of it, and nothing else
// changed. A key the object does not hold leaves the text as it was.
func withoutField(data []byte, key string) ([]byte, error) {
	for {
		top, err := parseObject(data, 0, len(data))
		if err != nil {
			return nil, err
		}

		if _, ok := top.last(key); !ok {
			return data, nil
		}

		data = removeMember(data, top, key)
	}
}

// nestedObject reads the object a top-level key holds, with offsets into the whole text.
func nestedObject(data []byte, block string) (object, error) {
	top, err := parseObject(data, 0, len(data))
	if err != nil {
		return object{}, err
	}

	m, ok := top.last(block)
	if !ok {
		return object{}, fmt.Errorf("no %s block", block)
	}

	inner, err := parseObject(data, m.from, m.end)
	if err != nil {
		return object{}, fmt.Errorf("%s: %w", block, err)
	}

	return inner, nil
}

// setMember sets key in one object of the text. An object written on one line gains the key on that
// line; one that spans lines gains it on a line of its own at its last member's indentation; an
// empty one opens onto lines one level in from the line its brace is on.
func setMember(data []byte, in object, key string, render func(indent string) ([]byte, error)) ([]byte, error) {
	if m, ok := in.last(key); ok {
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

	if len(in.members) == 0 {
		outer := indentBefore(data, in.open)
		indent := outer + "  "

		value, err := render(indent)
		if err != nil {
			return nil, err
		}

		inside := slices.Concat([]byte("\n"+indent), quoted, []byte(": "), value, []byte("\n"+outer))

		return slices.Concat(data[:in.open+1], inside, data[in.closing:]), nil
	}

	last := in.members[len(in.members)-1]
	indent := indentBefore(data, last.from)

	value, err := render(indent)
	if err != nil {
		return nil, err
	}

	separator := []byte(",\n" + indent)
	if !bytes.ContainsRune(data[in.open:in.closing], '\n') {
		separator = []byte(", ")
	}

	added := slices.Concat(separator, quoted, []byte(": "), value)

	return slices.Concat(data[:last.end], added, data[last.end:]), nil
}

// removeMember takes the last copy of key out of one object of the text, with the comma that joined
// it to its neighbour, so what is left reads as though it had never been there.
func removeMember(data []byte, in object, key string) []byte {
	at := -1

	for i, m := range in.members {
		if m.key == key {
			at = i
		}
	}

	m := in.members[at]

	switch {
	case len(in.members) == 1:
		return slices.Concat(data[:in.open+1], data[in.closing:])
	case at == 0:
		return slices.Concat(data[:m.keyFrom], data[in.members[1].keyFrom:])
	default:
		return slices.Concat(data[:in.members[at-1].end], data[m.end:])
	}
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
