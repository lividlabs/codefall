package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

const workingDir = "/work"

var (
	codefallDir = filepath.Join(workingDir, ".codefall")
	// A file the manifest records for each harness, which is what tells an install apart from a
	// skills directory that was already there: the skills go into the directory the harness reads,
	// and only a finished run puts them there.
	claudeSkill       = filepath.Join(workingDir, ".claude", "skills", "design", "SKILL.md")
	agentsSkill       = filepath.Join(workingDir, ".agents", "skills", "design", "SKILL.md")
	settingsPath      = filepath.Join(codefallDir, "settings.json")
	manifestPath      = filepath.Join(workingDir, manifest.Name)
	ignorePath        = filepath.Join(workingDir, settings.IgnoreName)
	gitIgnorePath     = filepath.Join(workingDir, settings.GitIgnoreName)
	gitAttributesPath = filepath.Join(workingDir, settings.GitAttributesName)
	// The script the fixture's local block names, by the relative path the block uses.
	localScript = filepath.Join(workingDir, "scripts", "local.sh")
	// The directory the fixture's settings declare their test cases live in.
	testingDir   = filepath.Join(workingDir, settings.DefaultTestDir)
	schemaRemedy = "create .codefall/settings.json; schema: " + settings.SchemaID
	fixRemedy    = "fix the fields above; schema: " + settings.SchemaID
	ignoreRemedy = "add " + settings.IgnoreEntry + " to " + settings.IgnoreName +
		", or run codefall init again"
	testsIgnoreRemedy = "add " + settings.IgnoreEntryTests + " to " + settings.IgnoreName +
		", or run codefall init again"
	stampRemedy = "add " + settings.RefreshStamp + " to " + settings.GitIgnoreName +
		", or run codefall init again"
	mergeRemedy = "add " + settings.InteractionsAttribute + " to " + settings.GitAttributesName +
		", or run codefall init again"
	leftoverRemedy = "remove the files " + manifest.Name +
		" lists for codex; codefall never deletes what it wrote"
)

// installedManifest is what a finished run leaves on record: one entry for the harness the settings
// name. droppedManifest is the same project after it stopped naming codex, with everything codefall
// wrote for codex still recorded.
const (
	installedManifest = `{"harnesses": {
  "claude": {"version": "v1.2.3", "files": [".claude/skills/design/SKILL.md"]}
}}`
	droppedManifest = `{"harnesses": {
  "claude": {"version": "v1.2.3", "files": [".claude/skills/design/SKILL.md"]},
  "codex": {"version": "v1.2.3", "files": [".agents/skills/design/SKILL.md"]}
}}`
)

const validSettings = `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude"],
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "local": { "start": "scripts/local.sh start", "update": "scripts/local.sh update" },
  "test": { "dir": "testing", "runners": ["playwright"] }
}`

// twoHarnessSettings is the same project set up for a second harness, which reads a second directory.
const twoHarnessSettings = `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude", "codex"],
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "local": { "start": "scripts/local.sh start", "update": "scripts/local.sh update" },
  "test": { "dir": "testing", "runners": ["playwright"] }
}`

// undeclaredSettings is a project that has never been equipped: complete settings with no local
// block, which is every project set up before the block existed.
const undeclaredSettings = `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude"],
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "test": { "dir": "testing", "runners": ["playwright"] }
}`

// withAgents is the fixture's settings with an agents list and, when they are not empty, a review
// order and a per-harness override, each written as JSON.
func withAgents(agents, reviewOrder, byHarness string) string {
	document := `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude"],
  "agents": ` + agents + `,`

	if byHarness != "" {
		document += "\n  \"agentsByHarness\": " + byHarness + ","
	}

	review := `{ "postToPullRequest": false`
	if reviewOrder != "" {
		review += `, "agents": ` + reviewOrder
	}

	return document + `
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "review": ` + review + ` },
  "local": { "start": "scripts/local.sh start", "update": "scripts/local.sh update" },
  "test": { "dir": "testing", "runners": ["playwright"] }
}`
}

// twoAgents is a project that reaches for Codex first and its own subagent after.
const twoAgents = `[
    { "name": "architect", "harness": "codex", "model": "gpt-5-codex" },
    { "name": "subagent", "harness": "current" }
  ]`

// withLocal is the fixture's settings with a different local block.
func withLocal(start, update string) string {
	return `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude"],
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "local": { "start": ` + fmt.Sprintf("%q", start) + `, "update": ` + fmt.Sprintf("%q", update) + ` },
  "test": { "dir": "testing", "runners": ["playwright"] }
}`
}

// withTest is the fixture's settings with a different test block, or with none when the block is
// empty — which is every project set up before the block existed.
func withTest(block string) string {
	document := `{
  "$schema": "` + settings.SchemaID + `",
  "version": 1,
  "tracker": "github",
  "harnesses": ["claude"],
  "github": { "issuesRepo": "lividlabs/codefall-cli", "issuesProject": 3 },
  "local": { "start": "scripts/local.sh start", "update": "scripts/local.sh update" }`

	if block == "" {
		return document + "\n}"
	}

	return document + ",\n  \"test\": " + block + "\n}"
}

// --- fakes -------------------------------------------------------------------------------------

type fakeFileSystem struct {
	dirs  map[string]bool
	files map[string][]byte
	errs  map[string]error
}

func (f *fakeFileSystem) DirExists(path string) (bool, error) {
	if err, ok := f.errs[path]; ok {
		return false, err
	}

	return f.dirs[path], nil
}

