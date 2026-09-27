package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// The definitions the embedded tree serves, one per harness. They mirror extensions/hooks/<harness>/,
// and the merge reads what comes back through the same doorway the real FS uses.
// notice is the session-start notice as a definition names it: a codefall script, so the merge
// knows the entry as codefall's own, beside a prime that is nobody's script.
const notice = `"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-session-notice.sh" --hook-json`

var hookDefinitions = map[string][]byte{
	"hooks/claude/hooks.json": []byte(`{
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}],
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(notice) + `}]}
    ]
  }
}`),
	"hooks/codex/hooks.json": []byte(`{
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}],
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
      {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(notice) + `}]}
    ]
  }
}`),
	"hooks/agy/hooks.json": []byte(`{
  "codefall-merge-guard": {
    "PreToolUse": [{"matcher": "run_command", "hooks": [{"command": "guard --antigravity"}]}]
  }
}`),
	"hooks/opencode/codefall.js": []byte("// the adapter\n"),
}

// quoted is a command as the JSON string literal a definition holds it as, so a test can write a
// command with quotes in it the way the shipped definitions do.
func quoted(command string) string {
	encoded, err := json.Marshal(command)
	if err != nil {
		panic(err)
	}

	return string(encoded)
}

// hookResult is what the fourth step did in a run that got that far.
func hookResult(t *testing.T, report domain.Report) domain.StepResult {
	t.Helper()

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	return results[3]
}

// The three JSON harnesses merge their definitions; OpenCode copies its plugin. A harness in the
// table without a definition would be a bug the table would not have room to hide.
func TestHookRegistersWhatTheTableSays(t *testing.T) {
	for _, tc := range []struct {
		harness string
		dest    string
	}{
		{harness.Claude, claudeFull},
		{harness.Codex, filepath.Join(workingDir, ".codex/hooks.json")},
		{harness.Agy, filepath.Join(workingDir, ".agents/hooks.json")},
		{harness.OpenCode, filepath.Join(workingDir, ".opencode/plugins/codefall.js")},
	} {
		t.Run(tc.harness, func(t *testing.T) {
			files := settled("")

			report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequestHarness(tc.harness), nil)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			result := hookResult(t, report)
			if result.Outcome != domain.OutcomeDone {
				t.Errorf("outcome = %v, want DONE", result.Outcome)
			}

			if _, ok := files.files[tc.dest]; !ok {
				t.Errorf("nothing landed at %q; files = %v", tc.dest, keysOf(files))
			}
		})
	}
}

// beadsRequestHarness is beadsRequest with the harness the test wants.
func beadsRequestHarness(name string) Request {
	request := beadsRequest()
	request.Harnesses = []string{name}
	return request
}

// keysOf maps the fake's written files, for failure messages.
func keysOf(files *fakeFileSystem) []string {
	keys := make([]string, 0, len(files.files))
	for path := range files.files {
		keys = append(keys, path)
	}
	slices.Sort(keys)
	return keys
}

// assertKeysSurvive holds the file to what it said before the run: every key it had is still there,
// with the same value. It is decoded rather than compared as bytes, because encoding/json sorts an
// object's keys and the file comes back in its order, not the one it was written in.
func assertKeysSurvive(t *testing.T, got []byte, before string) {
	t.Helper()

	var was, now map[string]any

	if err := json.Unmarshal([]byte(before), &was); err != nil {
		t.Fatalf("the settings the test started from are not an object: %v", err)
	}

	if err := json.Unmarshal(got, &now); err != nil {
		t.Fatalf("settings after the run are not an object: %v", err)
	}

	for _, key := range slices.Sorted(maps.Keys(was)) {
		if !reflect.DeepEqual(now[key], was[key]) {
			t.Errorf("%s = %#v, want %#v — the run lost what the file said", key, now[key], was[key])
		}
	}
}

