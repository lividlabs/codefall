package initcmd_test

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/samber/do/v2"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd"
	"github.com/lividlabs/codefall-cli/extensions"
)

// A provider whose static return type is the concrete type compiles and then fails at runtime with
// "could not find service". Resolving the whole graph through a real injector is the only thing that
// catches that, so this test does exactly what the composition root does.
func TestRegisterProvidesEverythingCommandNeeds(t *testing.T) {
	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	initcmd.Register(injector)

	cmd := initcmd.Command(injector)
	if cmd == nil {
		t.Fatal("Command returned nil")
	}

	// The package is initcmd; the command is init.
	if got := cmd.Name(); got != "init" {
		t.Errorf("Command().Name() = %q, want %q", got, "init")
	}
}

// installDir is where the extension step copies the shared scripts, so it is where every hook
// definition has to reach for the guard. It is one directory for every harness now, which is what
// the install layout decided (ADR-006).
const installDir = ".codefall"

// The hook step's table and the extension step's destinations agree with the embedded tree by
// three hand-maintained consistencies: every definition exists, every script a definition's
// commands point at exists under hooks/shared/, and the path they name sits under the directory
// the extension step copies that directory into. The application layer's tests run over a fake
// copy of the definitions, so nothing there would fail on a rename — this test runs over the
// real tree and does. It states the table again the way the schema test states the format: a
// pin, not a second source of truth.
func TestHookDefinitionsPointAtScriptsThatLand(t *testing.T) {
	tree := extensions.Files()

	for _, tc := range []struct {
		harness string
		source  string
		destDir string
	}{
		{"claude", "hooks/claude/hooks.json", installDir},
		{"codex", "hooks/codex/hooks.json", installDir},
		{"agy", "hooks/agy/hooks.json", installDir},
	} {
		t.Run(tc.harness, func(t *testing.T) {
			body, err := fs.ReadFile(tree, tc.source)
			if err != nil {
				t.Fatalf("read %s from the embedded tree: %v", tc.source, err)
			}

			var document any
			if err := json.Unmarshal(body, &document); err != nil {
				t.Fatalf("decode %s: %v", tc.source, err)
			}

			commands := commandsIn(document)
			if len(commands) == 0 {
				t.Fatalf("%s declares no commands", tc.source)
			}

			for _, command := range commands {
				at := strings.Index(command, "hooks/shared/")
				if at < 0 {
					continue
				}

				if prefix := command[:at]; !strings.HasSuffix(prefix, tc.destDir+"/") {
					t.Errorf("command %q points outside %s/, where the extension step copies the shared scripts", command, tc.destDir)
				}

				name := scriptName(command[at:])
				if _, err := fs.Stat(tree, "hooks/shared/"+name); err != nil {
					t.Errorf("hooks/shared/%s: %v — a renamed script would leave the %s guard pointing at nothing", name, err, tc.harness)
				}
			}
		})
	}
}

// The OpenCode plugin is the one definition that is not JSON, so its pin is textual: the guard and
// the session-start notice it delegates to are the same shared scripts the others name.
func TestOpenCodePluginPointsAtTheSharedScripts(t *testing.T) {
	tree := extensions.Files()

	body, err := fs.ReadFile(tree, "hooks/opencode/codefall.js")
	if err != nil {
		t.Fatalf("read hooks/opencode/codefall.js from the embedded tree: %v", err)
	}

	for _, script := range []string{"codefall-block-merge-to-main.sh", "codefall-session-notice.sh"} {
		// The plugin lands at <install>/.opencode/plugins/codefall.js and reaches the scripts from
		// its own directory, so two levels up is the install directory and .codefall/ is below it.
		referenced := "/../../" + installDir + "/hooks/shared/" + script
		if !strings.Contains(string(body), referenced) {
			t.Errorf("the plugin does not name %s — a renamed script would leave it pointing at nothing",
				referenced)
		}

		if _, err := fs.Stat(tree, "hooks/shared/"+script); err != nil {
			t.Errorf("hooks/shared/%s: %v", script, err)
		}
	}
}

