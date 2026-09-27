package userfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{name: "a version and nothing else", doc: Document{"version": 1.0}},
		{name: "the engineer persona", doc: Document{"version": 1.0, "persona": "engineer"}},
		{name: "the product-manager persona", doc: Document{"version": 1.0, "persona": "product-manager"}},
		{name: "a schema reference", doc: Document{"$schema": SchemaID, "version": 1.0}},
		{name: "a key the format does not name", doc: Document{"version": 1.0, "editor": "vim"}},
		{name: "a null persona reads as absent", doc: Document{"version": 1.0, "persona": nil}},
		{name: "an empty persona reads as absent", doc: Document{"version": 1.0, "persona": ""}},
		{
			name: "no version",
			doc:  Document{"persona": "engineer"},
			want: []string{"version: missing"},
		},
		{
			name: "another version",
			doc:  Document{"version": 2.0},
			want: []string{"version: must be 1"},
		},
		{
			name: "a fractional version",
			doc:  Document{"version": 1.5},
			want: []string{"version: must be 1"},
		},
		{
			name: "an unknown persona",
			doc:  Document{"version": 1.0, "persona": "designer"},
			want: []string{`persona: unknown value "designer" (expected "engineer", "product-manager")`},
		},
		{
			name: "a persona that is not a string",
			doc:  Document{"version": 1.0, "persona": 3.0},
			want: []string{"persona: must be a string"},
		},
		{
			name: "every problem at once",
			doc:  Document{"$schema": true, "persona": "designer"},
			want: []string{
				"$schema: must be a string",
				"version: missing",
				`persona: unknown value "designer" (expected "engineer", "product-manager")`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Validate = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPersona(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want string
	}{
		{name: "no document", doc: nil, want: PersonaEngineer},
		{name: "no persona", doc: Document{"version": 1.0}, want: PersonaEngineer},
		{name: "engineer", doc: Document{"persona": "engineer"}, want: PersonaEngineer},
		{name: "product-manager", doc: Document{"persona": "product-manager"}, want: PersonaProductManager},
		{name: "an unknown persona", doc: Document{"persona": "designer"}, want: PersonaEngineer},
		{name: "a persona that is not a string", doc: Document{"persona": true}, want: PersonaEngineer},
		{name: "an empty persona", doc: Document{"persona": ""}, want: PersonaEngineer},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Persona(tc.doc); got != tc.want {
				t.Errorf("Persona = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDefaultPersonaIsEngineer(t *testing.T) {
	if DefaultPersona != PersonaEngineer {
		t.Errorf("DefaultPersona = %q, want %q", DefaultPersona, PersonaEngineer)
	}
}

func TestParsePersona(t *testing.T) {
	for _, name := range Personas() {
		if got, err := ParsePersona(name); err != nil || got != name {
			t.Errorf("ParsePersona(%q) = %q, %v; want it back and no error", name, got, err)
		}
	}

	_, err := ParsePersona("designer")
	if err == nil {
		t.Fatal("ParsePersona(designer) returned no error")
	}

	const want = `unknown persona "designer" (known personas: engineer, product-manager)`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

// schemas/user.schema.json is the published definition of .codefall/user.json; this package
// validates the same shape by hand, so no schema library ships in the binary. This test holds the
// two equal. `go test` runs with the package directory as its working directory, so the relative
// path is stable.
const schemaPath = "../../../schemas/user.schema.json"

func TestSchemaMatchesTheFormat(t *testing.T) {
	data, err := os.ReadFile(filepath.FromSlash(schemaPath))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	var schema struct {
		Schema     string   `json:"$schema"`
		ID         string   `json:"$id"`
		Required   []string `json:"required"`
		Properties struct {
			Version struct {
				Const int `json:"const"`
			} `json:"version"`
			Persona struct {
				Type    string   `json:"type"`
				Enum    []string `json:"enum"`
				Default string   `json:"default"`
			} `json:"persona"`
		} `json:"properties"`
	}

	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}

	if schema.ID != SchemaID {
		t.Errorf("$id = %q, want %q", schema.ID, SchemaID)
	}

	if want := "https://json-schema.org/draft/2020-12/schema"; schema.Schema != want {
		t.Errorf("$schema = %q, want %q", schema.Schema, want)
	}

	// The persona is optional: a file that says nothing about it means the default.
	if want := []string{FieldVersion}; !slices.Equal(schema.Required, want) {
		t.Errorf("required = %q, want %q", schema.Required, want)
	}

	if schema.Properties.Version.Const != Version {
		t.Errorf("properties.version.const = %d, want %d", schema.Properties.Version.Const, Version)
	}

	persona := schema.Properties.Persona

	if persona.Type != "string" {
		t.Errorf("properties.persona.type = %q, want string", persona.Type)
	}

	if got := slices.Sorted(slices.Values(persona.Enum)); !slices.Equal(got, Personas()) {
		t.Errorf("properties.persona.enum = %q, want %q", got, Personas())
	}

	if persona.Default != DefaultPersona {
		t.Errorf("properties.persona.default = %q, want %q", persona.Default, DefaultPersona)
	}
}
