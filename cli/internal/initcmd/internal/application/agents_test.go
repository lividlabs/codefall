package application

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

var (
	agentsFull       = filepath.Join(workingDir, "AGENTS.md")
	claudeMemoryFull = filepath.Join(workingDir, "CLAUDE.md")
)

// agentsDocuments is what the fake tree holds under agents/: the four sections, each between its
// markers the way the shipped files are, and the two skeletons the testing step writes. Stand-ins
// for the real words, which the facade test pins against the real tree; what the application
// layer's tests need is that the step reads them from here and splices them where they go.
var agentsDocuments = map[string][]byte{
	"agents/sections/codefall.md": []byte(domain.CodefallSectionBegin + "\n## Codefall\n\nThe chain, " +
		"and `.codefall/shared/workflow.md` for the rest.\n" + domain.CodefallSectionEnd + "\n"),
	"agents/sections/beads.md": []byte(domain.BeadsSectionBegin + "\n## Beads\n\nThe tracker.\n" +
		domain.BeadsSectionEnd + "\n"),
	"agents/sections/local.md": []byte(domain.LocalSectionBegin + "\n## Local environment\n\nRun refresh.\n" +
		domain.LocalSectionEnd + "\n"),
	"agents/sections/testing.md": []byte(domain.TestingSectionBegin + "\n## Testing\n\nCases live under `" +
		domain.TestingRootPlaceholder + "/`.\n" + domain.TestingSectionEnd + "\n"),
	"agents/testing/AGENTS.md": []byte("# Testing\n\n## Runners\n\nNone declared yet.\n"),
	"agents/testing/README.md": []byte("# Test cases\n\n| Case ID | Modalities | Variants | Notes |\n"),
}

// What a file that has been through the step says, without the newline that ends each section: the
// Codefall section, the Beads section, the Local environment section, the Testing section for the
// root the fixture declares, and the four together as a file that had nothing else gains them.
var (
	codefallSection = sectionFixture("agents/sections/codefall.md", settings.DefaultTestDir)
	beadsSection    = sectionFixture("agents/sections/beads.md", settings.DefaultTestDir)
	localSection    = sectionFixture("agents/sections/local.md", settings.DefaultTestDir)
	testingSection  = sectionFixture("agents/sections/testing.md", settings.DefaultTestDir)
	threeSections   = beadsSection + "\n\n" + localSection + "\n\n" + testingSection
	allSections     = codefallSection + "\n\n" + threeSections
)

// sectionFixture is one section as it lands in a file: the fake tree's words with the root filled in
// and without the newline that ends the file.
func sectionFixture(source, root string) string {
	body := strings.ReplaceAll(string(agentsDocuments[source]), domain.TestingRootPlaceholder, root)

	return strings.TrimSuffix(body, "\n")
}

// agentsResult is what the fifth step did in a run that got that far.
func agentsResult(t *testing.T, report domain.Report) domain.StepResult {
	t.Helper()

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	return results[4]
}

// run is the whole init run, in a project whose settings and harness file are already settled, with
// whatever markdown the test wants beside them.
func run(t *testing.T, markdown map[string][]byte) (*fakeFileSystem, domain.StepResult) {
	t.Helper()

	files := settled("{}")
	for path, body := range markdown {
		files.files[path] = body
	}

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	return files, agentsResult(t, report)
}

// A project with no AGENTS.md gets one that is the four sections and nothing else, the Codefall
// section first because it is the frame the other three sit inside, and the CLAUDE.md that points
// at it.
func TestAgentsStepCreatesTheFilesThatAreNotThere(t *testing.T) {
	files, result := run(t, nil)

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "created AGENTS.md with the Codefall, Beads, Local environment and Testing sections and created CLAUDE.md"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if got := string(files.files[agentsFull]); got != allSections+"\n" {
		t.Errorf("AGENTS.md =\n%s\nwant\n%s", got, allSections+"\n")
	}

	if !strings.HasPrefix(string(files.files[agentsFull]), domain.CodefallSectionBegin+"\n") {
		t.Errorf("AGENTS.md opens %q, want the Codefall section first", firstLine(string(files.files[agentsFull])))
	}

	want = "See [AGENTS.md](AGENTS.md) — the rules for this repo live there, and there only.\n"
	if got := string(files.files[claudeMemoryFull]); got != want {
		t.Errorf("CLAUDE.md = %q, want %q", got, want)
	}
}

