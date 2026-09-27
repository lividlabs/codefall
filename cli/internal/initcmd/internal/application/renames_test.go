package application

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
)

var manifestFull = filepath.Join(workingDir, manifest.Name)

// formerSettings is a project set up before its harnesses were named for their binaries, in the
// shape a person might have left it: keys in their own order, a field codefall does not know, and a
// second mention of an old spelling outside the harnesses list, which is the project's own text and
// not a harness name.
const formerSettings = `{
  "version": 1,
  "note": "we moved off claude-code last year",
  "harnesses": [ "claude-code",
      "codex" ],
  "tracker": "beads",
  "beads": {},
  "test": { "dir": "testing" }
}
`

// formerManifest is the record a finished run left before the rename.
const formerManifest = `{
  "harnesses": {
    "claude-code": {"version": "v0.19.0", "files": [".claude/skills/design/SKILL.md"]},
    "codex": {"version": "v0.19.0", "files": [".agents/skills/design/SKILL.md"]}
  }
}`

// A rerun rewrites the old spelling in both files and says so. The settings file changes in the one
// name and nowhere else, because it is the project's file and every other byte of it is theirs.
func TestRunRewritesAFormerHarnessNameInTheSettingsAndTheManifest(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(formerSettings)
	files.files[manifestFull] = []byte(formerManifest)

	request := beadsRequest()
	request.Harnesses = []string{harness.Claude, harness.Codex}
	request.CLIVersion = "v0.20.0"

	report := runFor(t, files, newFakeExtensionSource(), request)

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	want := "renamed harness claude-code to claude in .codefall/settings.json and .codefall/manifest.json"
	if result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	wantSettings := strings.Replace(formerSettings, `[ "claude-code",`, `[ "claude",`, 1)
	if got := string(files.files[settingsFull]); got != wantSettings {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, wantSettings)
	}

	recorded := recordedManifestIn(t, files)
	if got, want := slices.Sorted(maps.Keys(recorded.Harnesses)), []string{harness.Claude, harness.Codex}; !slices.Equal(got, want) {
		t.Errorf("manifest harnesses = %q, want %q", got, want)
	}
}

// Files that already use the current names are left as they were, and the step says what it always
// said about settings that are already there.
func TestRunLeavesCurrentHarnessNamesAlone(t *testing.T) {
	current := strings.ReplaceAll(formerSettings, `"claude-code",`, `"claude",`)

	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(current)

	report := runFor(t, files, newFakeExtensionSource(), beadsRequest())

	if got := report.Results()[0].Outcome; got != domain.OutcomeSkipped {
		t.Errorf("settings outcome = %v, want SKIPPED", got)
	}

	if got := string(files.files[settingsFull]); got != current {
		t.Errorf("settings.json =\n%s\nwant it unchanged", got)
	}
}

// formerBesideCurrent is a settings file that names one harness under both spellings, one per line,
// which is what a person who added the new name by hand before upgrading leaves.
const formerBesideCurrent = `{
  "version": 1,
  "harnesses": [
    "claude",
    "claude-code"
  ],
  "tracker": "beads",
  "test": { "dir": "testing" }
}
`

// A rerun over a list that already names the harness under its current name removes the old
// spelling rather than rewriting it into a second copy, and the file keeps its layout. The manifest
// holding both entries keeps the current one, and the step says the old one was removed from both.
func TestRunRemovesAFormerHarnessNameTheFilesAlreadyListUnderItsCurrentName(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(formerBesideCurrent)
	files.files[manifestFull] = []byte(`{"harnesses": {
  "claude-code": {"version": "v0.19.0", "files": ["old"]},
  "claude": {"version": "v0.20.0", "files": [".claude/skills/design/SKILL.md"]}
}}`)

	request := beadsRequest()
	request.Harnesses = []string{harness.Claude}
	request.CLIVersion = "v0.20.0"

	report := runFor(t, files, newFakeExtensionSource(), request)

	result := report.Results()[0]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("settings outcome = %v, want DONE", result.Outcome)
	}

	want := "removed harness claude-code from .codefall/settings.json and .codefall/manifest.json, already listed as claude"
	if result.Detail != want {
		t.Errorf("settings detail = %q, want %q", result.Detail, want)
	}

	wantSettings := strings.Replace(formerBesideCurrent, "\"claude\",\n    \"claude-code\"", `"claude"`, 1)
	if got := string(files.files[settingsFull]); got != wantSettings {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, wantSettings)
	}

	recorded := recordedManifestIn(t, files)
	if got, want := slices.Sorted(maps.Keys(recorded.Harnesses)), []string{harness.Claude}; !slices.Equal(got, want) {
		t.Errorf("manifest harnesses = %q, want %q", got, want)
	}
}

