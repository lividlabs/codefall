package main

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/lividlabs/codefall/extensions"
)

// The installed text — the skills, the shared files, the `AGENTS.md` sections, and the hooks — names
// commands, and a name the command tree does not hold sends a reader to a command that refuses. This
// test holds the text to the tree from both sides at once: the tree is the one `build` returns, and
// the text is `extensions.Files()`, the embedded copy init installs, so neither is a list kept by
// hand. It lives in the composition root because that is the one package that can see both:
// `extensions/` cannot import `cli/internal/`, where the tree is built.
//
// A text names a command when the word `codefall` is followed by words that name commands somewhere
// in the tree: `codefall project init`, `codefall doctor`, and `codefall config agents` name
// commands, and each has to resolve from the root. A word after `codefall` that names no command
// anywhere — prose like `codefall only ever writes what it owns` — names nothing, and so does
// `codefall-upgrade`, which is the skill and not the command. A word that names a command somewhere
// but not where the sentence puts it fails the test, which is what a moved command wants: no reader
// left pointing at a name that refuses. The walk reads what init installs, minus what it leaves
// behind — `skills/AGENTS.md` and each skill's `NOTES.md` — and minus `lineage.md`, the record of
// former names.
func TestTheInstalledTextNamesCommandsTheTreeHolds(t *testing.T) {
	_, root := build()

	names := map[string]bool{}
	collectCommandNames(root, names)

	var stale []string

	for _, dir := range []string{"skills", "shared", "agents", "hooks"} {
		err := fs.WalkDir(extensions.Files(), dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !installed(path) {
				return nil
			}

			content, err := fs.ReadFile(extensions.Files(), path)
			if err != nil {
				return err
			}

			for _, mention := range staleMentions(string(content), root, names) {
				stale = append(stale, path+": "+mention)
			}

			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", dir, err)
		}
	}

	if len(stale) > 0 {
		t.Errorf("the installed text names commands the tree does not hold:\n%s",
			strings.Join(stale, "\n"))
	}
}

// The scanner's own contract, on text written for it: a name the tree does not hold fails, the
// names it holds pass however the sentence continues, and prose after `codefall` passes.
func TestStaleMentions(t *testing.T) {
	_, root := build()

	names := map[string]bool{}
	collectCommandNames(root, names)

	for _, tc := range []struct {
		text string
		want int
	}{
		{"run codefall init in the project", 1},
		{"run codefall project init in the project", 0},
		{"codefall doctor reports and never repairs", 0},
		{"codefall project doctor reports and never repairs", 0},
		{"codefall config agents lists the readers", 0},
		{"codefall project config agents lists the readers", 0},
		{"printf 'y\\n' | ACCESSIBLE=1 codefall project upgrade --location here", 0},
		{"codefall only ever writes what it owns", 0},
		{"/codefall-upgrade offers codefall project upgrade first", 0},
		{"the codefall project's settings file", 0},
	} {
		if got := staleMentions(tc.text, root, names); len(got) != tc.want {
			t.Errorf("%q: stale mentions = %v, want %d", tc.text, got, tc.want)
		}
	}
}

// installed reports whether init copies or writes the file: everything under the four roots the
// extension ships from, minus the maintainer docs init leaves behind and the lineage record, which
// holds former names as history.
func installed(path string) bool {
	base := path[strings.LastIndexByte(path, '/')+1:]

	return path != "skills/AGENTS.md" && base != "NOTES.md" && base != "lineage.md"
}

// collectCommandNames gathers every command's name in the tree, the hidden ones included: those
// are the words that mean a command somewhere, which is how a stale name is told apart from prose.
func collectCommandNames(cmd *cobra.Command, names map[string]bool) {
	names[cmd.Name()] = true

	for _, child := range cmd.Commands() {
		collectCommandNames(child, names)
	}
}

// staleMentions returns every `codefall …` in the text that the tree does not hold: a command word
// that resolves under no command before it.
func staleMentions(text string, root *cobra.Command, names map[string]bool) []string {
	var stale []string

	tokens := strings.Fields(text)

	for i, token := range tokens {
		if word(token) != "codefall" {
			continue
		}

		cmd := root
		named := "codefall"

		for _, next := range tokens[i+1:] {
			name := commandWord(next)
			if !names[name] {
				break // an argument or a word of prose, not a command
			}

			if child := childNamed(cmd, name); child == nil {
				stale = append(stale, named+" "+name)

				break
			} else {
				cmd = child
				named = named + " " + name
			}
		}
	}

	return stale
}

// childNamed finds a command's visible child by name. A hidden command does not hold its name: the
// root's refusals exist to name the commands under project, and text that still names one is what
// the test is for.
func childNamed(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if !child.Hidden && child.Name() == name {
			return child
		}
	}

	return nil
}

// surrounding is the punctuation that can sit against a word in prose, a code block, or a shell
// script. A hyphen is not in it: `codefall-upgrade` is the skill, and reading it as `codefall`
// followed by `upgrade` would name a command nobody named.
const surrounding = "`*_.,;:!?()[]{}<>\"'“”‘’—"

// word is a token with the punctuation that surrounds it stripped.
func word(token string) string {
	return strings.Trim(token, surrounding)
}

// commandWord is a word with a possessive's 's stripped too, so `init's` names init. The tool's own
// name is never read this way, so `codefall's` opens no command path.
func commandWord(token string) string {
	token = word(token)

	for _, suffix := range []string{"'s", "’s"} {
		token = strings.TrimSuffix(token, suffix)
	}

	return token
}
