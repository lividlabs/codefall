// Package harness is the one definition of the coding harnesses codefall can set up: their names,
// and where each one reads the skills the extension installs. Both components need it — initcmd
// installs for the harnesses a project chose, and doctor reports on what it finds there — so it
// lives here rather than in either one's domain (ADR-003).
//
// A harness is named for its binary, so a script that runs one needs no mapping from the name a
// project's settings record to the command it starts. Two harnesses were named for their products
// before that rule: the spellings they had are still read, and Parse returns the name each one has
// now.
//
// It is a pure shared module: it imports the standard library and samber/mo, which is what lets
// domain/ and application/ name it. Adding a dependency here breaks that permission and fails the
// pure-shared-modules rule in .golangci.yml.
//
// What belongs here is what a harness is. How a hook definition reaches one — which file in the
// embedded tree it comes from, and whether that file is merged or copied — stays initcmd's, because
// registering it is what init does.
package harness

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/samber/mo"
)

// The harnesses codefall can set up, each named for the binary that runs it.
const (
	Agy      = "agy"
	Claude   = "claude"
	Codex    = "codex"
	Muse     = "muse"
	OpenCode = "opencode"
)

// formerNames is every spelling a harness had before it was named for its binary, and the name it
// has now. A project's checked-in settings and manifest may still carry one, so Parse accepts them,
// and codefall init rewrites them on its next run.
var formerNames = map[string]string{
	"antigravity": Agy,
	"claude-code": Claude,
}

// The two skills directories, relative to the directory init installs in.
const (
	claudeDir = ".claude"
	agentsDir = ".agents"
)

// skillsDirs is where each harness reads the skills the extension step installs. Claude Code reads
// its own .claude/; a harness that follows the .agents/skills convention takes .agents/, and adding
// one of those is one row.
//
// The map is also the roster: a harness codefall can set up is one that has somewhere to install, so
// All and Parse read their answer from these keys rather than from a second list that could disagree
// with them.
var skillsDirs = map[string]string{
	Agy:      agentsDir,
	Claude:   claudeDir,
	Codex:    agentsDir,
	Muse:     agentsDir,
	OpenCode: agentsDir,
}

// All returns the harness names codefall can set up, sorted. Each call builds its own slice, so a
// caller that sorts or truncates what it gets back changes nothing for the next one.
func All() []string {
	return slices.Sorted(maps.Keys(skillsDirs))
}

// Parse returns the harness name when codefall can set it up, and an error naming the ones it can
// when it cannot. A former spelling is accepted and returns the name the harness has now, so a
// project whose files still carry one keeps working. A harness codefall does not support yet is not
// a typo, so the message says so rather than calling the value unknown.
func Parse(name string) (string, error) {
	if _, known := skillsDirs[name]; known {
		return name, nil
	}

	if current, former := formerNames[name]; former {
		return current, nil
	}

	return "", fmt.Errorf("harness %q is not supported yet (supported: %s)", name, strings.Join(All(), ", "))
}

// Renamed returns the name a harness has now when name is a spelling it had before, and None when
// it is not one: a current name, or a name codefall has never used. Callers that need the current
// name either way take Renamed(name).OrElse(name).
func Renamed(name string) mo.Option[string] {
	current, former := formerNames[name]
	if !former {
		return mo.None[string]()
	}

	return mo.Some(current)
}

// SkillsDir returns where the named harness reads skills, relative to the directory init installs
// in, or None when codefall cannot set that harness up. There is nothing useful to say about a name
// Parse has already refused, so this is an Option and not an error (ADR-GO-03).
func SkillsDir(name string) mo.Option[string] {
	dir, known := skillsDirs[name]
	if !known {
		return mo.None[string]()
	}

	return mo.Some(dir)
}