// Each file is described by what happened to it: the settings already named the harness, so the old
// spelling went, while the manifest had only the old spelling, so it was renamed.
func TestRunDescribesARemovalAndARenameInDifferentFilesSeparately(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(formerBesideCurrent)
	files.files[manifestFull] = []byte(formerManifest)

	request := beadsRequest()
	request.Harnesses = []string{harness.Claude}
	request.CLIVersion = "v0.20.0"

	report := runFor(t, files, newFakeExtensionSource(), request)

	want := "removed harness claude-code from .codefall/settings.json, already listed as claude; " +
		"renamed harness claude-code to claude in .codefall/manifest.json"
	if got := report.Results()[0].Detail; got != want {
		t.Errorf("settings detail = %q, want %q", got, want)
	}
}

func TestRespelledSettingsChangesOnlyTheNamesInTheHarnessesList(t *testing.T) {
	for _, tc := range []struct {
		name        string
		body        string
		want        string
		wantRenamed []string
		wantRemoved []string
	}{
		{
			name:        "a former spelling alone",
			body:        `{"harnesses": ["claude-code"]}`,
			want:        `{"harnesses": ["claude"]}`,
			wantRenamed: []string{"claude-code"},
		},
		{
			name:        "both former spellings, beside a current name",
			body:        `{"harnesses": ["antigravity", "codex", "claude-code"]}`,
			want:        `{"harnesses": ["agy", "codex", "claude"]}`,
			wantRenamed: []string{"antigravity", "claude-code"},
		},
		{
			// JSON may escape any character, and the name is what the file means, not how it spells it.
			name:        "a former spelling written with an escape",
			body:        `{"harnesses": ["claude\u002dcode"]}`,
			want:        `{"harnesses": ["claude"]}`,
			wantRenamed: []string{"claude-code"},
		},
		{
			name:        "the current name first, the former spelling last",
			body:        `{"harnesses": ["claude", "claude-code"]}`,
			want:        `{"harnesses": ["claude"]}`,
			wantRemoved: []string{"claude-code"},
		},
		{
			name:        "the former spelling first, the current name last",
			body:        `{"harnesses": ["claude-code", "claude"]}`,
			want:        `{"harnesses": ["claude"]}`,
			wantRemoved: []string{"claude-code"},
		},
		{
			name:        "the former spelling between two others, one per line",
			body:        "{\n  \"harnesses\": [\n    \"claude\",\n    \"claude-code\",\n    \"codex\"\n  ]\n}\n",
			want:        "{\n  \"harnesses\": [\n    \"claude\",\n    \"codex\"\n  ]\n}\n",
			wantRemoved: []string{"claude-code"},
		},
		{
			name:        "the former spelling first, one per line",
			body:        "{\n  \"harnesses\": [\n    \"claude-code\",\n    \"claude\"\n  ]\n}\n",
			want:        "{\n  \"harnesses\": [\n    \"claude\"\n  ]\n}\n",
			wantRemoved: []string{"claude-code"},
		},
		{
			name:        "both former spellings ahead of both current names",
			body:        `{"harnesses": ["antigravity", "claude-code", "claude", "agy"]}`,
			want:        `{"harnesses": ["claude", "agy"]}`,
			wantRemoved: []string{"antigravity", "claude-code"},
		},
		{
			name:        "one former spelling renamed and another removed",
			body:        `{"harnesses": ["antigravity", "claude", "claude-code"]}`,
			want:        `{"harnesses": ["agy", "claude"]}`,
			wantRenamed: []string{"antigravity"},
			wantRemoved: []string{"claude-code"},
		},
		{
			// The first is renamed, and then the list names the harness under its current name.
			name:        "the former spelling twice",
			body:        `{"harnesses": ["claude-code", "codex", "claude-code"]}`,
			want:        `{"harnesses": ["claude", "codex"]}`,
			wantRenamed: []string{"claude-code"},
			wantRemoved: []string{"claude-code"},
		},
		{
			name:        "an element that is not a name stays where it was",
			body:        `{"harnesses": ["claude", 1, "claude-code"]}`,
			want:        `{"harnesses": ["claude", 1]}`,
			wantRemoved: []string{"claude-code"},
		},
		{
			// A block of the project's own that happens to hold a harnesses field is not the one the
			// format defines.
			name: "a harnesses field inside another block",
			body: `{"custom": {"harnesses": ["claude-code"]}, "harnesses": ["codex"]}`,
			want: `{"custom": {"harnesses": ["claude-code"]}, "harnesses": ["codex"]}`,
		},
		{
			name: "a harnesses field that is not a list",
			body: `{"harnesses": "claude-code"}`,
			want: `{"harnesses": "claude-code"}`,
		},
		{
			name: "current names only",
			body: `{"harnesses": ["claude", "agy"]}`,
			want: `{"harnesses": ["claude", "agy"]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, edits, err := respelledSettings([]byte(tc.body))
			if err != nil {
				t.Fatalf("respelledSettings: %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("body = %s, want %s", got, tc.want)
			}

			if !slices.Equal(edits.renamed, tc.wantRenamed) {
				t.Errorf("renamed = %q, want %q", edits.renamed, tc.wantRenamed)
			}

			if !slices.Equal(edits.removed, tc.wantRemoved) {
				t.Errorf("removed = %q, want %q", edits.removed, tc.wantRemoved)
			}
		})
	}
}

// The question presentation asks before it calls a rerun a no-op: either file carrying an old
// spelling is work to do.
func TestFormerHarnessNamesReadsBothFiles(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		manifest string
		want     []string
	}{
		{name: "both files", settings: formerSettings, manifest: formerManifest, want: []string{"claude-code"}},
		{
			name:     "the manifest alone",
			settings: `{"harnesses": ["agy"]}`,
			manifest: `{"harnesses": {"antigravity": {"version": "v0.19.0"}}}`,
			want:     []string{"antigravity"},
		},
		{name: "a settings file naming both spellings", settings: formerBesideCurrent, want: []string{"claude-code"}},
		{name: "neither", settings: `{"harnesses": ["claude"]}`},
		{name: "no files at all"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem()

			if tc.settings != "" {
				files.files[settingsFull] = []byte(tc.settings)
			}

			if tc.manifest != "" {
				files.files[manifestFull] = []byte(tc.manifest)
			}

			got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).FormerHarnessNames(workingDir)
			if err != nil {
				t.Fatalf("FormerHarnessNames: %v", err)
			}

			if !slices.Equal(got, tc.want) {
				t.Errorf("FormerHarnessNames = %q, want %q", got, tc.want)
			}
		})
	}
}

// A rerun installs for the harnesses the settings record and holds the manifest against them, both
// under the names the harnesses have now, so an old spelling in either file cannot make a harness
// look uninstalled.
func TestARerunReadsFormerHarnessNamesAsTheCurrentOnes(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(formerSettings)
	files.files[manifestFull] = []byte(formerManifest)

	initialize := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges())

	chosen, err := initialize.ChosenHarnesses(workingDir)
	if err != nil {
		t.Fatalf("ChosenHarnesses: %v", err)
	}

	if got, want := chosen.OrEmpty(), []string{harness.Claude, harness.Codex}; !slices.Equal(got, want) {
		t.Errorf("ChosenHarnesses = %q, want %q", got, want)
	}

	installed, err := initialize.Installed(workingDir)
	if err != nil {
		t.Fatalf("Installed: %v", err)
	}

	want := map[string]string{harness.Claude: "v0.19.0", harness.Codex: "v0.19.0"}
	if got := installed.OrEmpty().Versions; !maps.Equal(got, want) {
		t.Errorf("Installed versions = %v, want %v", got, want)
	}
}