// What the JSON harnesses register on SessionStart: the Beads prime, and the notice as a second
// command, so an upgrade recognises the notice by its own script name and replaces that entry alone.
// Antigravity has no session event, and its definition says nothing about one.
func TestSessionStartRegistersTheNoticeBesideThePrime(t *testing.T) {
	tree := extensions.Files()

	for _, source := range []string{"hooks/claude/hooks.json", "hooks/codex/hooks.json"} {
		t.Run(source, func(t *testing.T) {
			body, err := fs.ReadFile(tree, source)
			if err != nil {
				t.Fatalf("read %s from the embedded tree: %v", source, err)
			}

			var document struct {
				Hooks map[string]any `json:"hooks"`
			}

			if err := json.Unmarshal(body, &document); err != nil {
				t.Fatalf("decode %s: %v", source, err)
			}

			commands := commandsIn(document.Hooks["SessionStart"])

			if !slices.Contains(commands, "bd prime --hook-json") {
				t.Errorf("SessionStart = %q, want the Beads prime among them", commands)
			}

			notices := 0
			for _, command := range commands {
				if strings.Contains(command, "codefall-session-notice.sh") {
					notices++

					if !strings.Contains(command, "--hook-json") {
						t.Errorf("%q does not select the JSON contract the event reads", command)
					}
				}
			}

			if notices != 1 {
				t.Errorf("SessionStart = %q, want exactly one command running the notice", commands)
			}
		})
	}

	body, err := fs.ReadFile(tree, "hooks/agy/hooks.json")
	if err != nil {
		t.Fatalf("read hooks/agy/hooks.json from the embedded tree: %v", err)
	}

	if strings.Contains(string(body), "codefall-session-notice.sh") {
		t.Error("the Antigravity definition registers the notice, and that harness has no session event")
	}
}

// The extension step copies exactly three subtrees: skills/ into each chosen harness's skills
// directory, and hooks/shared/ and shared/ into .codefall/. A tree that stopped shipping one of them
// would fail every install rather than one check, so all three are pinned against the real tree.
func TestTheTreeShipsEverySubtreeAnInstallCopies(t *testing.T) {
	for _, source := range []string{"skills", "hooks/shared", "shared"} {
		if _, err := fs.Stat(extensions.Files(), source); err != nil {
			t.Errorf("%s: %v — the extension step copies this subtree into every project", source, err)
		}
	}
}

// The maintainer documents stay embedded and are left behind at copy time, so this pins that they are
// still in the tree the step reads: a document that went missing would make the exclusion silently
// stop excluding anything.
func TestTheTreeStillHoldsTheDocumentsAnInstallLeavesBehind(t *testing.T) {
	tree := extensions.Files()

	for _, document := range []string{"AGENTS.md", "README.md", "docs/ROADMAP.md", "skills/AGENTS.md"} {
		if _, err := fs.Stat(tree, document); err != nil {
			t.Errorf("%s: %v — a maintainer document no install writes into a project", document, err)
		}
	}

	notes := 0

	if err := fs.WalkDir(tree, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && d.Name() == "NOTES.md" {
			notes++
		}

		return nil
	}); err != nil {
		t.Fatalf("walk skills/: %v", err)
	}

	if notes == 0 {
		t.Error("no skill ships a NOTES.md — the rule that leaves them behind has nothing to leave")
	}
}

// Every path a skill uses to reach a shared file is the one the install layout settled:
// ../../../.codefall/shared/<file> from <skills dir>/skills/<verb>/, and one level deeper from a
// supporting file. The file each of them names has to be in shared/, which is what lands there.
func TestSkillsReachSharedFilesByTheInstalledPath(t *testing.T) {
	tree := extensions.Files()
	reference := regexp.MustCompile(`(?:\.\./)+` + regexp.QuoteMeta(installDir) + `/shared/([A-Za-z0-9._-]+)`)
	found := 0

	if err := fs.WalkDir(tree, "skills", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		body, err := fs.ReadFile(tree, path)
		if err != nil {
			return err
		}

		for _, match := range reference.FindAllStringSubmatch(string(body), -1) {
			found++

			if _, err := fs.Stat(tree, "shared/"+match[1]); err != nil {
				t.Errorf("%s names %s, which is not in shared/: %v", path, match[0], err)
			}
		}

		// The path the layout replaced. A skill still using it would read a file no install writes.
		if strings.Contains(string(body), "../../shared/") {
			t.Errorf("%s still names ../../shared/, which no install writes any more", path)
		}

		return nil
	}); err != nil {
		t.Fatalf("walk skills/: %v", err)
	}

	if found == 0 {
		t.Error("no skill names a shared file — the path this pins is not being exercised")
	}
}