func (f *fakeFileSystem) Exists(path string) (bool, error) {
	if err, ok := f.errs[path]; ok {
		return false, err
	}

	_, isFile := f.files[path]

	return isFile || f.dirs[path], nil
}

func (f *fakeFileSystem) ReadFile(path string) ([]byte, error) {
	if err, ok := f.errs[path]; ok {
		return nil, err
	}

	data, ok := f.files[path]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
	}

	return data, nil
}

type runCall struct {
	dir     string
	command string
}

type fakeCommandRunner struct {
	paths map[string]string
	runs  map[string]CommandResult
	errs  map[string]error
	calls []runCall
}

func (r *fakeCommandRunner) LookPath(name string) mo.Option[string] {
	if path, ok := r.paths[name]; ok {
		return mo.Some(path)
	}

	return mo.None[string]()
}

func (r *fakeCommandRunner) Run(
	_ context.Context, dir, name string, args ...string,
) (CommandResult, error) {
	command := strings.TrimSpace(name + " " + strings.Join(args, " "))
	r.calls = append(r.calls, runCall{dir: dir, command: command})

	if err, ok := r.errs[command]; ok {
		return CommandResult{}, err
	}

	return r.runs[command], nil
}

// healthy is the fixture where every check passes; each case mutates it.
func healthy() (*fakeFileSystem, *fakeCommandRunner) {
	files := &fakeFileSystem{
		dirs: map[string]bool{codefallDir: true, testingDir: true},
		files: map[string][]byte{
			claudeSkill:  []byte("---\nname: design\n---\n"),
			settingsPath: []byte(validSettings),
			manifestPath: []byte(installedManifest),
			ignorePath: []byte(settings.IgnoreComment + "\n" + settings.IgnoreEntry + "\n\n" +
				settings.IgnoreTestsComment + "\n" + settings.IgnoreEntryTests + "\n"),
			gitIgnorePath:     []byte("node_modules/\n\n" + settings.GitIgnoreComment + "\n" + settings.RefreshStamp + "\n"),
			gitAttributesPath: []byte(settings.GitAttributesComment + "\n" + settings.InteractionsAttribute + "\n"),
			localScript:       []byte("#!/usr/bin/env bash\n"),
		},
		errs: map[string]error{},
	}

	runner := &fakeCommandRunner{
		paths: map[string]string{
			"bd": "/opt/homebrew/bin/bd", "gh": "/opt/homebrew/bin/gh",
			"make": "/usr/bin/make", "docker": "/usr/local/bin/docker",
		},
		runs: map[string]CommandResult{
			"bd version":                  {Stdout: "bd version 1.2.2 (Homebrew)\n"},
			"bd info":                     {Stdout: "beads: 12 issues open\n"},
			"gh --version":                {Stdout: "gh version 2.97.0 (2026-07-31)\nhttps://github.com/cli/cli\n"},
			"gh auth status --json hosts": {Stdout: ghHosts(`{"state":"success","active":true,"host":"github.com","login":"djensen47","tokenSource":"keyring","scopes":"gist, project, read:org, repo, workflow","gitProtocol":"ssh"}`)},
		},
		errs: map[string]error{},
	}

	return files, runner
}

func ghHosts(accounts ...string) string {
	return `{"hosts":{"github.com":[` + strings.Join(accounts, ",") + `]}}`
}

// --- the table ---------------------------------------------------------------------------------

type outcome struct {
	id     string
	status domain.Status
}

var allPass = []outcome{
	{domain.CodefallDir.ID, domain.StatusPass},
	{domain.SettingsFile.ID, domain.StatusPass},
	{domain.SettingsJSON.ID, domain.StatusPass},
	{domain.SettingsComplete.ID, domain.StatusPass},
	{domain.ReviewsIgnored.ID, domain.StatusPass},
	{domain.TestsIgnored.ID, domain.StatusPass},
	{domain.StampIgnored.ID, domain.StatusPass},
	{domain.InteractionsMerged.ID, domain.StatusPass},
	{domain.HarnessNames.ID, domain.StatusPass},
	{domain.HarnessesInstalled.ID, domain.StatusPass},
	{domain.HarnessesLeftOver.ID, domain.StatusPass},
	{domain.AgentsRunnable.ID, domain.StatusPass},
	{domain.AgentsCurrent.ID, domain.StatusPass},
	{domain.LocalDeclared.ID, domain.StatusPass},
	{domain.LocalRunnable.ID, domain.StatusPass},
	{domain.TestDeclared.ID, domain.StatusPass},
	{domain.TestEquipped.ID, domain.StatusPass},
	{domain.TestDirExists.ID, domain.StatusPass},
	{domain.BeadsInstalled.ID, domain.StatusPass},
	{domain.BeadsInitialized.ID, domain.StatusPass},
	{domain.GHInstalled.ID, domain.StatusPass},
	{domain.GHAuthenticated.ID, domain.StatusPass},
	{domain.GHScopes.ID, domain.StatusPass},
}

// outcomes restates allPass with some statuses changed and the skipped checks removed — a check that
// never ran is absent from the report, not present with a status of its own.
func outcomes(changes map[string]domain.Status, absent ...string) []outcome {
	result := make([]outcome, 0, len(allPass))

	for _, o := range allPass {
		if slices.Contains(absent, o.id) {
			continue
		}

		if status, ok := changes[o.id]; ok {
			o.status = status
		}

		result = append(result, o)
	}

	return result
}