// The merge joins what the project already has rather than replacing it: another SessionStart hook
// stays, and so does every key the file keeps for its own reasons.
func TestHookMergeKeepsWhatTheFileAlreadySays(t *testing.T) {
	const before = `{
  "enabledPlugins": {"codefall@codefall": true},
  "permissions": {"allow": ["Bash(git status:*)"]},
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "other"}]}],
    "SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "echo hello"}]}]
  }
}`

	files := settled(before)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	got := files.files[claudeFull]

	assertKeysSurvive(t, got, `{
  "enabledPlugins": {"codefall@codefall": true},
  "permissions": {"allow": ["Bash(git status:*)"]}
}`)

	var document struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}

	if err := json.Unmarshal(got, &document); err != nil {
		t.Fatalf("decode what was written: %v", err)
	}

	if got := document.Hooks["PreToolUse"]; len(got) != 2 || got[1].Hooks[0].Command != "guard" {
		t.Errorf("PreToolUse = %+v, want the project's entry kept and codefall's merged", got)
	}

	sessions := document.Hooks["SessionStart"]
	if len(sessions) != 3 || sessions[0].Hooks[0].Command != "echo hello" ||
		sessions[1].Hooks[0].Command != "bd prime --hook-json" ||
		sessions[2].Hooks[0].Command != notice {
		t.Errorf("SessionStart = %+v, want the project's entry first, then the prime and the notice",
			sessions)
	}
}

