package application

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// prefixed is a settled project's run under the prefix cf.
func prefixed(names ...string) Request {
	request := requestFor(names...)
	request.SkillPrefix = settings.SkillPrefixCf

	return request
}

// The rename every copy and splice goes through is built from the prefix the request carries and the
// skills the tree ships, and reaches both fetches: the skills directory and .codefall/.
func TestExtensionStepRenamesOntoTheRequestsPrefix(t *testing.T) {
	fetcher := newFakeExtensionSource()

	runFor(t, settled(""), fetcher, prefixed(harness.Claude))

	if len(fetcher.calls) != 2 {
		t.Fatalf("fetcher calls = %+v, want two", fetcher.calls)
	}

	for _, call := range fetcher.calls {
		if got := call.rename.Prefix(); got != settings.SkillPrefixCf {
			t.Errorf("the fetch into %s renamed onto %q, want %q", call.dir, got, settings.SkillPrefixCf)
		}

		if got := call.rename.Text("codefall-design"); got != "cf-design" {
			t.Errorf("the fetch into %s was handed a rename that gives %q, want cf-design", call.dir, got)
		}
	}

	// A request that carries no answer takes the default, which is what every project installed
	// before the field existed and what the source already says, so the rename changes nothing.
	fetcher = newFakeExtensionSource()
	runFor(t, settled(""), fetcher, requestFor(harness.Claude))

	if got := fetcher.calls[0].rename.Prefix(); got != settings.DefaultSkillPrefix {
		t.Errorf("a request with no prefix renamed onto %q, want the default %q", got, settings.DefaultSkillPrefix)
	}
}

// The sections spliced into AGENTS.md and the skeleton written at the testing root name the verbs, so
// each is written under the project's prefix; the manifest records the prefix the run installed under.
func TestRunWritesTheSectionsTheSkeletonAndTheManifestUnderThePrefix(t *testing.T) {
	files := settled("{}")
	request := prefixed(harness.Claude)
	request.CLIVersion = "v1.2.3"

	runFor(t, files, newFakeExtensionSource(), request)

	agents := string(files.files[agentsFull])
	if !strings.Contains(agents, "run through `/cf-test`") || strings.Contains(agents, "codefall-test") {
		t.Errorf("AGENTS.md =\n%s\nwant the Testing section to name /cf-test and nothing under codefall-", agents)
	}

	skeleton := string(files.files[testingAgentsFull])
	if !strings.Contains(skeleton, "`/cf-test`") || strings.Contains(skeleton, "codefall-test") {
		t.Errorf("testing/AGENTS.md =\n%s\nwant the skeleton to name /cf-test", skeleton)
	}

	if got := recordedManifestIn(t, files).SkillPrefix; got != settings.SkillPrefixCf {
		t.Errorf("manifest skillPrefix = %q, want %q", got, settings.SkillPrefixCf)
	}
}