// The skip rules, named so each case reads as the rule it exercises.
var (
	// The harnesses and local checks read the same settings the group above validates, so whatever
	// skips those checks skips them too.
	afterCodefallDir = []string{
		domain.SettingsFile.ID, domain.SettingsJSON.ID, domain.SettingsComplete.ID,
		domain.ReviewsIgnored.ID, domain.TestsIgnored.ID, domain.StampIgnored.ID,
		domain.InteractionsMerged.ID,
		domain.HarnessNames.ID, domain.HarnessesInstalled.ID, domain.HarnessesLeftOver.ID,
		domain.AgentsRunnable.ID, domain.AgentsCurrent.ID,
		domain.LocalDeclared.ID, domain.LocalRunnable.ID,
		domain.TestDeclared.ID, domain.TestEquipped.ID, domain.TestDirExists.ID,
	}
	afterSettingsFile = []string{
		domain.SettingsJSON.ID, domain.SettingsComplete.ID,
		domain.ReviewsIgnored.ID, domain.TestsIgnored.ID, domain.StampIgnored.ID,
		domain.InteractionsMerged.ID,
		domain.HarnessNames.ID, domain.HarnessesInstalled.ID, domain.HarnessesLeftOver.ID,
		domain.AgentsRunnable.ID, domain.AgentsCurrent.ID,
		domain.LocalDeclared.ID, domain.LocalRunnable.ID,
		domain.TestDeclared.ID, domain.TestEquipped.ID, domain.TestDirExists.ID,
	}
	afterSettingsJSON = []string{
		domain.SettingsComplete.ID,
		domain.ReviewsIgnored.ID, domain.TestsIgnored.ID, domain.StampIgnored.ID,
		domain.InteractionsMerged.ID,
		domain.HarnessNames.ID, domain.HarnessesInstalled.ID, domain.HarnessesLeftOver.ID,
		domain.AgentsRunnable.ID, domain.AgentsCurrent.ID,
		domain.LocalDeclared.ID, domain.LocalRunnable.ID,
		domain.TestDeclared.ID, domain.TestEquipped.ID, domain.TestDirExists.ID,
	}
	afterSettingsDone = []string{
		domain.ReviewsIgnored.ID, domain.TestsIgnored.ID, domain.StampIgnored.ID,
		domain.InteractionsMerged.ID,
		domain.HarnessNames.ID, domain.HarnessesInstalled.ID, domain.HarnessesLeftOver.ID,
		domain.AgentsRunnable.ID, domain.AgentsCurrent.ID,
		domain.LocalDeclared.ID, domain.LocalRunnable.ID,
		domain.TestDeclared.ID, domain.TestEquipped.ID, domain.TestDirExists.ID,
	}
	afterLocalUndeclared = []string{domain.LocalRunnable.ID}
	// A project that declares no testing root has nothing for the two checks after it to ask about.
	afterTestUndeclared = []string{domain.TestEquipped.ID, domain.TestDirExists.ID}
	afterBeadsMissing   = []string{domain.BeadsInitialized.ID}
	afterGHMissing      = []string{domain.GHAuthenticated.ID, domain.GHScopes.ID}
	afterGHAuth         = []string{domain.GHScopes.ID}
)

func TestDiagnoseRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mutate     func(*fakeFileSystem, *fakeCommandRunner)
		want       []outcome
		target     string
		wantDetail string
		wantRemedy mo.Option[string]
	}{
		{
			name:   "everything is in place",
			mutate: func(*fakeFileSystem, *fakeCommandRunner) {},
			want:   allPass,
			target: domain.GHScopes.ID,
		},
		{
			name: "the .codefall directory is missing",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.dirs, codefallDir)
			},
			want:       outcomes(map[string]domain.Status{domain.CodefallDir.ID: domain.StatusFail}, afterCodefallDir...),
			target:     domain.CodefallDir.ID,
			wantDetail: ".codefall/ not found",
			wantRemedy: mo.Some(schemaRemedy),
		},
		{
			name: "the .codefall directory cannot be stat'd",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.errs[codefallDir] = errors.New("permission denied")
			},
			want:       outcomes(map[string]domain.Status{domain.CodefallDir.ID: domain.StatusFail}, afterCodefallDir...),
			target:     domain.CodefallDir.ID,
			wantDetail: "Cannot stat " + codefallDir + ": permission denied",
			wantRemedy: mo.None[string](),
		},
		{
			name: "settings.json is missing",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, settingsPath)
			},
			want:       outcomes(map[string]domain.Status{domain.SettingsFile.ID: domain.StatusFail}, afterSettingsFile...),
			target:     domain.SettingsFile.ID,
			wantDetail: ".codefall/settings.json not found",
			wantRemedy: mo.Some(schemaRemedy),
		},
		{
			name: "settings.json cannot be read",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.errs[settingsPath] = errors.New("permission denied")
			},
			want:       outcomes(map[string]domain.Status{domain.SettingsFile.ID: domain.StatusFail}, afterSettingsFile...),
			target:     domain.SettingsFile.ID,
			wantDetail: "Cannot read .codefall/settings.json: permission denied",
			wantRemedy: mo.None[string](),
		},
		{
			name: "settings.json is not valid JSON",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte("{\n  \"version\": 1,,\n}")
			},
			want:   outcomes(map[string]domain.Status{domain.SettingsJSON.ID: domain.StatusFail}, afterSettingsJSON...),
			target: domain.SettingsJSON.ID,
			wantDetail: "settings.json is not valid JSON at byte 18: " +
				"invalid character ',' looking for beginning of object key string",
			wantRemedy: mo.None[string](),
		},
		{
			name: "settings.json is valid JSON but not an object",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(`["nope"]`)
			},
			want: outcomes(map[string]domain.Status{domain.SettingsComplete.ID: domain.StatusFail},
				afterSettingsDone...),
			target:     domain.SettingsComplete.ID,
			wantDetail: "settings.json's top level must be a JSON object",
			wantRemedy: mo.Some(fixRemedy),
		},
		{
			name: "settings.json is incomplete",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(`{"version": 2}`)
			},
			want: outcomes(map[string]domain.Status{domain.SettingsComplete.ID: domain.StatusFail},
				afterSettingsDone...),
			target:     domain.SettingsComplete.ID,
			wantDetail: "settings.json is incomplete: version: must be 1; tracker: missing; harnesses: missing",
			wantRemedy: mo.Some(fixRemedy),
		},
		{
			// Both entries live in that file, so both checks have the same thing to say about it.
			name: ".ignore is missing, so what codefall commits would show up in every search",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, ignorePath)
			},
			want: outcomes(map[string]domain.Status{
				domain.ReviewsIgnored.ID: domain.StatusWarn, domain.TestsIgnored.ID: domain.StatusWarn}),
			target:     domain.ReviewsIgnored.ID,
			wantDetail: ".ignore not found",
			wantRemedy: mo.Some(ignoreRemedy),
		},
		{
			name: ".ignore is there but does not name the reviews directory",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[ignorePath] = []byte("vendor/\n" + settings.IgnoreEntryTests + "\n")
			},
			want:       outcomes(map[string]domain.Status{domain.ReviewsIgnored.ID: domain.StatusWarn}),
			target:     domain.ReviewsIgnored.ID,
			wantDetail: ".ignore does not name .codefall/reviews/",
			wantRemedy: mo.Some(ignoreRemedy),
		},
		{
			// Every project set up before the test run reports existed looks like this: the one entry
			// it was written with, and the newer one missing.
			name: ".ignore is there but does not name the tests directory",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[ignorePath] = []byte(settings.IgnoreEntry + "\n")
			},
			want:       outcomes(map[string]domain.Status{domain.TestsIgnored.ID: domain.StatusWarn}),
			target:     domain.TestsIgnored.ID,
			wantDetail: ".ignore does not name .codefall/tests/",
			wantRemedy: mo.Some(testsIgnoreRemedy),
		},
		{
			name: ".ignore names its entries but indented",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[ignorePath] = []byte("  " + settings.IgnoreEntry + "  \n\t" +
					settings.IgnoreEntryTests + "\n")
			},
			want:   allPass,
			target: domain.ReviewsIgnored.ID,
		},
		{
			// A committed stamp would tell every other clone it was current at a commit it never
			// refreshed at. Nothing is wrong until refresh writes one, so it warns.
			name: ".gitignore is missing, so a refresh stamp would be committed",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, gitIgnorePath)
			},
			want:       outcomes(map[string]domain.Status{domain.StampIgnored.ID: domain.StatusWarn}),
			target:     domain.StampIgnored.ID,
			wantDetail: ".gitignore not found",
			wantRemedy: mo.Some(stampRemedy),
		},
		{
			name: ".gitignore is there but does not name the stamp",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[gitIgnorePath] = []byte("node_modules/\n")
			},
			want:       outcomes(map[string]domain.Status{domain.StampIgnored.ID: domain.StatusWarn}),
			target:     domain.StampIgnored.ID,
			wantDetail: ".gitignore does not name " + settings.RefreshStamp,
			wantRemedy: mo.Some(stampRemedy),
		},
		{
			// Without the union merge, two branches that both appended to bd's interaction log
			// conflict on every merge, over a file that has only one right resolution.
			name: ".gitattributes is missing, so the interaction log would conflict on merge",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, gitAttributesPath)
			},
			want:       outcomes(map[string]domain.Status{domain.InteractionsMerged.ID: domain.StatusWarn}),
			target:     domain.InteractionsMerged.ID,
			wantDetail: ".gitattributes not found",
			wantRemedy: mo.Some(mergeRemedy),
		},
		{
			name: ".gitattributes is there but does not name the merge driver",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[gitAttributesPath] = []byte("* text=auto\n")
			},
			want:       outcomes(map[string]domain.Status{domain.InteractionsMerged.ID: domain.StatusWarn}),
			target:     domain.InteractionsMerged.ID,
			wantDetail: ".gitattributes does not name " + settings.InteractionsAttribute,
			wantRemedy: mo.Some(mergeRemedy),
		},
		{
			// The skills directory a harness reads may well exist for its own reasons; what says
			// codefall is installed there is the files the manifest records it wrote into it.
			name: "a file the manifest records for the harness is gone",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, claudeSkill)
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessesInstalled.ID: domain.StatusFail}),
			target:     domain.HarnessesInstalled.ID,
			wantDetail: "codefall is not installed for claude",
			wantRemedy: mo.Some("codefall init"),
		},
		{
			// A harness codefall was never run for has no entry at all, which is the same answer as a
			// harness whose files are gone: there is nothing of codefall's in the directory it reads.
			name: "one of two harnesses has nothing installed",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(twoHarnessSettings)
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessesInstalled.ID: domain.StatusFail}),
			target:     domain.HarnessesInstalled.ID,
			wantDetail: "codefall is not installed for codex",
			wantRemedy: mo.Some("codefall init"),
		},
		{
			name: "both harnesses are installed",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(twoHarnessSettings)
				f.files[manifestPath] = []byte(droppedManifest)
				f.files[agentsSkill] = []byte("---\nname: design\n---\n")
			},
			want:       allPass,
			target:     domain.HarnessesInstalled.ID,
			wantDetail: "claude, codex",
		},
		{
			name: "a recorded file cannot be stat'd",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.errs[claudeSkill] = errors.New("permission denied")
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessesInstalled.ID: domain.StatusFail}),
			target:     domain.HarnessesInstalled.ID,
			wantDetail: "Cannot stat " + claudeSkill + ": permission denied",
			wantRemedy: mo.None[string](),
		},
		{
			// A project that drops a harness keeps every file codefall wrote for it, because codefall
			// only ever writes what it owns and never deletes. The directory is still on disk here,
			// which is the state the warning is about.
			name: "an install is left over from a harness the settings dropped",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[manifestPath] = []byte(droppedManifest)
				f.files[agentsSkill] = []byte("---\nname: design\n---\n")
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessesLeftOver.ID: domain.StatusWarn}),
			target:     domain.HarnessesLeftOver.ID,
			wantDetail: "codefall is still installed for codex, which the settings no longer name",
			wantRemedy: mo.Some(leftoverRemedy),
		},
		{
			// No record means no install that could be left over, so the left-over check is absent.
			// It also means no install the other check can confirm: the record is the evidence, and a
			// project with none has nothing that says a run ever finished. Init writes the manifest it
			// could not read the next time it runs, which is why the remedy is the same one.
			name: "there is no manifest to hold the settings against",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, manifestPath)
			},
			want: outcomes(map[string]domain.Status{domain.HarnessesInstalled.ID: domain.StatusFail},
				domain.HarnessesLeftOver.ID),
			target:     domain.HarnessesInstalled.ID,
			wantDetail: "codefall is not installed for claude",
			wantRemedy: mo.Some("codefall init"),
		},
		{
			// A project set up before its harnesses were named for their binaries records the old
			// spelling in both files. codefall still reads it, so nothing else in the report changes:
			// the settings' claude-code is the manifest's claude-code, and both are claude.
			name: "the settings and the manifest record a former harness name",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(strings.Replace(validSettings, `["claude"]`, `["claude-code"]`, 1))
				f.files[manifestPath] = []byte(strings.Replace(installedManifest, `"claude":`, `"claude-code":`, 1))
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessNames.ID: domain.StatusWarn}),
			target:     domain.HarnessNames.ID,
			wantDetail: ".codefall/settings.json and .codefall/manifest.json name claude-code (now claude)",
			wantRemedy: mo.Some(renameRemedy),
		},
		{
			// The two files can disagree — the settings rewritten by hand, the manifest left alone — and
			// the install is still found under the name the settings use.
			name: "only the manifest records a former harness name",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[manifestPath] = []byte(strings.Replace(installedManifest, `"claude":`, `"claude-code":`, 1))
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessNames.ID: domain.StatusWarn}),
			target:     domain.HarnessNames.ID,
			wantDetail: ".codefall/manifest.json names claude-code (now claude)",
			wantRemedy: mo.Some(renameRemedy),
		},
		{
			name: "only the settings record former harness names",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(strings.Replace(validSettings, `["claude"]`,
					`["claude-code", "antigravity", "claude"]`, 1))
				f.files[manifestPath] = []byte(`{"harnesses": {
  "claude": {"version": "v1.2.3", "files": [".claude/skills/design/SKILL.md"]},
  "agy": {"version": "v1.2.3", "files": [".agents/skills/design/SKILL.md"]}
}}`)
				f.files[agentsSkill] = []byte("---\nname: design\n---\n")
			},
			want:       outcomes(map[string]domain.Status{domain.HarnessNames.ID: domain.StatusWarn}),
			target:     domain.HarnessNames.ID,
			wantDetail: ".codefall/settings.json names antigravity (now agy) and claude-code (now claude)",
			wantRemedy: mo.Some(renameRemedy),
		},
		{
			// Settings written before the list existed name no agents, which means the default: this
			// harness's own subagent, which is always runnable and is itself the current entry.
			name:       "no agents are declared, so the default applies",
			mutate:     func(*fakeFileSystem, *fakeCommandRunner) {},
			want:       allPass,
			target:     domain.AgentsRunnable.ID,
			wantDetail: "subagent (current)",
		},
		{
			name: "every agent runs on a harness this machine has",
			mutate: func(f *fakeFileSystem, r *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(twoAgents, "", ""))
				r.paths["codex"] = "/opt/homebrew/bin/codex"
			},
			want:       allPass,
			target:     domain.AgentsRunnable.ID,
			wantDetail: "architect (codex, gpt-5-codex), subagent (current)",
		},
		{
			// A run skips what it cannot start and moves on, so this is information and a warning,
			// not a failure.
			name: "an agent runs on a harness that is not on PATH",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(twoAgents, "", ""))
			},
			want:       outcomes(map[string]domain.Status{domain.AgentsRunnable.ID: domain.StatusWarn}),
			target:     domain.AgentsRunnable.ID,
			wantDetail: "architect runs on codex, which is not on PATH",
			wantRemedy: mo.Some(agentsRemedy),
		},
		{
			// Every order is checked: the list itself, review's own, and each harness's. Here the
			// list has current but neither of the narrower orders does.
			name: "the review and per-harness orders name no agent on current",
			mutate: func(f *fakeFileSystem, r *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(twoAgents, `["architect"]`, `{ "claude": ["architect"] }`))
				r.paths["codex"] = "/opt/homebrew/bin/codex"
			},
			want:       outcomes(map[string]domain.Status{domain.AgentsCurrent.ID: domain.StatusWarn}),
			target:     domain.AgentsCurrent.ID,
			wantDetail: "review.agents, agentsByHarness.claude names no agent on current",
			wantRemedy: mo.Some(currentRemedy),
		},
		{
			// The consult block's own order is checked like review's.
			name: "the consult order names no agent on current",
			mutate: func(f *fakeFileSystem, r *fakeCommandRunner) {
				document := withAgents(twoAgents, "", "")
				f.files[settingsPath] = []byte(strings.Replace(document, `"review":`, `"consult": { "agents": ["architect"] },
  "review":`, 1))
				r.paths["codex"] = "/opt/homebrew/bin/codex"
			},
			want:       outcomes(map[string]domain.Status{domain.AgentsCurrent.ID: domain.StatusWarn}),
			target:     domain.AgentsCurrent.ID,
			wantDetail: "consult.agents names no agent on current",
			wantRemedy: mo.Some(currentRemedy),
		},
		{
			name: "the top-level order itself has no agent on current",
			mutate: func(f *fakeFileSystem, r *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(`[{ "name": "architect", "harness": "codex" }]`, "", ""))
				r.paths["codex"] = "/opt/homebrew/bin/codex"
			},
			want:       outcomes(map[string]domain.Status{domain.AgentsCurrent.ID: domain.StatusWarn}),
			target:     domain.AgentsCurrent.ID,
			wantDetail: "agents names no agent on current",
			wantRemedy: mo.Some(currentRemedy),
		},
		{
			// What the list may hold is the format's to say, so a bad entry is the settings-complete
			// check's finding and the two agent checks are absent, like everything after it.
			name: "an agent names a harness codefall cannot start",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(`[{ "name": "helper", "harness": "cursor" }]`, "", ""))
			},
			want: outcomes(map[string]domain.Status{domain.SettingsComplete.ID: domain.StatusFail},
				afterSettingsDone...),
			target: domain.SettingsComplete.ID,
			wantDetail: `settings.json is incomplete: agents: [0].harness: unknown value "cursor" ` +
				`(expected "agy", "claude", "codex", "current", "muse", "opencode")`,
			wantRemedy: mo.Some(fixRemedy),
		},
		{
			name: "review's order names an agent the list does not define",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withAgents(twoAgents, `["reviewer"]`, ""))
			},
			want: outcomes(map[string]domain.Status{domain.SettingsComplete.ID: domain.StatusFail},
				afterSettingsDone...),
			target:     domain.SettingsComplete.ID,
			wantDetail: `settings.json is incomplete: review.agents: names "reviewer", which agents does not define`,
			wantRemedy: mo.Some(fixRemedy),
		},
		{
			// Every project set up before the local block existed looks like this. Nothing else
			// stops working, so it warns, and the remedy is the verb that fills the block in.
			name: "the local commands are not declared",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(undeclaredSettings)
			},
			want: outcomes(map[string]domain.Status{domain.LocalDeclared.ID: domain.StatusWarn},
				afterLocalUndeclared...),
			target:     domain.LocalDeclared.ID,
			wantDetail: "no local start and update commands are declared in settings.json",
			wantRemedy: mo.Some(equipRemedy),
		},
		{
			// Both commands name the same script, which is the usual shape, so it is reported once
			// with both fields beside it.
			name: "the declared script is not in the project",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.files, localScript)
			},
			want:       outcomes(map[string]domain.Status{domain.LocalRunnable.ID: domain.StatusFail}),
			target:     domain.LocalRunnable.ID,
			wantDetail: "scripts/local.sh is not in the project (local.start, local.update)",
			wantRemedy: mo.Some(equipRemedy),
		},
		{
			name: "the declared commands name programs on PATH",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withLocal("docker compose up -d", "make sync"))
			},
			want:   allPass,
			target: domain.LocalRunnable.ID,
		},
		{
			name: "a declared command names a program that is not on PATH",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withLocal("docker compose up -d", "npm run db:migrate"))
			},
			want:       outcomes(map[string]domain.Status{domain.LocalRunnable.ID: domain.StatusFail}),
			target:     domain.LocalRunnable.ID,
			wantDetail: "npm is not on PATH (local.update)",
			wantRemedy: mo.Some(equipRemedy),
		},
		{
			// A leading VAR=value sets the environment for the program after it; the program is the
			// next word.
			name: "a declared command sets a variable before the program",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withLocal("COMPOSE_PROFILES=dev docker compose up -d", "make sync"))
			},
			want:   allPass,
			target: domain.LocalRunnable.ID,
		},
		{
			name: "a declared command is only spaces",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withLocal("   ", "make sync"))
			},
			want:       outcomes(map[string]domain.Status{domain.LocalRunnable.ID: domain.StatusFail}),
			target:     domain.LocalRunnable.ID,
			wantDetail: "local.start runs nothing",
			wantRemedy: mo.Some(equipRemedy),
		},
		{
			name: "a declared command names an absolute path that exists",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withLocal("/opt/tools/up.sh", "make sync"))
				f.files["/opt/tools/up.sh"] = []byte("#!/bin/sh\n")
			},
			want:   allPass,
			target: domain.LocalRunnable.ID,
		},
		{
			name: "the declared script cannot be stat'd",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.errs[localScript] = errors.New("permission denied")
			},
			want:       outcomes(map[string]domain.Status{domain.LocalRunnable.ID: domain.StatusFail}),
			target:     domain.LocalRunnable.ID,
			wantDetail: "Cannot stat scripts/local.sh: permission denied",
			wantRemedy: mo.None[string](),
		},
		{
			// Every project set up before the test block existed looks like this. Nothing else stops
			// working, so it warns, and the remedy is the command that asks where the cases go.
			name: "the testing root is not declared",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withTest(""))
			},
			want: outcomes(map[string]domain.Status{domain.TestDeclared.ID: domain.StatusWarn},
				afterTestUndeclared...),
			target:     domain.TestDeclared.ID,
			wantDetail: "no testing root is declared in settings.json",
			wantRemedy: mo.Some(initRemedy),
		},
		{
			// A block Validate would reject is no declaration at all, and doctor's settings check has
			// already said what is wrong with it.
			name: "the declared root is not a relative path inside the project",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withTest(`{ "dir": "/srv/testing" }`))
			},
			want: outcomes(map[string]domain.Status{
				domain.SettingsComplete.ID: domain.StatusFail, domain.TestDeclared.ID: domain.StatusWarn},
				afterSettingsDone...),
			target:     domain.SettingsComplete.ID,
			wantDetail: `settings.json is incomplete: test.dir: must be a relative path inside the project, such as "testing"`,
			wantRemedy: mo.Some(fixRemedy),
		},
		{
			// A root with no runner is a project that has declared where its cases go and has not
			// installed anything to run them with, which is equip's work.
			name: "no test runner is declared",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withTest(`{ "dir": "testing" }`))
			},
			want:       outcomes(map[string]domain.Status{domain.TestEquipped.ID: domain.StatusWarn}),
			target:     domain.TestEquipped.ID,
			wantDetail: "no test runner is declared in settings.json",
			wantRemedy: mo.Some(equipTestRemedy),
		},
		{
			name: "the runner list is empty",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withTest(`{ "dir": "testing", "runners": [] }`))
			},
			want:       outcomes(map[string]domain.Status{domain.TestEquipped.ID: domain.StatusWarn}),
			target:     domain.TestEquipped.ID,
			wantDetail: "no test runner is declared in settings.json",
			wantRemedy: mo.Some(equipTestRemedy),
		},
		{
			name: "every declared runner is reported",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.files[settingsPath] = []byte(withTest(`{ "dir": "testing", "runners": ["playwright", "go-test"] }`))
			},
			want:       allPass,
			target:     domain.TestEquipped.ID,
			wantDetail: "playwright, go-test",
		},
		{
			// It fails rather than warns: the project declared the directory, and every verb that
			// writes a case or runs one looks for it.
			name: "the declared testing directory is not there",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				delete(f.dirs, testingDir)
			},
			want:       outcomes(map[string]domain.Status{domain.TestDirExists.ID: domain.StatusFail}),
			target:     domain.TestDirExists.ID,
			wantDetail: "testing/ is declared in settings.json and is not there",
			wantRemedy: mo.Some(initRemedy),
		},
		{
			name: "the declared testing directory cannot be stat'd",
			mutate: func(f *fakeFileSystem, _ *fakeCommandRunner) {
				f.errs[testingDir] = errors.New("permission denied")
			},
			want:       outcomes(map[string]domain.Status{domain.TestDirExists.ID: domain.StatusFail}),
			target:     domain.TestDirExists.ID,
			wantDetail: "Cannot stat testing: permission denied",
			wantRemedy: mo.None[string](),
		},
		{
			name: "bd is not installed",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				delete(r.paths, "bd")
			},
			want:       outcomes(map[string]domain.Status{domain.BeadsInstalled.ID: domain.StatusFail}, afterBeadsMissing...),
			target:     domain.BeadsInstalled.ID,
			wantDetail: "bd is not on PATH",
			wantRemedy: mo.Some("brew install beads"),
		},
		{
			name: "bd version fails, so the path is the detail",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.runs["bd version"] = CommandResult{ExitCode: 2, Stderr: "unknown command\n"}
			},
			want:       allPass,
			target:     domain.BeadsInstalled.ID,
			wantDetail: "/opt/homebrew/bin/bd",
			wantRemedy: mo.None[string](),
		},
		{
			name: "beads is not initialized here",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.runs["bd info"] = CommandResult{
					ExitCode: 1,
					Stderr:   "Error: no beads database found\nrun bd init\n",
				}
			},
			want:       outcomes(map[string]domain.Status{domain.BeadsInitialized.ID: domain.StatusWarn}),
			target:     domain.BeadsInitialized.ID,
			wantDetail: "This repository has no Beads database (bd info exited 1: Error: no beads database found)",
			wantRemedy: mo.Some("bd init"),
		},
		{
			name: "bd info cannot be started",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.errs["bd info"] = errors.New("run bd: fork/exec: resource temporarily unavailable")
			},
			want:       outcomes(map[string]domain.Status{domain.BeadsInitialized.ID: domain.StatusWarn}),
			target:     domain.BeadsInitialized.ID,
			wantDetail: "Could not run bd info: run bd: fork/exec: resource temporarily unavailable",
			wantRemedy: mo.Some("bd init"),
		},
		{
			name: "gh is not installed",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				delete(r.paths, "gh")
			},
			want:       outcomes(map[string]domain.Status{domain.GHInstalled.ID: domain.StatusFail}, afterGHMissing...),
			target:     domain.GHInstalled.ID,
			wantDetail: "gh is not on PATH",
			wantRemedy: mo.Some("brew install gh"),
		},
		{
			name: "gh auth status cannot be started",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.errs["gh auth status --json hosts"] = errors.New("run gh: signal: killed")
			},
			want:       outcomes(map[string]domain.Status{domain.GHAuthenticated.ID: domain.StatusFail}, afterGHAuth...),
			target:     domain.GHAuthenticated.ID,
			wantDetail: "Could not run gh auth status: run gh: signal: killed",
			wantRemedy: mo.None[string](),
		},
		{
			name: "gh auth status prints something that is not JSON",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.runs["gh auth status --json hosts"] = CommandResult{
					ExitCode: 1,
					Stderr:   "unknown flag: --json\n",
				}
			},
			want:       outcomes(map[string]domain.Status{domain.GHAuthenticated.ID: domain.StatusFail}, afterGHAuth...),
			target:     domain.GHAuthenticated.ID,
			wantDetail: "Unexpected output from gh auth status: unknown flag: --json",
			wantRemedy: mo.Some("gh auth login"),
		},
		{
			name: "no github.com account is active",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.runs["gh auth status --json hosts"] = CommandResult{
					Stdout: ghHosts(`{"state":"success","active":false,"login":"djensen47","scopes":"repo"}`),
				}
			},
			want:       outcomes(map[string]domain.Status{domain.GHAuthenticated.ID: domain.StatusFail}, afterGHAuth...),
			target:     domain.GHAuthenticated.ID,
			wantDetail: "No active github.com account",
			wantRemedy: mo.Some("gh auth login"),
		},
		{
			name: "the active account is in an error state",
			mutate: func(_ *fakeFileSystem, r *fakeCommandRunner) {
				r.runs["gh auth status --json hosts"] = CommandResult{
					Stdout: ghHosts(`{"state":"error","active":true,"login":"djensen47","scopes":"repo","error":"token expired"}`),
				}
			},
			want:       outcomes(map[string]domain.Status{domain.GHAuthenticated.ID: domain.StatusFail}, afterGHAuth...),
			target:     domain.GHAuthenticated.ID,
			wantDetail: `djensen47: state "error" token expired`,
			wantRemedy: mo.Some("gh auth login"),
		},
		{
			name:       "the token is missing repo",
			mutate:     withScopes("gist, project, read:org"),
			want:       outcomes(map[string]domain.Status{domain.GHScopes.ID: domain.StatusFail}),
			target:     domain.GHScopes.ID,
			wantDetail: "The token is missing repo",
			wantRemedy: mo.Some("gh auth refresh -s repo"),
		},
		{
			name:   "project satisfies read:project",
			mutate: withScopes("repo, project"),
			want:   allPass,
			target: domain.GHScopes.ID,
		},
		{
			name:       "the token has neither project scope",
			mutate:     withScopes("repo, workflow"),
			want:       outcomes(map[string]domain.Status{domain.GHScopes.ID: domain.StatusWarn}),
			target:     domain.GHScopes.ID,
			wantDetail: "The token is missing read:project, project",
			wantRemedy: mo.Some("gh auth refresh -s read:project -s project"),
		},
		{
			name:       "the token has none of the three scopes",
			mutate:     withScopes("gist"),
			want:       outcomes(map[string]domain.Status{domain.GHScopes.ID: domain.StatusFail}),
			target:     domain.GHScopes.ID,
			wantDetail: "The token is missing repo, read:project, project",
			wantRemedy: mo.Some("gh auth refresh -s repo -s read:project -s project"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files, runner := healthy()
			tc.mutate(files, runner)

			report, err := NewDiagnose(files, runner).Run(t.Context(), workingDir)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			assertOutcomes(t, report, tc.want)

			if tc.wantDetail != "" || tc.wantRemedy.IsPresent() {
				assertResult(t, report, tc.target, tc.wantDetail, tc.wantRemedy)
			}
		})
	}
}