// A file that already says something keeps every word of it and gains the sections at the end, with
// one blank line in between however the file happened to end.
func TestAgentsStepAppendsToAFileThatAlreadySaysSomething(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before string
	}{
		{name: "the file ends with a newline", before: "# Project\n\nWhat it is.\n"},
		{name: "the file does not", before: "# Project\n\nWhat it is."},
		{name: "the file ends with several newlines", before: "# Project\n\nWhat it is.\n\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, result := run(t, map[string][]byte{agentsFull: []byte(tc.before)})

			if result.Outcome != domain.OutcomeDone {
				t.Errorf("outcome = %v, want DONE", result.Outcome)
			}

			want := "added the Codefall, Beads, Local environment and Testing sections to AGENTS.md and created CLAUDE.md"
			if result.Detail != want {
				t.Errorf("detail = %q, want %q", result.Detail, want)
			}

			want = "# Project\n\nWhat it is.\n\n" + allSections + "\n"
			if got := string(files.files[agentsFull]); got != want {
				t.Errorf("AGENTS.md =\n%q\nwant\n%q", got, want)
			}
		})
	}
}

// A section that is already there is rewritten between its markers, and what the file says on either
// side comes back byte for byte. A section that is not there yet goes at the end, after the
// project's own words rather than beside the section it belongs with, because the step never moves
// what somebody else wrote.
func TestAgentsStepReplacesTheSectionInPlace(t *testing.T) {
	const before = "# Project\n\nWhat it is.\n\n" +
		domain.BeadsSectionBegin + "\n## Beads\n\nSomething older.\n" + domain.BeadsSectionEnd +
		"\n\n## Afterwards\n\nMore of the project's own words.\n"

	files, result := run(t, map[string][]byte{agentsFull: []byte(before)})

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "updated the Beads section in AGENTS.md, added the Codefall, Local environment and Testing " +
		"sections to AGENTS.md and created CLAUDE.md"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want = "# Project\n\nWhat it is.\n\n" + beadsSection +
		"\n\n## Afterwards\n\nMore of the project's own words.\n\n" + codefallSection + "\n\n" +
		localSection + "\n\n" + testingSection + "\n"

	if got := string(files.files[agentsFull]); got != want {
		t.Errorf("AGENTS.md =\n%q\nwant\n%q", got, want)
	}
}

// A project set up before the Local environment and Testing sections existed has the Beads section
// and nothing else of codefall's. The next run adds the ones it is missing and leaves the one it has
// alone, which is how a section reaches every project without a scaffold.
func TestAgentsStepAddsTheSectionAProjectIsMissing(t *testing.T) {
	before := "# Project\n\n" + beadsSection + "\n"

	files, result := run(t, map[string][]byte{
		agentsFull:       []byte(before),
		claudeMemoryFull: []byte("See [AGENTS.md](AGENTS.md).\n"),
	})

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	if want := "added the Codefall, Local environment and Testing sections to AGENTS.md"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want := "# Project\n\n" + beadsSection + "\n\n" + codefallSection + "\n\n" + localSection + "\n\n" +
		testingSection + "\n"
	if got := string(files.files[agentsFull]); got != want {
		t.Errorf("AGENTS.md =\n%q\nwant\n%q", got, want)
	}
}

// A project set up before the Codefall section existed has the other three. The next run adds the
// Codefall section at the end, after them: first is where it goes in a file that has none, and the
// step never moves what somebody else wrote, including sections it wrote itself on an earlier run.
func TestAgentsStepAddsTheCodefallSectionAfterTheThreeAProjectHas(t *testing.T) {
	before := "# Project\n\n" + threeSections + "\n"

	files, result := run(t, map[string][]byte{
		agentsFull:       []byte(before),
		claudeMemoryFull: []byte("See [AGENTS.md](AGENTS.md).\n"),
	})

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	if want := "added the Codefall section to AGENTS.md"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want := "# Project\n\n" + threeSections + "\n\n" + codefallSection + "\n"
	if got := string(files.files[agentsFull]); got != want {
		t.Errorf("AGENTS.md =\n%q\nwant\n%q", got, want)
	}
}

// A Codefall section from an earlier version is rewritten between its own markers, and the three
// sections after it, already current, are not touched.
func TestAgentsStepReplacesTheCodefallSectionInPlace(t *testing.T) {
	before := "# Project\n\n" + domain.CodefallSectionBegin + "\n## Codefall\n\nSomething older.\n" +
		domain.CodefallSectionEnd + "\n\n" + threeSections + "\n"

	files, result := run(t, map[string][]byte{
		agentsFull:       []byte(before),
		claudeMemoryFull: []byte("See [AGENTS.md](AGENTS.md).\n"),
	})

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	if want := "updated the Codefall section in AGENTS.md"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want := "# Project\n\n" + allSections + "\n"
	if got := string(files.files[agentsFull]); got != want {
		t.Errorf("AGENTS.md =\n%q\nwant\n%q", got, want)
	}
}