// New settings state the prefix, the default included, and the step's sentence names a chosen one.
func TestSettingsStepWritesThePrefix(t *testing.T) {
	files := newFakeFileSystem()
	request := Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Claude},
		SkillPrefix: settings.SkillPrefixCfall}

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(t.Context(), request, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if want := "wrote .codefall/settings.json (tracker: beads, harnesses: claude, skill prefix: cfall)"; report.Results()[0].Detail != want {
		t.Errorf("detail = %q, want %q", report.Results()[0].Detail, want)
	}

	if !strings.Contains(string(files.files[settingsFull]), "\n  \"skillPrefix\": \"cfall\",\n") {
		t.Errorf("settings.json =\n%s\nwant the prefix stated", files.files[settingsFull])
	}

	// An unknown prefix is the format's refusal, before anything is written.
	request.SkillPrefix = "code"

	_, err = NewInitialize(newFakeFileSystem(), toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(t.Context(), request, nil)
	if err == nil || !strings.Contains(err.Error(), `unknown skill prefix "code"`) {
		t.Errorf("Run with an unknown prefix error = %v, want the prefix named", err)
	}
}

// The prefix the settings declare comes back as written, the default when they declare none or are
// not there, and an error for a value the format refuses: the run would otherwise install under a
// name the file does not say.
func TestDeclaredSkillPrefix(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings string
		missing  bool
		want     string
		wantErr  string
	}{
		{name: "a declared prefix", settings: `{"skillPrefix": "cf"}`, want: settings.SkillPrefixCf},
		{name: "settings that declare none", settings: `{"tracker": "beads"}`, want: settings.DefaultSkillPrefix},
		{name: "no settings", missing: true, want: settings.DefaultSkillPrefix},
		{name: "a prefix the format refuses", settings: `{"skillPrefix": "code"}`, wantErr: `unknown skill prefix "code"`},
		{name: "settings that do not decode", settings: `{`, wantErr: "decode .codefall/settings.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem()
			if !tc.missing {
				files.files[settingsFull] = []byte(tc.settings)
			}

			got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).DeclaredSkillPrefix(workingDir)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("DeclaredSkillPrefix error = %v, want it to mention %q", err, tc.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("DeclaredSkillPrefix: %v", err)
			}

			if got != tc.want {
				t.Errorf("DeclaredSkillPrefix = %q, want %q", got, tc.want)
			}
		})
	}

	files := newFakeFileSystem()
	files.errs[settingsFull] = errors.New("permission denied")

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).DeclaredSkillPrefix(workingDir); err == nil ||
		!strings.Contains(err.Error(), "permission denied") {
		t.Errorf("DeclaredSkillPrefix error = %v, want the read failure carried", err)
	}
}

