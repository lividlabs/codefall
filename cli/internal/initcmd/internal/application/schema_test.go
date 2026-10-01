package application

import (
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// A former URL is replaced where it stands, and nothing else in the file moves: not the key order,
// not the field codefall does not know, not the spacing. A URL codefall never wrote is left alone.
func TestSettingsWithCurrentSchemaReplacesOnlyAFormerURL(t *testing.T) {
	former := settings.FormerSchemaIDs[0]

	for _, tc := range []struct {
		name    string
		data    string
		want    string
		changed bool
	}{
		{
			name:    "a former URL",
			data:    "{\n  \"$schema\": \"" + former + "\",\n  \"note\": \"ours\",\n  \"version\": 1\n}\n",
			want:    "{\n  \"$schema\": \"" + settings.SchemaID + "\",\n  \"note\": \"ours\",\n  \"version\": 1\n}\n",
			changed: true,
		},
		{
			name:    "a former URL that is not the first key",
			data:    "{\"version\": 1, \"$schema\":\"" + former + "\"}",
			want:    "{\"version\": 1, \"$schema\":\"" + settings.SchemaID + "\"}",
			changed: true,
		},
		{
			name: "the current URL",
			data: "{\"$schema\": \"" + settings.SchemaID + "\"}",
			want: "{\"$schema\": \"" + settings.SchemaID + "\"}",
		},
		{
			name: "a URL of the project's own",
			data: "{\"$schema\": \"./schemas/settings.schema.json\"}",
			want: "{\"$schema\": \"./schemas/settings.schema.json\"}",
		},
		{
			name: "no $schema at all",
			data: "{\"version\": 1}",
			want: "{\"version\": 1}",
		},
		{
			name: "a $schema that is not a string",
			data: "{\"$schema\": 1}",
			want: "{\"$schema\": 1}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, err := settingsWithCurrentSchema([]byte(tc.data))
			if err != nil {
				t.Fatalf("settingsWithCurrentSchema: %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("settings =\n%s\nwant\n%s", got, tc.want)
			}

			if changed != tc.changed {
				t.Errorf("changed = %v, want %v", changed, tc.changed)
			}
		})
	}
}

// An upgrade of settings that name a former schema URL points them at the current one, and the
// settings step says so.
func TestRunPointsAFormerSchemaAtTheCurrentOne(t *testing.T) {
	files := settled("")
	files.files[settingsFull] = []byte("{\n  \"$schema\": \"" + settings.FormerSchemaIDs[0] + "\",\n  \"version\": 1,\n  \"harnesses\": [\n    \"claude\"\n  ],\n  \"tracker\": \"beads\",\n  \"test\": { \"dir\": \"testing\" }\n}\n")

	report := runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude))

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	if want := "pointed $schema in .codefall/settings.json at the current schema"; result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	want := "{\n  \"$schema\": \"" + settings.SchemaID + "\",\n  \"version\": 1,\n  \"harnesses\": [\n    \"claude\"\n  ],\n  \"tracker\": \"beads\",\n  \"test\": { \"dir\": \"testing\" }\n}\n"
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}
