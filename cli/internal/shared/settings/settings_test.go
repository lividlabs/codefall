package settings

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// complete returns the settings from the schema's example, which every case here mutates.
func complete() Document {
	return Document{
		"$schema":      SchemaID,
		"version":      1.0,
		"tracker":      "github",
		FieldHarnesses: []any{"claude"},
		"github": map[string]any{
			"issuesRepo":    "lividlabs/codefall-cli",
			"issuesProject": 3.0,
		},
	}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  Document
		want []string
	}{
		{
			name: "complete",
			doc:  complete(),
		},
		{
			name: "complete without the optional fields",
			doc: Document{
				"version":      1.0,
				"tracker":      "github",
				FieldHarnesses: []any{"claude"},
				"github":       map[string]any{"issuesRepo": "lividlabs/codefall-cli"},
			},
		},
		{
			name: "every harness codefall can set up, at once",
			doc: with(complete(), FieldHarnesses,
				[]any{"agy", "claude", "codex", "muse", "opencode"}),
		},
		{
			// A project whose checked-in settings predate the rename is still valid: doctor warns
			// about the old spelling, and codefall init rewrites it.
			name: "a harness under the spelling it had before it was named for its binary",
			doc:  with(complete(), FieldHarnesses, []any{"claude-code", "antigravity"}),
		},
		{
			name: "unknown top-level keys are ignored",
			doc: Document{
				"version":      1.0,
				"tracker":      "github",
				FieldHarnesses: []any{"claude"},
				"github":       map[string]any{"issuesRepo": "a/b"},
				"nonsense":     "ignored",
			},
		},
		{
			name: "a block for a tracker nobody knows about is ignored",
			doc: Document{
				"version":      1.0,
				"tracker":      "github",
				FieldHarnesses: []any{"claude"},
				"github":       map[string]any{"issuesRepo": "a/b"},
				"gitlab":       map[string]any{"project": "x"},
			},
		},
		{
			name: "empty document",
			doc:  Document{},
			want: []string{"version: missing", "tracker: missing", "harnesses: missing"},
		},
		{
			name: "harnesses missing",
			doc:  without(complete(), FieldHarnesses),
			want: []string{"harnesses: missing"},
		},
		{
			name: "harnesses is not an array",
			doc:  with(complete(), FieldHarnesses, "claude"),
			want: []string{"harnesses: must be an array of harness names"},
		},
		{
			name: "harnesses holds something that is not a name",
			doc:  with(complete(), FieldHarnesses, []any{1.0}),
			want: []string{"harnesses: must be an array of harness names"},
		},
		{
			// An empty list is refused rather than read as "none": a project codefall set up for no
			// harness at all has nowhere to install.
			name: "harnesses is empty",
			doc:  with(complete(), FieldHarnesses, []any{}),
			want: []string{"harnesses: must name at least one harness"},
		},
		{
			name: "a harness codefall cannot set up",
			doc:  with(complete(), FieldHarnesses, []any{"claude", "cursor"}),
			want: []string{`harnesses: unknown value "cursor" ` +
				`(expected "agy", "claude", "codex", "muse", "opencode")`},
		},
		{
			name: "version missing",
			doc:  without(complete(), "version"),
			want: []string{"version: missing"},
		},
		{
			name: "version empty string counts as missing",
			doc:  with(complete(), "version", ""),
			want: []string{"version: missing"},
		},
		{
			name: "version is another number",
			doc:  with(complete(), "version", 2.0),
			want: []string{"version: must be 1"},
		},
		{
			name: "version is a string",
			doc:  with(complete(), "version", "1"),
			want: []string{"version: must be 1"},
		},
		{
			name: "version is fractional",
			doc:  with(complete(), "version", 1.5),
			want: []string{"version: must be 1"},
		},
		{
			name: "tracker missing suppresses the block problems",
			doc:  without(complete(), "tracker"),
			want: []string{"tracker: missing"},
		},
		{
			name: "tracker is not a string",
			doc:  with(complete(), "tracker", 42.0),
			want: []string{"tracker: must be a string"},
		},
		{
			name: "unknown tracker suppresses the block problems",
			doc:  without(with(complete(), "tracker", "gitlab"), "github"),
			want: []string{`tracker: unknown value "gitlab" (expected "beads", "github")`},
		},
		{
			name: "beads tracker with an empty beads block is complete",
			doc: Document{
				"version":      1.0,
				"tracker":      "beads",
				FieldHarnesses: []any{"codex"},
				"beads":        map[string]any{},
			},
		},
		{
			name: "beads block missing",
			doc: Document{
				"version":      1.0,
				"tracker":      "beads",
				FieldHarnesses: []any{"codex"},
			},
			want: []string{`beads: missing (required when tracker is "beads")`},
		},
		{
			name: "github block present while tracker is beads",
			doc: Document{
				"version":      1.0,
				"tracker":      "beads",
				FieldHarnesses: []any{"codex"},
				"beads":        map[string]any{},
				"github":       map[string]any{"issuesRepo": "a/b"},
			},
			want: []string{`github: present but tracker is "beads" — remove it`},
		},
		{
			name: "block missing",
			doc:  without(complete(), "github"),
			want: []string{`github: missing (required when tracker is "github")`},
		},
		{
			name: "block is not an object",
			doc:  with(complete(), "github", "lividlabs/codefall-cli"),
			want: []string{"github: must be an object"},
		},
		{
			name: "repo missing",
			doc:  with(complete(), "github", map[string]any{"issuesProject": 3.0}),
			want: []string{"github.issuesRepo: missing"},
		},
		{
			name: "repo empty counts as missing",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": ""}),
			want: []string{"github.issuesRepo: missing"},
		},
		{
			name: "repo without a slash",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": "no-slash"}),
			want: []string{"github.issuesRepo: must match owner/name"},
		},
		{
			name: "repo with two slashes",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": "a/b/c"}),
			want: []string{"github.issuesRepo: must match owner/name"},
		},
		{
			name: "repo is not a string",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": 3.0}),
			want: []string{"github.issuesRepo: must be a string"},
		},
		{
			name: "project is zero",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": "a/b", "issuesProject": 0.0}),
			want: []string{"github.issuesProject: must be a positive integer"},
		},
		{
			name: "project is fractional",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": "a/b", "issuesProject": 1.5}),
			want: []string{"github.issuesProject: must be a positive integer"},
		},
		{
			name: "project is a string",
			doc:  with(complete(), "github", map[string]any{"issuesRepo": "a/b", "issuesProject": "3"}),
			want: []string{"github.issuesProject: must be a positive integer"},
		},
		{
			name: "$schema is not a string",
			doc:  with(complete(), "$schema", 1.0),
			want: []string{"$schema: must be a string"},
		},
		{
			name: "a complete local block",
			doc:  with(complete(), BlockLocal, map[string]any{"start": "make up", "update": "make sync"}),
		},
		{
			name: "local block is not an object",
			doc:  with(complete(), BlockLocal, "make sync"),
			want: []string{"local: must be an object"},
		},
		{
			name: "review block is not an object",
			doc:  with(complete(), BlockReview, true),
			want: []string{"review: must be an object"},
		},
		{
			// Both commands or neither: update assumes start has run (ADR-005).
			name: "local block missing update",
			doc:  with(complete(), BlockLocal, map[string]any{"start": "make up"}),
			want: []string{"local.update: missing"},
		},
		{
			name: "local block missing start",
			doc:  with(complete(), BlockLocal, map[string]any{"update": "make sync"}),
			want: []string{"local.start: missing"},
		},
		{
			name: "local command empty counts as missing",
			doc:  with(complete(), BlockLocal, map[string]any{"start": "", "update": "make sync"}),
			want: []string{"local.start: missing"},
		},
		{
			name: "local command is not a string",
			doc:  with(complete(), BlockLocal, map[string]any{"start": "make up", "update": []any{"make", "sync"}}),
			want: []string{"local.update: must be a string"},
		},
		{
			name: "a complete test block",
			doc: with(complete(), BlockTest,
				map[string]any{"dir": "testing", "runners": []any{"playwright"}}),
		},
		{
			// A project declares its root before it is equipped, so the runners are optional and an
			// empty list is what "declared, not equipped yet" looks like (ADR-007).
			name: "a test block with no runners",
			doc:  with(complete(), BlockTest, map[string]any{"dir": "testing"}),
		},
		{
			name: "a test block with an empty runner list",
			doc:  with(complete(), BlockTest, map[string]any{"dir": "e2e", "runners": []any{}}),
		},
		{
			name: "test block is not an object",
			doc:  with(complete(), BlockTest, "testing"),
			want: []string{"test: must be an object"},
		},
		{
			name: "test block missing the directory",
			doc:  with(complete(), BlockTest, map[string]any{"runners": []any{"go-test"}}),
			want: []string{"test.dir: missing"},
		},
		{
			name: "test directory is absolute",
			doc:  with(complete(), BlockTest, map[string]any{"dir": "/srv/testing"}),
			want: []string{`test.dir: must be a relative path inside the project, such as "testing"`},
		},
		{
			name: "test directory climbs out of the project",
			doc:  with(complete(), BlockTest, map[string]any{"dir": "../testing"}),
			want: []string{`test.dir: must be a relative path inside the project, such as "testing"`},
		},
		{
			name: "test directory is not a string",
			doc:  with(complete(), BlockTest, map[string]any{"dir": 1.0}),
			want: []string{"test.dir: must be a string"},
		},
		{
			name: "runners is not an array",
			doc:  with(complete(), BlockTest, map[string]any{"dir": "testing", "runners": "playwright"}),
			want: []string{"test.runners: must be an array of runner names"},
		},
		{
			name: "a runner codefall does not know",
			doc: with(complete(), BlockTest,
				map[string]any{"dir": "testing", "runners": []any{"cypress"}}),
			want: []string{`test.runners: unknown value "cypress" (expected "go-test", "playwright")`},
		},
		{
			name: "the same runner twice",
			doc: with(complete(), BlockTest,
				map[string]any{"dir": "testing", "runners": []any{"playwright", "playwright"}}),
			want: []string{`test.runners: names "playwright" twice`},
		},
		{
			name: "every problem is reported at once, in definition order",
			doc: Document{
				"$schema": 1.0,
				"version": 2.0,
				"tracker": "github",
				"github":  map[string]any{"issuesRepo": "no-slash", "issuesProject": 0.0},
				"local":   map[string]any{"start": "make up"},
			},
			want: []string{
				"$schema: must be a string",
				"version: must be 1",
				"harnesses: missing",
				"local.update: missing",
				"github.issuesRepo: must match owner/name",
				"github.issuesProject: must be a positive integer",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Validate(tc.doc); !slices.Equal(got, tc.want) {
				t.Errorf("Validate() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A block for a tracker that is not selected is a problem once that tracker is known — beads and
// github already prove this pairwise above. This test adds a third, synthetic tracker for its
// length to show the rule keeps generalizing, which is also the proof that adding a row is all a
// further tracker needs.
func TestValidateRejectsAnotherKnownTrackersBlock(t *testing.T) {
	trackerFields["fake"] = []fieldSpec{{"token", true, isString}}
	t.Cleanup(func() { delete(trackerFields, "fake") })

	doc := with(complete(), "fake", map[string]any{"token": "x"})

	want := []string{`fake: present but tracker is "github" — remove it`}
	if got := Validate(doc); !slices.Equal(got, want) {
		t.Errorf("Validate() = %q, want %q", got, want)
	}
}

func TestTrackers(t *testing.T) {
	if got, want := Trackers(), []string{TrackerBeads, TrackerGitHub}; !slices.Equal(got, want) {
		t.Errorf("Trackers() = %q, want %q", got, want)
	}

	// The caller gets a copy: mutating it must not change what the next caller sees. Order is not
	// cosmetic — the schema test holds the schema's tracker enum equal to this list.
	Trackers()[0] = "mutated"

	if got := Trackers()[0]; got != TrackerBeads {
		t.Errorf("Trackers()[0] after a caller mutated its copy = %q, want %q", got, TrackerBeads)
	}
}

func TestParseTracker(t *testing.T) {
	for _, tracker := range Trackers() {
		if got, err := ParseTracker(tracker); err != nil || got != tracker {
			t.Errorf("ParseTracker(%q) = %q, %v, want %q, nil", tracker, got, err, tracker)
		}
	}

	_, err := ParseTracker("jira")
	if err == nil {
		t.Fatal(`ParseTracker("jira") = nil error, want an error`)
	}

	// The message lists what would have worked, because that is what the reader needs next.
	for _, want := range append([]string{"jira"}, Trackers()...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ParseTracker(%q) error = %q, want it to mention %q", "jira", err, want)
		}
	}
}

func TestValidateRepo(t *testing.T) {
	for _, repo := range []string{"owner/name", "lividlabs/codefall-cli", "a_b.c-d/e.f_g-h"} {
		if err := ValidateRepo(repo); err != nil {
			t.Errorf("ValidateRepo(%q) = %v, want nil", repo, err)
		}
	}

	for _, repo := range []string{"", "name", "owner/name/extra", "owner /name", "owner/na me"} {
		if err := ValidateRepo(repo); err == nil {
			t.Errorf("ValidateRepo(%q) = nil, want an error", repo)
		}
	}
}

func TestValidateProject(t *testing.T) {
	for _, number := range []int{1, 3, 4096} {
		if err := ValidateProject(number); err != nil {
			t.Errorf("ValidateProject(%d) = %v, want nil", number, err)
		}
	}

	for _, number := range []int{0, -1} {
		if err := ValidateProject(number); err == nil {
			t.Errorf("ValidateProject(%d) = nil, want an error", number)
		}
	}
}

func TestRequiredFields(t *testing.T) {
	if got, want := RequiredFields(), []string{"version", "tracker", "harnesses"}; !slices.Equal(got, want) {
		t.Errorf("RequiredFields() = %q, want %q", got, want)
	}

	if got, want := RequiredTrackerFields(TrackerBeads), []string{}; !slices.Equal(got, want) {
		t.Errorf("RequiredTrackerFields(%q) = %q, want %q", TrackerBeads, got, want)
	}

	if got, want := RequiredTrackerFields(TrackerGitHub), []string{"issuesRepo"}; !slices.Equal(got, want) {
		t.Errorf("RequiredTrackerFields(%q) = %q, want %q", TrackerGitHub, got, want)
	}

	if got := RequiredTrackerFields("nonesuch"); len(got) != 0 {
		t.Errorf("RequiredTrackerFields(%q) = %q, want none", "nonesuch", got)
	}

	if got, want := RequiredLocalFields(), []string{"start", "update"}; !slices.Equal(got, want) {
		t.Errorf("RequiredLocalFields() = %q, want %q", got, want)
	}

	if got, want := RequiredTestFields(), []string{"dir"}; !slices.Equal(got, want) {
		t.Errorf("RequiredTestFields() = %q, want %q", got, want)
	}
}

func TestTestRunners(t *testing.T) {
	if got, want := TestRunners(), []string{RunnerGoTest, RunnerPlaywright}; !slices.Equal(got, want) {
		t.Errorf("TestRunners() = %q, want %q", got, want)
	}

	// Order is not cosmetic: the schema test holds the schema's runner enum equal to this list.
	TestRunners()[0] = "mutated"

	if got := TestRunners()[0]; got != RunnerGoTest {
		t.Errorf("TestRunners()[0] after a caller mutated its copy = %q, want %q", got, RunnerGoTest)
	}
}

func TestValidateTestDir(t *testing.T) {
	for _, dir := range []string{"testing", "e2e", "test/e2e", "packages/web/testing", "tests_2"} {
		if err := ValidateTestDir(dir); err != nil {
			t.Errorf("ValidateTestDir(%q) = %v, want nil", dir, err)
		}
	}

	// Absolute, climbing out, hidden, a Windows path, and a name with a space in it: each of them
	// would mean something different on another machine, or nothing at all.
	for _, dir := range []string{"", "/testing", "../testing", "testing/../..", ".testing", "./testing",
		`C:\testing`, "testing dir"} {
		if err := ValidateTestDir(dir); err == nil {
			t.Errorf("ValidateTestDir(%q) = nil, want an error", dir)
		}
	}
}

func TestTestArtifacts(t *testing.T) {
	for _, tc := range []struct{ root, want string }{
		{root: "testing", want: "testing/.artifacts/"},
		{root: "packages/web/e2e", want: "packages/web/e2e/.artifacts/"},
		{root: "testing/", want: "testing/.artifacts/"},
	} {
		if got := TestArtifacts(tc.root); got != tc.want {
			t.Errorf("TestArtifacts(%q) = %q, want %q", tc.root, got, tc.want)
		}
	}
}

func TestTestDeclaration(t *testing.T) {
	declared := with(complete(), BlockTest,
		map[string]any{"dir": "e2e", "runners": []any{"playwright", "go-test"}})

	got, ok := TestDeclaration(declared).Get()
	if !ok || got.Dir != "e2e" || !slices.Equal(got.Runners, []string{"playwright", "go-test"}) {
		t.Errorf("TestDeclaration() = %+v, %v, want the root and both runners, true", got, ok)
	}

	// A root with no runners is a declaration: the project has said where its cases live and has not
	// been equipped yet.
	got, ok = TestDeclaration(with(complete(), BlockTest, map[string]any{"dir": "testing"})).Get()
	if !ok || got.Dir != "testing" || len(got.Runners) != 0 {
		t.Errorf("TestDeclaration() of a root with no runners = %+v, %v, want the root and no runners, true", got, ok)
	}

	// Everything Validate would reject is None here, by the same rule the local block follows.
	for name, doc := range map[string]Document{
		"absent":                  complete(),
		"not an object":           with(complete(), BlockTest, "testing"),
		"no directory":            with(complete(), BlockTest, map[string]any{"runners": []any{"go-test"}}),
		"an absolute root":        with(complete(), BlockTest, map[string]any{"dir": "/srv/testing"}),
		"a root that climbs":      with(complete(), BlockTest, map[string]any{"dir": "../testing"}),
		"a root that is not text": with(complete(), BlockTest, map[string]any{"dir": 1.0}),
		"an unknown runner": with(complete(), BlockTest,
			map[string]any{"dir": "testing", "runners": []any{"cypress"}}),
		"null block": with(complete(), BlockTest, nil),
	} {
		if got := TestDeclaration(doc); got.IsPresent() {
			t.Errorf("TestDeclaration() with the block %s = %v, want None", name, got)
		}
	}
}

// Every entry in either ignore file is read by one rule: a whole line, surrounding space ignored,
// and a comment is not an entry — neither ripgrep nor git reads it as one.
func TestNamesEntry(t *testing.T) {
	for _, tc := range []struct {
		name  string
		body  string
		names bool
	}{
		{name: "written plainly", body: "vendor/\n%s\n", names: true},
		{name: "indented", body: "  %s  \n", names: true},
		{name: "without a trailing newline", body: "%s", names: true},
		{name: "commented out", body: "# %s\n", names: false},
		{name: "as a prefix of a longer entry", body: "%s.bak\n", names: false},
		{name: "absent", body: "vendor/\nnode_modules/\n", names: false},
		{name: "empty", body: "", names: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, entry := range []string{IgnoreEntry, IgnoreEntryTests, RefreshStamp, TestArtifacts("testing")} {
				body := fmt.Sprintf(tc.body, entry)

				if got := NamesEntry(body, entry); got != tc.names {
					t.Errorf("NamesEntry(%q, %q) = %v, want %v", body, entry, got, tc.names)
				}
			}
		})
	}
}

func TestHarnesses(t *testing.T) {
	if got, want := Harnesses(complete()), []string{"claude"}; !slices.Equal(got, want) {
		t.Errorf("Harnesses() = %q, want %q", got, want)
	}

	// Whatever is not a name is left out rather than reported: Validate already refused it.
	doc := with(complete(), FieldHarnesses, []any{"codex", 1.0, "opencode"})
	if got, want := Harnesses(doc), []string{"codex", "opencode"}; !slices.Equal(got, want) {
		t.Errorf("Harnesses() = %q, want %q", got, want)
	}

	if got := Harnesses(without(complete(), FieldHarnesses)); len(got) != 0 {
		t.Errorf("Harnesses() without the field = %q, want none", got)
	}
}

func TestLocalCommands(t *testing.T) {
	declared := with(complete(), BlockLocal, map[string]any{"start": "make up", "update": "make sync"})

	got, ok := LocalCommands(declared).Get()
	if want := (Local{Start: "make up", Update: "make sync"}); !ok || got != want {
		t.Errorf("LocalCommands() = %v, %v, want %+v, true", got, ok, want)
	}

	// A block Validate would reject is not a declaration; the caller has already been told why.
	for name, doc := range map[string]Document{
		"absent":          complete(),
		"not an object":   with(complete(), BlockLocal, "make sync"),
		"missing update":  with(complete(), BlockLocal, map[string]any{"start": "make up"}),
		"empty start":     with(complete(), BlockLocal, map[string]any{"start": "", "update": "make sync"}),
		"update not text": with(complete(), BlockLocal, map[string]any{"start": "make up", "update": 1.0}),
		"null block":      with(complete(), BlockLocal, nil),
	} {
		if got := LocalCommands(doc); got.IsPresent() {
			t.Errorf("LocalCommands() with the block %s = %v, want None", name, got)
		}
	}
}

func with(doc Document, key string, value any) Document {
	doc[key] = value

	return doc
}

func without(doc Document, key string) Document {
	delete(doc, key)

	return doc
}