// A skill directory the previous run wrote under the former prefix is removed because this run wrote
// nothing into it, and the report calls that the rename it is. A skill the tree renamed composes with
// the prefix: conceptualize became codefall-envision, which this project installs as cf-envision.
func TestCleanupNamesAPrefixChangeAsARename(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"claude": {"version": "v0.28.0", "files": [".claude/skills/codefall-design/SKILL.md", ".claude/skills/conceptualize/SKILL.md"]}},
  "shared": {"version": "v0.28.0", "files": [".codefall/hooks/shared/codefall-block-merge-to-main.sh", ".codefall/shared/preflight.sh"]}
}`, ".claude/skills/codefall-design/SKILL.md", ".claude/skills/conceptualize/SKILL.md")

	result := cleanupResult(t, runFor(t, files, newFakeExtensionSource(), prefixed(harness.Claude)))

	want := "removed .claude/skills/codefall-design/ (renamed to cf-design) and .claude/skills/conceptualize/ (renamed to cf-envision)"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	// Moving back reads the same way: the directory under cf is the one under codefall now.
	files = upgrading(`{
  "harnesses": {"claude": {"version": "v0.28.0", "files": [".claude/skills/cf-design/SKILL.md"]}},
  "shared": {"version": "v0.28.0", "files": [".codefall/shared/preflight.sh"]},
  "skillPrefix": "cf"
}`, ".claude/skills/cf-design/SKILL.md")

	result = cleanupResult(t, runFor(t, files, newFakeExtensionSource(), requestFor(harness.Claude)))

	if want := "removed .claude/skills/cf-design/ (renamed to codefall-design)"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}
}

// When the prefix moved, the step also says what the project's own files still call the skills,
// read from git so only tracked files outside what the run wrote are named, and which customization
// files the skills will no longer find. None of those files is changed. AGENTS.md is read directly,
// outside codefall's sections, because the sections still carry the former names until the agents
// step rewrites them later in the run: a mention inside one is not the project's.
func TestCleanupReportsTheProjectsOwnMentionsOfTheFormerPrefix(t *testing.T) {
	files := upgrading(`{
  "harnesses": {"claude": {"version": "v0.28.0", "files": [".claude/skills/codefall-design/SKILL.md"]}},
  "shared": {"version": "v0.28.0", "files": [".codefall/shared/preflight.sh"]}
}`, ".claude/skills/codefall-design/SKILL.md", ".codefall/skills/codefall-design/CUSTOMIZE.md")
	files.files[agentsFull] = []byte("# Project\n\nRun /codefall-design before anything else.\n\n" + allSections + "\n")

	grep := "git grep -l -I -w -E codefall-(design|envision|test) -- . :(exclude).codefall :(exclude)AGENTS.md :(exclude).claude/skills"

	runner := toolsInstalled()
	runner.runs[grep] = CommandResult{Stdout: "docs/designs/DESIGN-001.md\n"}

	report, err := NewInitialize(files, runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), prefixed(harness.Claude), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := "removed .claude/skills/codefall-design/ (renamed to cf-design); " +
		"2 tracked files still name a skill as codefall-<verb> and are left as they are, the project's own: " +
		"AGENTS.md and docs/designs/DESIGN-001.md; " +
		"move .codefall/skills/codefall-design/CUSTOMIZE.md to .codefall/skills/cf-design/ for cf-design to read it"
	if got := cleanupResult(t, report).Detail; got != want {
		t.Errorf("detail = %q, want %q", got, want)
	}

	if _, still := files.files[filepath.Join(workingDir, ".codefall/skills/codefall-design/CUSTOMIZE.md")]; !still {
		t.Error("the customization file is gone, want it left where it was")
	}

	// git finding nothing is a report with nothing to add rather than an error, and a mention inside
	// codefall's own sections of AGENTS.md is not the project's.
	runner = toolsInstalled()
	runner.runs[grep] = CommandResult{ExitCode: 1}

	files = upgrading(`{
  "harnesses": {"claude": {"version": "v0.28.0", "files": [".claude/skills/codefall-design/SKILL.md"]}},
  "shared": {"version": "v0.28.0", "files": [".codefall/shared/preflight.sh"]}
}`, ".claude/skills/codefall-design/SKILL.md")
	files.files[agentsFull] = []byte("# Project\n\n" + allSections + "\n")

	report, err = NewInitialize(files, runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), prefixed(harness.Claude), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if want := "removed .claude/skills/codefall-design/ (renamed to cf-design)"; cleanupResult(t, report).Detail != want {
		t.Errorf("detail = %q, want %q", cleanupResult(t, report).Detail, want)
	}

	// A run whose prefix did not move asks git nothing.
	runner = toolsInstalled()
	runFor(t, upgrading(`{"harnesses": {"claude": {"version": "v0.28.0", "files": [".claude/skills/design/SKILL.md"]}},
  "shared": {"version": "v0.28.0", "files": [".codefall/shared/preflight.sh"]}}`, ".claude/skills/design/SKILL.md"),
		newFakeExtensionSource(), requestFor(harness.Claude))

	for _, call := range runner.calls {
		if strings.HasPrefix(call.command, "git grep") {
			t.Errorf("ran %q on a run whose prefix did not move", call.command)
		}
	}
}

// A tree that names a skill any other way than codefall-<verb> is a binary shipped wrong, and the run
// says so in the extension step rather than installing half a rename.
func TestRunRefusesATreeWhoseSkillsAreNotPrefixed(t *testing.T) {
	fetcher := newFakeExtensionSource()
	fetcher.skills = []string{"design"}

	_, err := NewInitialize(settled(""), toolsInstalled(), fetcher, noChanges()).Run(t.Context(), prefixed(harness.Claude), nil)
	if err == nil || !strings.HasPrefix(err.Error(), domain.ExtensionStep.ID+": ") ||
		!strings.Contains(err.Error(), `skill "design" is not named codefall-<verb>`) {
		t.Errorf("Run error = %v, want the extension step to name the skill", err)
	}
}
