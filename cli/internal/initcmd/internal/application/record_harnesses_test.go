package application

import (
	"slices"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
)

// The list is spliced in the layout the file already uses, and nothing else in the file moves: not
// the key order, not the field codefall does not know, not the spacing.
func TestSettingsWithHarnessesAppendsInTheFilesOwnLayout(t *testing.T) {
	for _, tc := range []struct {
		name  string
		data  string
		want  string
		added []string
	}{
		{
			name:  "a list spanning lines",
			data:  "{\n  \"version\": 1,\n  \"note\": \"ours\",\n  \"harnesses\": [\n    \"claude\"\n  ],\n  \"tracker\": \"beads\"\n}\n",
			want:  "{\n  \"version\": 1,\n  \"note\": \"ours\",\n  \"harnesses\": [\n    \"claude\",\n    \"codex\",\n    \"muse\"\n  ],\n  \"tracker\": \"beads\"\n}\n",
			added: []string{harness.Codex, harness.Muse},
		},
		{
			name:  "a list on one line",
			data:  "{\"harnesses\": [\"claude\"], \"tracker\": \"beads\"}",
			want:  "{\"harnesses\": [\"claude\", \"codex\"], \"tracker\": \"beads\"}",
			added: []string{harness.Codex},
		},
		{
			name: "nothing to add",
			data: "{\"harnesses\": [\"claude\", \"codex\"]}",
			want: "{\"harnesses\": [\"claude\", \"codex\"]}",
		},
		{
			name: "no harnesses list at all",
			data: "{\"tracker\": \"beads\"}",
			want: "{\"tracker\": \"beads\"}",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, added, err := settingsWithHarnesses([]byte(tc.data), []string{harness.Claude, harness.Codex, harness.Muse}[:1+len(tc.added)])
			if err != nil {
				t.Fatalf("settingsWithHarnesses: %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("settings =\n%s\nwant\n%s", got, tc.want)
			}

			if !slices.Equal(added, tc.added) {
				t.Errorf("added = %q, want %q", added, tc.added)
			}
		})
	}
}

// An upgrade for a harness the settings do not name records it, and the settings step says so; a
// run for the harnesses the settings already name leaves them as they were.
func TestRunRecordsAHarnessTheFlagAdded(t *testing.T) {
	files := settled("")
	files.files[settingsFull] = []byte("{\n  \"version\": 1,\n  \"harnesses\": [\n    \"claude\"\n  ],\n  \"tracker\": \"beads\",\n  \"test\": { \"dir\": \"testing\" }\n}\n")

	report := runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude, harness.Codex))

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	if want := "added harness codex to .codefall/settings.json"; result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	want := "{\n  \"version\": 1,\n  \"harnesses\": [\n    \"claude\",\n    \"codex\"\n  ],\n  \"tracker\": \"beads\",\n  \"test\": { \"dir\": \"testing\" }\n}\n"
	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}