// The agents step reads its four sections from agents/sections/ and the testing step its two
// skeletons from agents/testing/, and the extension step copies none of it. The application layer's
// tests run over stand-ins, so this is the pin against the real words: each section opens and
// closes with the markers the domain names, the Testing section holds the placeholder the step
// fills and names the verbs a reader needs from it, the Codefall section names the installed
// detail file, and each skeleton is a whole document. It states the table again, the way the hook
// test does: a pin, not a second source of truth.
func TestTheTreeShipsTheSectionsInitWrites(t *testing.T) {
	tree := extensions.Files()

	for _, tc := range []struct {
		source string
		begin  string
		end    string
		names  []string
	}{
		{
			source: "agents/sections/codefall.md",
			begin:  "<!-- BEGIN CODEFALL PROCESS -->",
			end:    "<!-- END CODEFALL PROCESS -->",
			names:  []string{installDir + "/shared/workflow.md"},
		},
		{
			source: "agents/sections/beads.md",
			begin:  "<!-- BEGIN CODEFALL BEADS -->",
			end:    "<!-- END CODEFALL BEADS -->",
		},
		{
			source: "agents/sections/local.md",
			begin:  "<!-- BEGIN CODEFALL LOCAL -->",
			end:    "<!-- END CODEFALL LOCAL -->",
			names:  []string{"/codefall-refresh", "/codefall-equip"},
		},
		{
			source: "agents/sections/testing.md",
			begin:  "<!-- BEGIN CODEFALL TESTING -->",
			end:    "<!-- END CODEFALL TESTING -->",
			names:  []string{"{{TESTING_ROOT}}", "/codefall-implement", "/codefall-test", "/codefall-equip"},
		},
	} {
		t.Run(tc.source, func(t *testing.T) {
			data, err := fs.ReadFile(tree, tc.source)
			if err != nil {
				t.Fatalf("read %s from the embedded tree: %v — the agents step writes this section", tc.source, err)
			}

			body := string(data)

			if !strings.HasPrefix(body, tc.begin+"\n") {
				t.Errorf("%s does not open with %s", tc.source, tc.begin)
			}

			if !strings.HasSuffix(body, "\n"+tc.end+"\n") {
				t.Errorf("%s does not close with %s and a newline", tc.source, tc.end)
			}

			if strings.Count(body, tc.begin) != 1 || strings.Count(body, tc.end) != 1 {
				t.Errorf("%s holds a marker more than once — a later run would replace the wrong span", tc.source)
			}

			if strings.Contains(body, "../../shared/") || strings.Contains(body, "../../../"+installDir) {
				t.Errorf("%s names a file by a skill's relative path, and it is read from the project root", tc.source)
			}

			for _, name := range tc.names {
				if !strings.Contains(body, name) {
					t.Errorf("%s does not name %s", tc.source, name)
				}
			}
		})
	}

	for _, tc := range []struct {
		source   string
		headings []string
	}{
		{
			source: "agents/testing/AGENTS.md",
			headings: []string{
				"## Runners", "## Setup and state-forcing commands", "## Real side effects", "## Environment notes",
			},
		},
		{
			source:   "agents/testing/README.md",
			headings: []string{"| Case ID | Modalities | Variants | Notes |"},
		},
	} {
		t.Run(tc.source, func(t *testing.T) {
			data, err := fs.ReadFile(tree, tc.source)
			if err != nil {
				t.Fatalf("read %s from the embedded tree: %v — the testing step writes this skeleton", tc.source, err)
			}

			body := string(data)

			if !strings.HasPrefix(body, "# ") || !strings.HasSuffix(body, "\n") {
				t.Errorf("%s is not a whole document:\n%s", tc.source, body)
			}

			for _, heading := range tc.headings {
				if !strings.Contains(body, heading) {
					t.Errorf("%s has no %q:\n%s", tc.source, heading, body)
				}
			}
		})
	}
}

// commandsIn collects every command string in a decoded hook document, whatever depth the
// harness's format keeps it at.
func commandsIn(value any) []string {
	switch shaped := value.(type) {
	case map[string]any:
		var commands []string
		if command, ok := shaped["command"].(string); ok {
			commands = append(commands, command)
		}
		for _, nested := range shaped {
			commands = append(commands, commandsIn(nested)...)
		}
		return commands

	case []any:
		var commands []string
		for _, nested := range shaped {
			commands = append(commands, commandsIn(nested)...)
		}
		return commands
	}

	return nil
}

// scriptName takes the file name off the front of what follows "hooks/shared/", stopping at the
// first space or closing quote.
func scriptName(rest string) string {
	if cut := strings.IndexAny(rest, " \""); cut >= 0 {
		return rest[len("hooks/shared/"):cut]
	}
	return strings.TrimSuffix(rest[len("hooks/shared/"):], "\n")
}