// A destination that already has every entry in the definition — dedupe is per entry, not all or
// nothing — is left alone, whatever else the file says.
func TestHookMergeSkipsWhatIsAlreadyThere(t *testing.T) {
	before := `{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}],
  "SessionStart": [
    {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
    {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(notice) + `}]}
  ]
}}`

	files := settled(before)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := hookResult(t, report)
	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	want := "codefall's hooks are already in .claude/settings.json"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if got := string(files.files[claudeFull]); got != before {
		t.Errorf(".claude/settings.json = %q, want it untouched", got)
	}
}

// The same command under another matcher is not the same entry: the harness would run it for other
// tool calls, and this event would stay unguarded. It joins rather than counting as present. The
// exception is codefall's own match-all entry, which a narrower one already covers.
func TestHookMergeAddsTheSameCommandUnderADifferentMatcher(t *testing.T) {
	definitions := maps.Clone(hookDefinitions)
	definitions["hooks/claude/hooks.json"] = []byte(`{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}]
}}`)

	files := settled(`{"hooks": {"PreToolUse": [` +
		`{"matcher": "Edit", "hooks": [{"type": "command", "command": "guard"}]}]}}`)

	report, err := NewInitialize(files, toolsInstalled(), sourceOf(definitions)).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Fatalf("outcome = %v, want DONE", got)
	}

	entries := eventEntries(t, files, "PreToolUse")
	if len(entries) != 2 || matcherOf(entries[0]) != "Edit" || matcherOf(entries[1]) != "Bash" {
		t.Errorf("PreToolUse = %v, want the Edit entry kept and the Bash one added", entries)
	}

	for _, entry := range entries {
		if got := commandsOf(entry); !slices.Equal(got, []string{"guard"}) {
			t.Errorf("entry commands = %q, want both running the guard", got)
		}
	}
}

// A section the project nulled out says nothing: it merges like an absent key rather than stopping
// the run the way a mistyped one does.
func TestHookMergeTreatsANulledSectionAsAbsent(t *testing.T) {
	files := settled(`{"hooks": null}`)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	got := string(files.files[claudeFull])
	if !strings.Contains(got, "bd prime --hook-json") || !strings.Contains(got, `"PreToolUse"`) {
		t.Errorf(".claude/settings.json = %s, want both events merged", got)
	}
}

// Antigravity keys its hooks by name at the top level, and the merge grows the same arrays under
// them. A flag the project set on codefall's own hook — disabled it, say — is a scalar, and scalars
// the destination says are kept.
func TestHookMergeKeepsTheProjectsAntigravityFlags(t *testing.T) {
	files := settled("")
	files.files[filepath.Join(workingDir, ".agents/hooks.json")] = []byte(
		`{"codefall-merge-guard": {"enabled": false}}`)

	request := beadsRequestHarness(harness.Agy)
	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	got := string(files.files[filepath.Join(workingDir, ".agents/hooks.json")])
	if !strings.Contains(got, `"enabled": false`) {
		t.Errorf(".agents/hooks.json = %s, want the project's enabled flag kept", got)
	}
	if !strings.Contains(got, `"PreToolUse"`) {
		t.Errorf(".agents/hooks.json = %s, want the guard merged", got)
	}
}

// The OpenCode plugin is copied, and a rerun with nothing new is a skip, not a rewrite.
func TestHookCopySkipsAnInstallationItAlreadyHas(t *testing.T) {
	files := settled("")
	request := beadsRequestHarness(harness.OpenCode)

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil); err != nil {
		t.Fatalf("first run: %v", err)
	}

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	result := hookResult(t, report)
	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	want := "codefall's plugin at .opencode/plugins/codefall.js is already installed"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// An installed plugin init cannot read stops the step with the read error, not a copy over it.
func TestHookCopyReportsAFileItCannotRead(t *testing.T) {
	files := settled("")
	path := filepath.Join(workingDir, ".opencode/plugins/codefall.js")
	files.errs[path] = errors.New("permission denied")

	_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).hook(t.Context(), beadsRequestHarness(harness.OpenCode))
	if err == nil || !strings.Contains(err.Error(), "read .opencode/plugins/codefall.js") {
		t.Errorf("hook error = %v, want it to say the file could not be read", err)
	}
}

// A file the step cannot make sense of stops the run rather than being written over: it is somebody
// else's file, and init is not the thing that should decide what it meant. Both halves are checked,
// because the merge builds the merged document in place before a later key can fail it — so the
// error alone would not say the file survived.
//
// The step is called directly rather than through a run, so a failure here is the hook step's own
// and not a step before it stopping first.
func TestHookMergeReportsAFileItCannotWorkWith(t *testing.T) {
	for _, tc := range []struct {
		name     string
		harness  string
		have     map[string][]byte
		settings string
		want     string
	}{
		{
			name:     "the file is not JSON",
			harness:  harness.Claude,
			settings: "{ not json",
			want:     "decode .claude/settings.json",
		},
		{
			name:     "the file is a JSON array",
			harness:  harness.Claude,
			settings: `[{"hooks": {}}]`,
			want:     "decode .claude/settings.json",
		},
		{
			name:     "the file is a JSON string",
			harness:  harness.Claude,
			settings: `"hooks"`,
			want:     "decode .claude/settings.json",
		},
		{
			name:     "hooks is not an object",
			harness:  harness.Claude,
			settings: `{"hooks": ["SessionStart"]}`,
			want:     ".claude/settings.json: hooks is not an object",
		},
		{
			name:     "the event is not an array",
			harness:  harness.Claude,
			settings: `{"hooks": {"SessionStart": {"matcher": ""}}}`,
			want:     ".claude/settings.json: hooks.SessionStart is not an array",
		},
		{
			name:    "an event is not an array under agy's hook name",
			harness: harness.Agy,
			have:    map[string][]byte{filepath.Join(workingDir, ".agents/hooks.json"): []byte(`{"codefall-merge-guard": {"PreToolUse": {"matcher": ""}}}`)},
			want:    ".agents/hooks.json: codefall-merge-guard.PreToolUse is not an array",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled(tc.settings)
			for path, body := range tc.have {
				files.files[path] = body
			}

			before := maps.Clone(files.files)

			_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).hook(t.Context(), beadsRequestHarness(tc.harness))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("hook error = %v, want it to mention %q", err, tc.want)
			}

			for path, body := range files.files {
				if was, had := before[path]; !had || !bytes.Equal(was, body) {
					t.Errorf("%s = %q, want it left as it was (%q)", path, body, was)
				}
			}
		})
	}
}

// A file that cannot be read at all is the step's error too, named so the reader knows which file.
func TestHookMergeReportsAFileItCannotRead(t *testing.T) {
	files := settled("{}")
	files.errs[claudeFull] = errors.New("permission denied")

	_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).hook(t.Context(), beadsRequest())
	if err == nil || !strings.Contains(err.Error(), "read .claude/settings.json") {
		t.Errorf("hook error = %v, want it to name .claude/settings.json the way the report does", err)
	}
}

// The file is somebody else's, so what it says comes back byte for byte. json.Marshal would escape
// <, > and & into \u sequences and quietly rewrite a permission rule; the step's encoder is told
// not to.
func TestHookMergeDoesNotEscapeWhatTheFileAlreadySays(t *testing.T) {
	const rule = "Bash(test a && b < c > d)"

	files := settled(`{"permissions": {"allow": ["` + rule + `"]}}`)

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := string(files.files[claudeFull])
	if !strings.Contains(got, rule) {
		t.Errorf(".claude/settings.json =\n%s\nwant %q in it, unescaped", got, rule)
	}

	if strings.Contains(got, `\u00`) {
		t.Errorf(".claude/settings.json =\n%s\nwant no \\u escapes in it", got)
	}
}

// A harness that takes codefall only as skills has no hook definition, and the step says so rather
// than writing somewhere it should not.
func TestHookSkipsAHarnessWithNoDefinition(t *testing.T) {
	request := beadsRequestHarness(harness.Muse)

	report, err := NewInitialize(settled(""), toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := hookResult(t, report)
	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	want := "codefall has no hooks for muse"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// The mechanism table guards the step the same way the extension step is guarded.
func TestHookRefusesAHarnessItDoesNotKnow(t *testing.T) {
	request := beadsRequestHarness("aider")

	// The extension step refuses this harness first, so the hook step is asked on its own.
	_, err := NewInitialize(settled(""), toolsInstalled(), newFakeExtensionSource()).hook(t.Context(), request)
	if err == nil || !strings.Contains(err.Error(), `harness "aider" has no extension mechanism`) {
		t.Errorf("hook error = %v, want it to say the harness has no extension mechanism", err)
	}
}

// claudeDefinition is the Claude definition with one guard, one prime, and the session-start
// notice, as the embedded tree ships them. Tests that care what the merge does with a particular
// destination seed their own source from it.
func claudeDefinition(t *testing.T, guard string) map[string][]byte {
	t.Helper()

	definitions := maps.Clone(hookDefinitions)
	definitions["hooks/claude/hooks.json"] = []byte(`{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + quoted(guard) + `}]}],
  "SessionStart": [
    {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
    {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(notice) + `}]}
  ]
}}`)

	return definitions
}

// sourceOf is an extension source serving one set of definitions in place of the usual ones, with
// the documents under agents/ as they always are: the run reads those after the hook step, and a
// test about hooks has nothing to say about them.
func sourceOf(definitions map[string][]byte) *fakeExtensionSource {
	source := newFakeExtensionSource()
	maps.Copy(source.data, definitions)

	return source
}

// eventEntries is what one hook event holds in a written destination.
func eventEntries(t *testing.T, files *fakeFileSystem, event string) []any {
	t.Helper()

	var written map[string]any
	if err := json.Unmarshal(files.files[claudeFull], &written); err != nil {
		t.Fatalf("decode %s: %v", claudeFull, err)
	}

	hooks, ok := written["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("%s holds no hooks object: %v", claudeFull, written)
	}

	entries, _ := hooks[event].([]any)

	return entries
}

// A shipped command that changes — a renamed script, a new flag, a project that moved below the
// repository root, the install layout moving the script to .codefall/ — used to read as an unrelated
// registration, so the upgrade appended the new one and left the old one running beside it with
// nothing able to remove it. codefall's own entry is the one naming a codefall- script under the
// same matcher, and the upgrade replaces it. What the destination holds here is what an install
// before the layout registered, which is the position every existing project is in.
func TestHookMergeReplacesItsOwnEntryWhenTheCommandChanges(t *testing.T) {
	const guard = `"$(git rev-parse --show-toplevel)/apps/web/.codefall/hooks/shared/codefall-block-merge-to-main.sh"`

	files := settled(`{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "\"$(git rev-parse --show-toplevel)/.claude/hooks/shared/codefall-block-merge-to-main.sh\""}]}]
}}`)

	report, err := NewInitialize(files, toolsInstalled(), sourceOf(claudeDefinition(t, guard))).
		Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result := hookResult(t, report); result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	entries := eventEntries(t, files, "PreToolUse")
	if len(entries) != 1 {
		t.Fatalf("PreToolUse = %v, want the stale registration replaced rather than joined", entries)
	}

	if got := commandsOf(entries[0]); !slices.Equal(got, []string{guard}) {
		t.Errorf("commands = %q, want %q", got, guard)
	}
}

// An entry whose commands are only partly registered used to be appended whole, so the command the
// destination already ran ran twice. Only what is new joins.
func TestHookMergeAppendsOnlyTheCommandsThatAreNew(t *testing.T) {
	definitions := maps.Clone(hookDefinitions)
	definitions["hooks/claude/hooks.json"] = []byte(`{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [
    {"type": "command", "command": "guard"},
    {"type": "command", "command": "audit"}
  ]}]
}}`)

	files := settled(`{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}]
}}`)

	report, err := NewInitialize(files, toolsInstalled(), sourceOf(definitions)).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result := hookResult(t, report); result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	var commands []string
	for _, entry := range eventEntries(t, files, "PreToolUse") {
		commands = append(commands, commandsOf(entry)...)
	}

	slices.Sort(commands)
	if want := []string{"audit", "guard"}; !slices.Equal(commands, want) {
		t.Errorf("commands = %q, want %q — guard runs once and audit joins", commands, want)
	}
}

// A project that primes Beads under a narrower matcher has said what it wants. codefall's entry
// matches everything, so the project's narrowing already covers it: adding it would prime on every
// session source and twice on the one the project chose. The notice beside it is codefall's own
// entry and is already registered here, so the whole definition is a skip.
func TestHookMergeLeavesANarrowedMatcherAlone(t *testing.T) {
	before := `{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}],
  "SessionStart": [
    {"matcher": "startup", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
    {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(notice) + `}]}
  ]
}}`

	files := settled(before)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if result := hookResult(t, report); result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	if got := string(files.files[claudeFull]); got != before {
		t.Errorf(".claude/settings.json = %q, want it untouched", got)
	}
}

// The definition registers two commands on SessionStart: a prime that is nobody's script, and the
// notice, which is codefall's. A project installed before the notice existed runs the prime already,
// so only the notice joins — and it joins as its own entry rather than being folded into the one
// the destination already has.
func TestHookMergeAddsTheNoticeBesideAPrimeItAlreadyRuns(t *testing.T) {
	const before = `{"hooks": {
  "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "guard"}]}],
  "SessionStart": [{"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]}]
}}`

	files := settled(before)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	var commands []string
	for _, entry := range eventEntries(t, files, "SessionStart") {
		commands = append(commands, commandsOf(entry)...)
	}

	if want := []string{"bd prime --hook-json", notice}; !slices.Equal(commands, want) {
		t.Errorf("SessionStart commands = %q, want %q — the prime runs once and the notice joins",
			commands, want)
	}
}

// The notice is codefall's own entry, known by the script its command names, so a command that
// changes — the flag, the path a subdirectory install writes in — replaces the registration rather
// than leaving the old one running beside the new one. Everything else the event holds is the
// project's: its own entry stays where it is, and the prime is not registered twice.
func TestHookMergeReplacesTheNoticeWhenItsCommandChanges(t *testing.T) {
	const stale = `"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-session-notice.sh"`

	files := settled(`{"hooks": {
  "SessionStart": [
    {"matcher": "", "hooks": [{"type": "command", "command": "echo hello"}]},
    {"matcher": "", "hooks": [{"type": "command", "command": ` + quoted(stale) + `}]},
    {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]}
  ]
}}`)

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := hookResult(t, report).Outcome; got != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", got)
	}

	var commands []string
	for _, entry := range eventEntries(t, files, "SessionStart") {
		commands = append(commands, commandsOf(entry)...)
	}

	want := []string{"echo hello", notice, "bd prime --hook-json"}
	if !slices.Equal(commands, want) {
		t.Errorf("SessionStart commands = %q, want %q — the stale notice replaced in place, "+
			"the project's entry and the prime left as they are", commands, want)
	}
}

// The mirror of the object and array cases: a file shaped against the harness is reported, not
// quietly left as it is. Which side holds the scalar does not change whose file it is.
func TestHookMergeReportsAValueWhereTheDefinitionHasAScalar(t *testing.T) {
	definitions := maps.Clone(hookDefinitions)
	definitions["hooks/claude/hooks.json"] = []byte(`{"description": "codefall"}`)

	files := settled(`{"description": {"written": "by hand"}}`)

	_, err := NewInitialize(files, toolsInstalled(), sourceOf(definitions)).Run(t.Context(), beadsRequest(), nil)
	if err == nil || !strings.Contains(err.Error(), ".claude/settings.json: description is not a scalar") {
		t.Fatalf("Run error = %v, want the file reported rather than left as it is", err)
	}
}
