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

// A run that writes the settings whole has nothing old left in them, and the manifest is the one file
// that can still carry a former spelling.
func TestRunWithForceRewritesAFormerHarnessNameInTheManifest(t *testing.T) {
	files := newFakeFileSystem()
	files.files[settingsFull] = []byte(formerSettings)
	files.files[manifestFull] = []byte(formerManifest)

	request := beadsRequest()
	request.Force = true

	report := runFor(t, files, newFakeExtensionSource(), request)

	detail := report.Results()[0].Detail
	if !strings.HasPrefix(detail, "wrote .codefall/settings.json (") ||
		!strings.HasSuffix(detail, "; renamed harness claude-code to claude in .codefall/manifest.json") {
		t.Errorf("settings detail = %q, want the write and the manifest rename", detail)
	}

	if strings.Contains(string(files.files[settingsFull]), "claude-code") {
		t.Errorf("settings.json = %s, want no former spelling", files.files[settingsFull])
	}

	if _, left := recordedManifestIn(t, files).Harnesses["claude-code"]; left {
		t.Errorf("manifest = %s, want no former spelling", files.files[manifestFull])
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

func TestRespelledSettingsChangesOnlyTheNamesInTheHarnessesList(t *testing.T) {
	for _, tc := range []struct {
		name        string
		body        string
		want        string
		wantRenamed []string
	}{
		{
			name:        "both former spellings, beside a current name",
			body:        `{"harnesses": ["antigravity", "codex", "claude-code"]}`,
			want:        `{"harnesses": ["agy", "codex", "claude"]}`,
			wantRenamed: []string{"antigravity", "claude-code"},
		},
		{
			// JSON may escape any character, and the name is what the file means, not how it spells it.
			name:        "a former spelling written with an escape",
			body:        `{"harnesses": ["claude-code"]}`,
			want:        `{"harnesses": ["claude"]}`,
			wantRenamed: []string{"claude-code"},
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
			got, renamed, err := respelledSettings([]byte(tc.body))
			if err != nil {
				t.Fatalf("respelledSettings: %v", err)
			}

			if string(got) != tc.want {
				t.Errorf("body = %s, want %s", got, tc.want)
			}

			if !slices.Equal(renamed, tc.wantRenamed) {
				t.Errorf("renamed = %q, want %q", renamed, tc.wantRenamed)
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

			got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).FormerHarnessNames(workingDir)
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

	initialize := NewInitialize(files, toolsInstalled(), newFakeExtensionSource())

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
