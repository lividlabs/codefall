package application

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

var (
	testingAgentsFull = filepath.Join(testingFull, "AGENTS.md")
	testingReadmeFull = filepath.Join(testingFull, "README.md")
	testingClaudeFull = filepath.Join(testingFull, "CLAUDE.md")
)

// testingResult is what the sixth step did in a run that got that far.
func testingResult(t *testing.T, files *fakeFileSystem, request Request) domain.StepResult {
	t.Helper()

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(), request, nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	return results[5]
}

// A project with no testing tree gets the whole one: the directories, the two skeletons, and the
// CLAUDE.md pointer for the harness that reads one.
func TestTestingStepMakesTheTreeAndDeclaresTheRoot(t *testing.T) {
	files := settled("{}")

	result := testingResult(t, files, beadsRequest())

	if result.Step != domain.TestingStep {
		t.Fatalf("the sixth step = %+v, want %+v", result.Step, domain.TestingStep)
	}

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "declared testing/ in .codefall/settings.json and created testing/AGENTS.md, " +
		"testing/README.md and testing/CLAUDE.md"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	for _, made := range []string{testingFull, testCasesFull} {
		if !strings.Contains(strings.Join(files.made, " "), made) {
			t.Errorf("created %q, want it to include %q", files.made, made)
		}
	}

	if got, want := string(files.files[testingAgentsFull]), string(agentsDocuments["agents/testing/AGENTS.md"]); got != want {
		t.Errorf("testing/AGENTS.md =\n%s\nwant the skeleton from the tree:\n%s", got, want)
	}

	if got, want := string(files.files[testingReadmeFull]), string(agentsDocuments["agents/testing/README.md"]); got != want {
		t.Errorf("testing/README.md =\n%s\nwant the skeleton from the tree:\n%s", got, want)
	}

	if got := string(files.files[testingClaudeFull]); got != claudePointer {
		t.Errorf("testing/CLAUDE.md = %q, want the pointer %q", got, claudePointer)
	}
}

// The root the request declares is where everything goes, including the block the step writes.
func TestTestingStepWorksAtTheDeclaredRoot(t *testing.T) {
	files := settled("{}")

	request := beadsRequest()
	request.TestDir = "packages/web/e2e"

	result := testingResult(t, files, request)

	want := "declared packages/web/e2e/ in .codefall/settings.json and created " +
		"packages/web/e2e/AGENTS.md, packages/web/e2e/README.md and packages/web/e2e/CLAUDE.md"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if _, written := files.files[filepath.Join(workingDir, "packages", "web", "e2e", "README.md")]; !written {
		t.Errorf("wrote %q, want the tree at the declared root", keysOf(files))
	}

	if got := string(files.files[settingsFull]); !strings.Contains(got, `"dir": "packages/web/e2e"`) {
		t.Errorf("settings.json =\n%s\nwant it to declare the root", got)
	}
}