// withScopes rewrites the fixture's gh auth status output with a different scope string.
func withScopes(scopes string) func(*fakeFileSystem, *fakeCommandRunner) {
	return func(_ *fakeFileSystem, r *fakeCommandRunner) {
		r.runs["gh auth status --json hosts"] = CommandResult{
			Stdout: ghHosts(fmt.Sprintf(
				`{"state":"success","active":true,"login":"djensen47","scopes":%q}`, scopes)),
		}
	}
}

func assertOutcomes(t *testing.T, report domain.Report, want []outcome) {
	t.Helper()

	got := make([]outcome, 0, len(want))
	for _, result := range report.Results() {
		got = append(got, outcome{result.Check.ID, result.Status})
	}

	if !slices.Equal(got, want) {
		t.Errorf("outcomes =\n  %v\nwant\n  %v", got, want)
	}
}

func assertResult(t *testing.T, report domain.Report, id, wantDetail string, wantRemedy mo.Option[string]) {
	t.Helper()

	for _, result := range report.Results() {
		if result.Check.ID != id {
			continue
		}

		if got := result.Detail.OrElse(""); got != wantDetail {
			t.Errorf("%s detail = %q, want %q", id, got, wantDetail)
		}

		if got := result.Remedy; got != wantRemedy {
			t.Errorf("%s remedy = %v, want %v", id, got, wantRemedy)
		}

		return
	}

	t.Errorf("check %q is not in the report", id)
}

// --- the calls the use case makes ----------------------------------------------------------------

func TestDiagnoseRunsToolsInTheWorkingDirectory(t *testing.T) {
	files, runner := healthy()

	if _, err := NewDiagnose(files, runner).Run(t.Context(), workingDir); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []runCall{
		{workingDir, "bd version"},
		{workingDir, "bd info"},
		{workingDir, "gh --version"},
		{workingDir, "gh auth status --json hosts"},
	}

	if !slices.Equal(runner.calls, want) {
		t.Errorf("calls =\n  %v\nwant\n  %v", runner.calls, want)
	}
}

func TestDiagnoseReturnsAnErrorForACancelledContext(t *testing.T) {
	files, runner := healthy()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := NewDiagnose(files, runner).Run(ctx, workingDir)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
}