// A second run has nothing to do: every section is what codefall would have written, and the file is
// not touched. This is what makes init safe to run again.
func TestAgentsStepSkipsSectionsThatAreCurrent(t *testing.T) {
	before := "# Project\n\n" + allSections + "\n"

	files, result := run(t, map[string][]byte{
		agentsFull:       []byte(before),
		claudeMemoryFull: []byte("See [AGENTS.md](AGENTS.md).\n"),
	})

	if result.Outcome != domain.OutcomeSkipped {
		t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
	}

	if want := "codefall's sections in AGENTS.md are current"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if got := string(files.files[agentsFull]); got != before {
		t.Errorf("AGENTS.md = %q, want it untouched", got)
	}
}

// The sections are current and CLAUDE.md is not there: the step has one thing to report, and it is
// that one thing rather than a sentence about AGENTS.md that would not be true.
func TestAgentsStepReportsTheOneThingItDid(t *testing.T) {
	_, result := run(t, map[string][]byte{agentsFull: []byte(allSections + "\n")})

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	if want := "created CLAUDE.md"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// A CLAUDE.md the project already has is left alone, whatever it says: it is somebody's file, and a
// pointer written over it would throw the rules away rather than point at them.
func TestAgentsStepLeavesAClaudeFileThatIsAlreadyThere(t *testing.T) {
	for _, before := range []string{"# House rules\n\nDo the thing.\n", ""} {
		files, _ := run(t, map[string][]byte{claudeMemoryFull: []byte(before)})

		if got := string(files.files[claudeMemoryFull]); got != before {
			t.Errorf("CLAUDE.md = %q, want it untouched", got)
		}
	}
}

// Only Claude Code reads CLAUDE.md, so only Claude Code gets one. The harness the section itself is
// written for is every harness: AGENTS.md is the file they all read.
func TestAgentsStepWritesClaudeMdOnlyForClaudeCode(t *testing.T) {
	files := settled("{}")

	request := beadsRequest()
	request.Harnesses = []string{"aider"}

	result, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).agents(t.Context(), request)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}

	if want := "created AGENTS.md with the Codefall, Beads, Local environment and Testing sections"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if _, written := files.files[claudeMemoryFull]; written {
		t.Errorf("CLAUDE.md = %q, want no file for a harness that does not read one",
			files.files[claudeMemoryFull])
	}
}

// An opening marker with no closing one is a file the step cannot edit without guessing where
// codefall's words stop, so it stops the run instead and says which marker it wanted — for either
// section, and before anything is written, so a file with one good section and one broken one
// comes back untouched.
func TestAgentsStepRefusesAnUnclosedSection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		before string
		begin  string
		end    string
	}{
		{
			name:   "the Beads section",
			before: "# Project\n\n" + domain.BeadsSectionBegin + "\n## Beads\n\nHalf a section.\n",
			begin:  domain.BeadsSectionBegin,
			end:    domain.BeadsSectionEnd,
		},
		{
			name: "the Local environment section, after a Beads section that is out of date",
			before: "# Project\n\n" + domain.BeadsSectionBegin + "\nOlder.\n" + domain.BeadsSectionEnd +
				"\n\n" + domain.LocalSectionBegin + "\nHalf a section.\n",
			begin: domain.LocalSectionBegin,
			end:   domain.LocalSectionEnd,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("{}")
			files.files[agentsFull] = []byte(tc.before)

			_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).agents(t.Context(), beadsRequest())
			if err == nil || !strings.Contains(err.Error(), "AGENTS.md has "+tc.begin+" with no "+tc.end+" after it") {
				t.Errorf("agents error = %v, want it to name the file and the missing marker", err)
			}

			if got := string(files.files[agentsFull]); got != tc.before {
				t.Errorf("AGENTS.md = %q, want it untouched", got)
			}
		})
	}
}