// The block goes in beside what the file already says, and every other byte comes back as it was:
// most projects get theirs on a rerun over settings a run before this one wrote, and a project's own
// keys and codefall-equip's blocks are in that file too.
func TestTestingStepDeclaresTheRootWithoutDisturbingTheSettings(t *testing.T) {
	const before = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "beads": {},
  "local": {
    "start": "make up",
    "update": "make sync"
  }
}
`

	files := settled("{}")
	files.files[settingsFull] = []byte(before)

	testingResult(t, files, beadsRequest())

	// Every line above the block is the one that was there, the local block's own closing brace
	// included: only the document's last brace moves.
	const want = `{
  "version": 1,
  "tracker": "beads",
  "harnesses": [
    "claude"
  ],
  "beads": {},
  "local": {
    "start": "make up",
    "update": "make sync"
  },
  "test": {
    "dir": "testing"
  }
}
`

	if got := string(files.files[settingsFull]); got != want {
		t.Errorf("settings.json =\n%s\nwant\n%s", got, want)
	}
}

// A declaration that is already there is never moved and never rewritten, whatever this run was
// told: the project's cases are at the path it declared, and the runners beside it are equip's.
func TestTestingStepLeavesADeclarationThatIsAlreadyThere(t *testing.T) {
	const before = `{
  "test": {
    "dir": "e2e",
    "runners": [
      "playwright"
    ]
  }
}
`

	files := settled("{}")
	files.files[settingsFull] = []byte(before)

	request := beadsRequest()
	request.TestDir = "e2e"

	result := testingResult(t, files, request)

	if got := string(files.files[settingsFull]); got != before {
		t.Errorf("settings.json =\n%s\nwant it untouched", got)
	}

	if strings.Contains(result.Detail, "declared") {
		t.Errorf("detail = %q, want it not to claim a declaration it did not write", result.Detail)
	}
}

// Every document is the project's from the moment it exists, so a second run leaves all of them
// alone and has nothing to report.
func TestTestingStepSkipsATreeThatIsAlreadyThere(t *testing.T) {
	files := settled("{}")
	files.files[settingsFull] = []byte(`{"test": {"dir": "testing"}}`)
	files.files[testingAgentsFull] = []byte("# Testing\n\nWhat this project does.\n")
	files.files[testingReadmeFull] = []byte("# Testing\n\nOurs.\n")
	files.files[testingClaudeFull] = []byte("Ours too.\n")

	result := testingResult(t, files, beadsRequest())

	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	if want := "the testing tree at testing/ is current"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if got := string(files.files[testingAgentsFull]); got != "# Testing\n\nWhat this project does.\n" {
		t.Errorf("testing/AGENTS.md =\n%s\nwant it untouched", got)
	}
}

// Only Claude Code reads a CLAUDE.md, so only Claude Code gets one. AGENTS.md is the file every
// harness reads, and it is written either way.
func TestTestingStepWritesTheClaudePointerOnlyForClaudeCode(t *testing.T) {
	files := settled("{}")

	request := beadsRequest()
	request.Harnesses = []string{harness.Codex}

	result := testingResult(t, files, request)

	if strings.Contains(result.Detail, "CLAUDE.md") {
		t.Errorf("detail = %q, want no pointer for a harness that reads none", result.Detail)
	}

	if _, written := files.files[testingClaudeFull]; written {
		t.Errorf("testing/CLAUDE.md = %q, want no file", files.files[testingClaudeFull])
	}
}

// A root that is not a relative path inside the project stops the run rather than making a tree
// somewhere the project did not mean. Presentation refuses it first; this is the use case's own
// invariant, for a caller that is not the command.
func TestTestingStepRefusesARootOutsideTheProject(t *testing.T) {
	request := beadsRequest()
	request.TestDir = "../testing"

	_, err := NewInitialize(settled("{}"), toolsInstalled(), newFakeExtensionSource()).testing(
		t.Context(), request)
	if err == nil || !strings.Contains(err.Error(), "relative path inside the project") {
		t.Errorf("testing error = %v, want it to refuse the root", err)
	}
}

// A skeleton the tree does not hold is a binary shipped wrong: the step stops and names the file,
// before it writes anything at the root.
func TestTestingStepRefusesASkeletonTheTreeDoesNotHold(t *testing.T) {
	files := settled("{}")

	source := newFakeExtensionSource()
	delete(source.data, "agents/testing/README.md")

	_, err := NewInitialize(files, toolsInstalled(), source).Run(t.Context(), beadsRequest(), nil)
	if err == nil || !strings.Contains(err.Error(), "read agents/testing/README.md") {
		t.Errorf("Run error = %v, want it to name the skeleton the tree does not hold", err)
	}

	if _, written := files.files[testingAgentsFull]; written {
		t.Errorf("testing/AGENTS.md = %q, want nothing written when a skeleton is missing",
			files.files[testingAgentsFull])
	}
}

// A file the step cannot use is its error, named so the reader knows which one.
func TestTestingStepReportsWhatItCannotUse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeFileSystem)
		want  string
	}{
		{
			name:  "the directory cannot be made",
			setup: func(f *fakeFileSystem) { f.errs["mkdir "+testCasesFull] = errors.New("read-only file system") },
			want:  "create testing/test-cases/",
		},
		{
			name:  "the settings cannot be read",
			setup: func(f *fakeFileSystem) { f.errs[settingsFull] = errors.New("permission denied") },
			want:  "read .codefall/settings.json",
		},
		{
			name:  "the settings are not JSON",
			setup: func(f *fakeFileSystem) { f.files[settingsFull] = []byte("{,}") },
			want:  "decode .codefall/settings.json",
		},
		{
			name: "the settings cannot be written",
			setup: func(f *fakeFileSystem) {
				f.errs["write "+settingsFull] = errors.New("no space left on device")
			},
			want: "write .codefall/settings.json",
		},
		{
			name:  "a skeleton cannot be read",
			setup: func(f *fakeFileSystem) { f.errs[testingReadmeFull] = errors.New("permission denied") },
			want:  "read testing/README.md",
		},
		{
			name: "a skeleton cannot be written",
			setup: func(f *fakeFileSystem) {
				f.errs["write "+testingAgentsFull] = errors.New("read-only file system")
			},
			want: "write testing/AGENTS.md",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("{}")
			tc.setup(files)

			_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).testing(
				t.Context(), beadsRequest())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("testing error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// The splice puts the block at the end of whatever object the file holds, and leaves everything
// before it byte for byte.
func TestWithTestBlock(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		root string
		want string
	}{
		{
			name: "an empty object",
			body: "{}\n",
			root: settings.DefaultTestDir,
			want: "{\n  \"test\": {\n    \"dir\": \"testing\"\n  }\n}\n",
		},
		{
			name: "an object on one line",
			body: `{"version": 1}`,
			root: "e2e",
			want: "{\"version\": 1,\n  \"test\": {\n    \"dir\": \"e2e\"\n  }\n}\n",
		},
		{
			name: "a file with no trailing newline",
			body: "{\n  \"version\": 1\n}",
			root: "e2e",
			want: "{\n  \"version\": 1,\n  \"test\": {\n    \"dir\": \"e2e\"\n  }\n}\n",
		},
		{
			name: "a root that needs escaping",
			body: "{}",
			root: `test"ing`,
			want: "{\n  \"test\": {\n    \"dir\": \"test\\\"ing\"\n  }\n}\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := withTestBlock(tc.body, tc.root); got != tc.want {
				t.Errorf("withTestBlock(%q, %q) =\n%q\nwant\n%q", tc.body, tc.root, got, tc.want)
			}
		})
	}
}
