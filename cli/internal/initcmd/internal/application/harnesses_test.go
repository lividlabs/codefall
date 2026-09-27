package application

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
)

// requestFor is a settled project's run for whichever harnesses the test names.
func requestFor(names ...string) Request {
	request := beadsRequest()
	request.Harnesses = names

	return request
}

// runFor performs a whole run and hands back the report, for a test that reads several steps of it.
func runFor(t *testing.T, files *fakeFileSystem, source ExtensionSource, request Request) domain.Report {
	t.Helper()

	report, err := NewInitialize(files, toolsInstalled(), source).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	return report
}

// recordedManifestIn decodes the install record a finished run wrote.
func recordedManifestIn(t *testing.T, files *fakeFileSystem) manifest.Document {
	t.Helper()

	var recorded manifest.Document
	if err := json.Unmarshal(files.files[filepath.Join(workingDir, manifest.Name)], &recorded); err != nil {
		t.Fatalf("decode %s: %v", manifest.Name, err)
	}

	return recorded
}

// chosen is what every step reads the set through, so a name given twice is one install and the
// order a flag or a survey collected them in cannot change what a step does.
func TestChosenSortsTheSetAndDropsRepeats(t *testing.T) {
	got := chosen(requestFor(harness.Codex, harness.Claude, harness.Codex))

	if want := []string{harness.Claude, harness.Codex}; !slices.Equal(got, want) {
		t.Errorf("chosen = %q, want %q", got, want)
	}
}

// Two harnesses that read different directories are two copies, and the step names both.
func TestExtensionStepInstallsIntoEveryDirectoryTheChosenHarnessesRead(t *testing.T) {
	fetcher := newFakeExtensionSource()

	report := runFor(t, settled(""), fetcher, requestFor(harness.Claude, harness.Codex))

	result := report.Results()[1]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	// The directories are named in the order a reader would sort them, not the order the harnesses
	// were given in.
	want := "installed codefall's skills into .agents/ and .claude/ and its shared files into .codefall/"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	// Two skills directories and the one .codefall/ every run writes, whatever harnesses it is for.
	dirs := make([]string, 0, len(fetcher.calls))
	for _, call := range fetcher.calls {
		dirs = append(dirs, call.dir)
	}

	wantDirs := []string{
		filepath.Join(workingDir, ".claude"),
		filepath.Join(workingDir, ".agents"),
		filepath.Join(workingDir, ".codefall"),
	}

	if !slices.Equal(dirs, wantDirs) {
		t.Errorf("fetches = %+v, want one into each of %q", fetcher.calls, wantDirs)
	}
}

// Four of the five harnesses read .agents/, so a run for four of them copies the tree once. Each one
// records the same files, because each one reads them.
func TestExtensionStepCopiesOncePerDirectoryHoweverManyHarnessesShareIt(t *testing.T) {
	fetcher := newFakeExtensionSource()
	files := settled("")

	request := requestFor(harness.Agy, harness.Codex, harness.Muse, harness.OpenCode)
	request.CLIVersion = "v1.2.3"

	report := runFor(t, files, fetcher, request)

	if want := "installed codefall's skills into .agents/ and its shared files into .codefall/"; report.Results()[1].Detail != want {
		t.Errorf("detail = %q, want %q", report.Results()[1].Detail, want)
	}

	if len(fetcher.calls) != 2 || fetcher.calls[0].dir != filepath.Join(workingDir, ".agents") {
		t.Errorf("fetches = %+v, want one into %q and one into .codefall/",
			fetcher.calls, filepath.Join(workingDir, ".agents"))
	}

	install := manifest.Install{Version: "v1.2.3", Files: []string{".agents/skills/design/SKILL.md"}}
	want := manifest.Document{Harnesses: map[string]manifest.Install{
		harness.Agy:      install,
		harness.Codex:    install,
		harness.Muse:     install,
		harness.OpenCode: install,
	}, Shared: sharedInstall("v1.2.3")}

	if got := recordedManifestIn(t, files); !reflect.DeepEqual(got, want) {
		t.Errorf("manifest = %+v, want %+v", got, want)
	}
}

// The recorded paths say which directory a file landed in. Four harnesses share one directory, so a
// path relative to the harness's own skills directory could not be read back.
func TestTheManifestRecordsWhereEachHarnessFilesLanded(t *testing.T) {
	files := settled("")

	request := requestFor(harness.Claude, harness.Codex)
	request.CLIVersion = "v1.2.3"

	runFor(t, files, newFakeExtensionSource(), request)

	want := manifest.Document{Harnesses: map[string]manifest.Install{
		harness.Claude: {Version: "v1.2.3", Files: []string{".claude/skills/design/SKILL.md"}},
		harness.Codex:  {Version: "v1.2.3", Files: []string{".agents/skills/design/SKILL.md"}},
	}, Shared: sharedInstall("v1.2.3")}

	if got := recordedManifestIn(t, files); !reflect.DeepEqual(got, want) {
		t.Errorf("manifest = %+v, want %+v", got, want)
	}
}