// A file that cannot be read or written is the step's error, named so the reader knows which file.
func TestAgentsStepReportsAFileItCannotUse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*fakeFileSystem)
		want  string
	}{
		{
			name:  "AGENTS.md cannot be read",
			setup: func(f *fakeFileSystem) { f.errs[agentsFull] = errors.New("permission denied") },
			want:  "read AGENTS.md",
		},
		{
			name:  "AGENTS.md cannot be written",
			setup: func(f *fakeFileSystem) { f.errs["write "+agentsFull] = errors.New("read-only file system") },
			want:  "write AGENTS.md",
		},
		{
			name:  "CLAUDE.md cannot be read",
			setup: func(f *fakeFileSystem) { f.errs[claudeMemoryFull] = errors.New("permission denied") },
			want:  "read CLAUDE.md",
		},
		{
			name: "CLAUDE.md cannot be written",
			setup: func(f *fakeFileSystem) {
				f.errs["write "+claudeMemoryFull] = errors.New("read-only file system")
			},
			want: "write CLAUDE.md",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("{}")
			tc.setup(files)

			_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).agents(t.Context(), beadsRequest())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("agents error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

// The Testing section names the root the project declared, wherever the request puts it, and no
// placeholder survives into somebody's file.
func TestAgentsStepNamesTheDeclaredTestingRoot(t *testing.T) {
	files := settled("{}")

	request := beadsRequest()
	request.TestDir = "packages/web/e2e"

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).agents(t.Context(), request); err != nil {
		t.Fatalf("agents: %v", err)
	}

	got := string(files.files[agentsFull])

	if strings.Contains(got, domain.TestingRootPlaceholder) {
		t.Errorf("AGENTS.md still holds %q:\n%s", domain.TestingRootPlaceholder, got)
	}

	if !strings.Contains(got, "packages/web/e2e/") {
		t.Errorf("AGENTS.md does not name the declared root:\n%s", got)
	}
}

// A section the tree does not hold, or one whose markers are not where the splice expects them, is a
// binary shipped wrong. The step stops before it reads or writes anything of the project's, and
// names the file, so a broken build fails every install the same way rather than writing a section
// a later run could not find.
func TestAgentsStepRefusesASectionTheTreeShipsWrong(t *testing.T) {
	for _, tc := range []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "the file is not in the tree",
			body: nil,
			want: "read agents/sections/beads.md",
		},
		{
			name: "the opening marker is not the first line",
			body: []byte("## Beads\n" + domain.BeadsSectionBegin + "\n\nWords.\n" + domain.BeadsSectionEnd + "\n"),
			want: "agents/sections/beads.md does not open with " + domain.BeadsSectionBegin,
		},
		{
			name: "the closing marker is not the last line",
			body: []byte(domain.BeadsSectionBegin + "\n\nWords.\n" + domain.BeadsSectionEnd + "\n\nMore.\n"),
			want: "agents/sections/beads.md does not close with " + domain.BeadsSectionEnd,
		},
		{
			name: "the file does not end with a newline",
			body: []byte(domain.BeadsSectionBegin + "\n\nWords.\n" + domain.BeadsSectionEnd),
			want: "agents/sections/beads.md does not close with " + domain.BeadsSectionEnd,
		},
		{
			name: "a marker appears twice",
			body: []byte(domain.BeadsSectionBegin + "\n\n" + domain.BeadsSectionBegin + "\n" + domain.BeadsSectionEnd + "\n"),
			want: "agents/sections/beads.md holds a marker more than once",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("{}")
			files.files[agentsFull] = []byte("# Project\n")

			source := newFakeExtensionSource()
			if tc.body == nil {
				delete(source.data, "agents/sections/beads.md")
			} else {
				source.data["agents/sections/beads.md"] = tc.body
			}

			_, err := NewInitialize(files, toolsInstalled(), source, noChanges()).agents(t.Context(), beadsRequest())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("agents error = %v, want it to say %q", err, tc.want)
			}

			if got := string(files.files[agentsFull]); got != "# Project\n" {
				t.Errorf("AGENTS.md = %q, want it untouched", got)
			}
		})
	}
}

// The run reaches the section last, after bd init has made whatever commit it was going to make:
// codefall's edit to AGENTS.md is the author's to commit, not bd's.
func TestAgentsStepRunsAfterBeads(t *testing.T) {
	observer := &recordingObserver{}

	if _, err := NewInitialize(settled("{}"), toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(
		t.Context(), beadsRequest(), observer,
	); err != nil {
		t.Fatalf("Run: %v", err)
	}

	beads := slices.Index(observer.started, domain.BeadsStep)
	agents := slices.Index(observer.started, domain.AgentsStep)

	if beads < 0 || agents < 0 || agents < beads {
		t.Errorf("started = %+v, want %+v after %+v", observer.started, domain.AgentsStep, domain.BeadsStep)
	}
}

// firstLine is the opening line of a file, for a message about what it starts with.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")

	return line
}
