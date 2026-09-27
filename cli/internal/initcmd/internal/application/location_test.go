package application

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/samber/mo"
)

const (
	gitRootAndPrefix = "git rev-parse --show-toplevel --show-prefix"
	gitPrefix        = "git rev-parse --show-prefix"
)

// The root is offered only when there is a choice to make: below it. At the root, outside a work
// tree, and without git, there is nothing to ask.
func TestRepositoryRoot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result CommandResult
		noGit  bool
		want   mo.Option[string]
	}{
		{name: "a subdirectory", result: CommandResult{Stdout: "/repo\napps/web/\n"}, want: mo.Some("/repo")},
		{name: "the root", result: CommandResult{Stdout: "/repo\n\n"}, want: mo.None[string]()},
		{name: "outside a work tree", result: CommandResult{ExitCode: 128,
			Stderr: "fatal: not a git repository\n"}, want: mo.None[string]()},
		{name: "no git", noGit: true, want: mo.None[string]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := toolsInstalled()
			runner.runs[gitRootAndPrefix] = tc.result

			if tc.noGit {
				delete(runner.paths, "git")
			}

			got := NewInitialize(newFakeFileSystem(), runner, newFakeExtensionSource(), noChanges()).
				RepositoryRoot(t.Context(), workingDir)
			if got != tc.want {
				t.Errorf("RepositoryRoot = %v, want %v", got, tc.want)
			}
		})
	}
}

// rootedClaudeDefinition is the Claude definition as the embedded tree ships it: the guard and the
// session-start notice named from the repository root, the prime a bare command.
var rootedClaudeDefinition = []byte(`{
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command",
      "command": "\"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-block-merge-to-main.sh\""}]}],
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "bd prime --hook-json"}]},
      {"matcher": "", "hooks": [{"type": "command",
        "command": "\"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-session-notice.sh\" --hook-json"}]}
    ]
  }
}`)

// A subdirectory install copies the scripts below the root, so every command naming one has to name
// it there — the guard and the notice alike. A command that does not name the root is left as it
// is, and at the root nothing changes at all.
func TestHookWritesTheSubdirectoryIntoCommandsThatNameTheRoot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix string
		guard  string
		notice string
	}{
		{name: "a subdirectory", prefix: "apps/web/",
			guard:  `"$(git rev-parse --show-toplevel)/apps/web/.codefall/hooks/shared/codefall-block-merge-to-main.sh"`,
			notice: `"$(git rev-parse --show-toplevel)/apps/web/.codefall/hooks/shared/codefall-session-notice.sh" --hook-json`},
		{name: "the root", prefix: "",
			guard:  `"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-block-merge-to-main.sh"`,
			notice: `"$(git rev-parse --show-toplevel)/.codefall/hooks/shared/codefall-session-notice.sh" --hook-json`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("")
			runner := toolsInstalled()
			runner.runs[gitPrefix] = CommandResult{Stdout: tc.prefix + "\n"}

			source := newFakeExtensionSource()
			source.data["hooks/claude/hooks.json"] = rootedClaudeDefinition

			if _, err := NewInitialize(files, runner, source, noChanges()).Run(t.Context(), beadsRequest(), nil); err != nil {
				t.Fatalf("Run: %v", err)
			}

			var written map[string]any
			if err := json.Unmarshal(files.files[claudeFull], &written); err != nil {
				t.Fatalf("decode %s: %v", claudeFull, err)
			}

			commands := strings.Join(commandsOf(written), "\n")
			if !strings.Contains(commands, tc.guard) {
				t.Errorf("commands =\n%s\nwant the guard as %s", commands, tc.guard)
			}

			if !strings.Contains(commands, tc.notice) {
				t.Errorf("commands =\n%s\nwant the notice as %s", commands, tc.notice)
			}

			if !strings.Contains(commands, "bd prime --hook-json") {
				t.Errorf("commands =\n%s\nwant the prime left as it is", commands)
			}
		})
	}
}

// A directory name that means something inside double quotes would change what the guard runs, so
// the run refuses it before the first step rather than stopping halfway through.
func TestRunRefusesASubdirectoryItCannotQuote(t *testing.T) {
	files := newFakeFileSystem()
	runner := toolsInstalled()
	runner.runs[gitPrefix] = CommandResult{Stdout: "apps/$(rm -rf ~)/\n"}
	observer := &recordingObserver{}

	_, err := NewInitialize(files, runner, newFakeExtensionSource(), noChanges()).Run(t.Context(), beadsRequest(), observer)
	if err == nil || !strings.Contains(err.Error(), "cannot quote") {
		t.Fatalf("Run error = %v, want the subdirectory refused", err)
	}

	if len(observer.started) != 0 || len(files.files) != 0 {
		t.Errorf("started %v and wrote %v, want nothing begun", observer.started, keysOf(files))
	}
}

// commandsOf is every command string in a decoded document, whatever depth its format keeps it at.
func commandsOf(value any) []string {
	var commands []string

	switch shaped := value.(type) {
	case map[string]any:
		for key, child := range shaped {
			if command, ok := child.(string); ok && key == commandKey {
				commands = append(commands, command)
				continue
			}

			commands = append(commands, commandsOf(child)...)
		}
	case []any:
		for _, child := range shaped {
			commands = append(commands, commandsOf(child)...)
		}
	}

	return commands
}