// A run for one harness has done nothing to another harness's install, and erasing the record of it
// would make the next run for that harness repeat work it has already done. This is what a manifest
// naming a single harness could not do.
func TestTheManifestKeepsWhatAnEarlierRunRecordedForAnotherHarness(t *testing.T) {
	files := settled("")
	files.files[filepath.Join(workingDir, manifest.Name)] = []byte(
		`{"harnesses": {"claude": {"version": "v0.1.0", "files": [".claude/skills/old/SKILL.md"]}}}`)

	request := requestFor(harness.Codex)
	request.CLIVersion = "v1.2.3"

	runFor(t, files, newFakeExtensionSource(), request)

	want := manifest.Document{Harnesses: map[string]manifest.Install{
		harness.Claude: {Version: "v0.1.0", Files: []string{".claude/skills/old/SKILL.md"}},
		harness.Codex:  {Version: "v1.2.3", Files: []string{".agents/skills/design/SKILL.md"}},
	}, Shared: sharedInstall("v1.2.3")}

	if got := recordedManifestIn(t, files); !reflect.DeepEqual(got, want) {
		t.Errorf("manifest = %+v, want %+v", got, want)
	}
}

// The hook step registers with each chosen harness and reports every registration. A single-harness
// run still reports that harness's own sentence, which is what the step has always said.
func TestHookStepRegistersWithEveryChosenHarness(t *testing.T) {
	files := settled("")

	report := runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude, harness.Codex))

	result := hookResult(t, report)
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "merged codefall's hooks into .claude/settings.json; " +
		"merged codefall's hooks into .codex/hooks.json"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	for _, dest := range []string{claudeFull, filepath.Join(workingDir, ".codex/hooks.json")} {
		if _, landed := files.files[dest]; !landed {
			t.Errorf("nothing landed at %q; files = %v", dest, keysOf(files))
		}
	}
}

// Each harness's registration keeps its own shape: two are merged into the file the harness reads,
// one is a copied plugin, and one takes codefall only as skills. The step reports all four.
func TestHookStepReportsEachHarnessOwnOutcome(t *testing.T) {
	files := settled("")

	report := runFor(t, files, newFakeExtensionSource(),
		requestFor(harness.Agy, harness.Codex, harness.Muse, harness.OpenCode))

	result := hookResult(t, report)
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "merged codefall's hooks into .agents/hooks.json; " +
		"merged codefall's hooks into .codex/hooks.json; " +
		"codefall has no hooks for muse; " +
		"installed codefall's plugin at .opencode/plugins/codefall.js"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// Nothing changed anywhere is still a skip, however many harnesses were asked about: a rerun that
// finds every registration in place has nothing to report having done.
func TestHookStepSkipsWhenNoHarnessNeededAnything(t *testing.T) {
	files := settled("")
	request := requestFor(harness.Muse, harness.OpenCode)

	runFor(t, files, newFakeExtensionSource(), request)

	result := hookResult(t, runFor(t, files, newFakeExtensionSource(), request))
	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	want := "codefall has no hooks for muse; " +
		"codefall's plugin at .opencode/plugins/codefall.js is already installed"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// CLAUDE.md is Claude Code's file, so a run that includes Claude Code writes it whatever else it is
// for, and a run that does not include it writes nothing.
func TestTheClaudePointerFollowsWhetherClaudeCodeIsAmongThem(t *testing.T) {
	for _, tc := range []struct {
		name    string
		names   []string
		pointer bool
	}{
		{name: "claude beside another harness", names: []string{harness.Claude, harness.Codex}, pointer: true},
		{name: "harnesses that do not read it", names: []string{harness.Codex, harness.Muse}, pointer: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("")

			runFor(t, files, newFakeExtensionSource(), requestFor(tc.names...))

			if _, written := files.files[claudeMemoryFull]; written != tc.pointer {
				t.Errorf("CLAUDE.md written = %v, want %v", written, tc.pointer)
			}
		})
	}
}

// A run for no harness has nowhere to install. Every step after preflight would report doing nothing
// rather than failing, so the run refuses to start instead — where nothing has been touched yet.
func TestPreflightRefusesARunForNoHarness(t *testing.T) {
	files := newFakeFileSystem()

	_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(), requestFor(), nil)

	if want := "no harness to set up"; err == nil || err.Error() != want {
		t.Errorf("Run error = %v, want exactly %q", err, want)
	}

	if len(files.files) != 0 || len(files.made) != 0 {
		t.Errorf("wrote %v and created %v, want nothing touched", files.files, files.made)
	}
}
