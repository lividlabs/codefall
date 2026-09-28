// Package userfile is the one definition of the .codefall/user.json format: the settings that
// describe the person at the keyboard rather than the project. The file is never checked in —
// codefall init adds it to .gitignore — so two people on one project each keep their own. doctor
// reads it, the skills read it through the shared preflight script, and a codefall command writes
// it, so it lives here rather than in any one component's domain (ADR-003).
//
// It is a sibling of the settings module rather than a second document type inside it. settings is
// the checked-in project file, and everything in it is shared by everyone who clones the project;
// this file belongs to one person on one machine, and changes for different reasons and at different
// times. Keeping the two apart keeps settings to one concern, and makes it plain which file a new
// field belongs in.
//
// It is a pure shared module: it imports the standard library and nothing else, which is what lets
// domain/ and application/ name it. Adding a dependency here breaks that permission and fails the
// pure-shared-modules rule in .golangci.yml.
package userfile

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// Document is a decoded user file: the generic shape a JSON decoder produces. Validation reads that
// shape without knowing it came from JSON, as the settings module does.
type Document map[string]any

// The published constants of the user file's format. The schema test in this package holds
// schemas/user.schema.json equal to these.
const (
	// Name is where the file lives, beside settings.json. It is also the .gitignore entry.
	Name     = ".codefall/user.json"
	Version  = 1
	SchemaID = "https://raw.githubusercontent.com/lividlabs/codefall-cli/main/schemas/user.schema.json"
	// FieldVersion is the format's version, which every file carries.
	FieldVersion = "version"
	// FieldPersona is who the person at the keyboard works as. It is optional, and absent means
	// DefaultPersona.
	FieldPersona = "persona"
	// The personas codefall knows.
	PersonaEngineer       = "engineer"
	PersonaProductManager = "product-manager"
	// DefaultPersona is the persona of a person who has not said, and of a file that is missing or
	// cannot be read: every skill was written for an engineer before the field existed.
	DefaultPersona = PersonaEngineer
	// GitIgnoreComment says why the .gitignore line is there, for whoever finds the file later.
	GitIgnoreComment = "# codefall user file: settings for the person at this keyboard, never checked in."
)

// personas is every persona codefall knows, as a set, so the validator and the schema read one
// definition.
var personas = map[string]bool{
	PersonaEngineer:       true,
	PersonaProductManager: true,
}

// Personas returns the known persona names, sorted.
func Personas() []string {
	names := make([]string, 0, len(personas))
	for name := range personas {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// ParsePersona returns the persona when it is one codefall knows, and an error listing the known
// ones when it is not.
func ParsePersona(name string) (string, error) {
	if personas[name] {
		return name, nil
	}

	return "", fmt.Errorf("unknown persona %q (known personas: %s)", name, strings.Join(Personas(), ", "))
}

// Persona returns the persona a document says the person works as. A document that says nothing,
// or says something Validate would reject, yields DefaultPersona: a caller that cares whether the
// file is valid has Validate for that, and every other caller needs an answer it can act on.
func Persona(doc Document) string {
	value, present := lookup(doc, FieldPersona)
	if !present {
		return DefaultPersona
	}

	name, ok := value.(string)
	if !ok || !personas[name] {
		return DefaultPersona
	}

	return name
}

// Validate reports every problem with a user file at once, each as "path: reason". An empty result
// means the file is acceptable. Keys the format does not name are ignored, so a file a later codefall
// wrote still reads.
func Validate(doc Document) []string {
	var problems []string

	if value, present := lookup(doc, "$schema"); present {
		if _, ok := value.(string); !ok {
			problems = append(problems, "$schema: must be a string")
		}
	}

	version, present := lookup(doc, FieldVersion)

	switch {
	case !present:
		problems = append(problems, FieldVersion+": missing")
	case !isVersion(version):
		problems = append(problems, fmt.Sprintf("%s: must be %d", FieldVersion, Version))
	}

	if value, present := lookup(doc, FieldPersona); present {
		name, ok := value.(string)

		switch {
		case !ok:
			problems = append(problems, FieldPersona+": must be a string")
		case !personas[name]:
			problems = append(problems, FieldPersona+": "+unknownValue(name, Personas()))
		}
	}

	return problems
}

// lookup reports whether a key carries a usable value. A null and an empty string both count as
// missing, the rule the settings file follows, so a half-filled file reads the same as an absent
// field.
func lookup(doc Document, name string) (any, bool) {
	value, ok := doc[name]
	if !ok || value == nil {
		return nil, false
	}

	if text, isText := value.(string); isText && text == "" {
		return nil, false
	}

	return value, true
}

// isVersion accepts the one version there is. JSON has one number type, so the version arrives as a
// float64 and a fractional one has to be refused here rather than by the decoder.
func isVersion(v any) bool {
	number, ok := v.(float64)

	return ok && number == math.Trunc(number) && int(number) == Version
}

// unknownValue is the value it read, and the ones that would have been accepted, because that is
// what the reader needs next.
func unknownValue(name string, expected []string) string {
	quoted := make([]string, 0, len(expected))
	for _, value := range expected {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return fmt.Sprintf("unknown value %q (expected %s)", name, strings.Join(quoted, ", "))
}
