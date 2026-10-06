package domain

import (
	"fmt"
	"slices"
	"strings"

	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// sourcePrefix is the prefix every skill carries in the embedded tree. The source has one spelling and
// the install rewrites it; a tree that named a skill any other way would be a binary shipped wrong.
const sourcePrefix = settings.SkillPrefixCodefall

// SkillRename is the rewrite an install applies to the embedded tree so the skills land under the
// prefix the project chose (ADR-015). It rewrites whole skill names, `<prefix>-<verb>`, wherever they
// appear — a directory name, a frontmatter name, a slash command, a relative path into another
// skill, a sentence — and nothing else, because the names it knows are the skill directories the tree
// ships and a match is bounded on both sides.
//
// It recognises a skill under any prefix a project may choose, not only the source's. That is what
// lets the cleanup step tell `codefall-design/` removed and `cf-design/` written apart from a skill
// that went away, and what makes a project's move back to the default a rename like any other.
//
// The zero value renames nothing, which is what a test that is not about the prefix passes.
type SkillRename struct {
	prefix string
	// verbs is every skill's name without its prefix, longest first, so a verb that is a prefix of
	// another verb cannot win a match the longer one should have.
	verbs []string
}

// NewSkillRename builds the rewrite onto prefix for the skills the tree ships, each named as the tree
// names it, `codefall-<verb>`. A prefix outside the closed set and a skill named any other way are
// each an error, because the first is a settings file the validator would refuse and the second is a
// tree the authoring rule in extensions/skills/AGENTS.md forbids.
func NewSkillRename(prefix string, skills []string) (SkillRename, error) {
	chosen, err := settings.ParseSkillPrefix(prefix)
	if err != nil {
		return SkillRename{}, err
	}

	verbs := make([]string, 0, len(skills))

	for _, skill := range skills {
		verb, prefixed := strings.CutPrefix(skill, sourcePrefix+"-")
		if !prefixed || verb == "" {
			return SkillRename{}, fmt.Errorf("skill %q is not named %s-<verb>", skill, sourcePrefix)
		}

		verbs = append(verbs, verb)
	}

	slices.SortFunc(verbs, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}

		return strings.Compare(a, b)
	})

	return SkillRename{prefix: chosen, verbs: slices.Compact(verbs)}, nil
}

// Prefix is the prefix the rewrite renames onto, or "" for the zero value.
func (r SkillRename) Prefix() string {
	return r.prefix
}

// Text returns s with every whole skill name renamed onto the prefix. A skill name is one of the
// prefixes a project may choose, a hyphen, and a verb the tree ships, with no letter, digit,
// underscore, or hyphen on either side of it; `codefall-design` is renamed, `codefall-designs`,
// `my-codefall-design`, and `codefall-lineage` are not. A path is text like any other, because a
// slash is a boundary, so the same call renames `skills/codefall-design/SKILL.md`.
func (r SkillRename) Text(s string) string {
	if len(r.verbs) == 0 {
		return s
	}

	var out strings.Builder

	for at := 0; at < len(s); {
		name, verb, ok := r.nameAt(s, at)
		if !ok {
			out.WriteByte(s[at])
			at++

			continue
		}

		out.WriteString(r.prefix + "-" + verb)
		at += len(name)
	}

	return out.String()
}

// Names returns every whole skill name in s, under any prefix a project may choose, in the order
// they appear: what a reader that wants to know what a file calls the skills, rather than to rename
// them, asks for.
func (r SkillRename) Names(s string) []string {
	var names []string

	for at := 0; at < len(s); {
		name, _, ok := r.nameAt(s, at)
		if !ok {
			at++

			continue
		}

		names = append(names, name)
		at += len(name)
	}

	return names
}

// nameAt reports the skill name that begins at offset at in s, and its verb, when one does.
func (r SkillRename) nameAt(s string, at int) (string, string, bool) {
	if at > 0 && nameByte(s[at-1]) {
		return "", "", false
	}

	for _, prefix := range settings.SkillPrefixes() {
		rest, prefixed := strings.CutPrefix(s[at:], prefix+"-")
		if !prefixed {
			continue
		}

		for _, verb := range r.verbs {
			if !strings.HasPrefix(rest, verb) {
				continue
			}

			end := at + len(prefix) + 1 + len(verb)
			if end < len(s) && nameByte(s[end]) {
				continue
			}

			return s[at:end], verb, true
		}
	}

	return "", "", false
}

// nameByte reports whether a byte could be part of a skill name, which is what decides where one
// ends: the character classes a skill directory is spelled from, and the hyphen that joins them.
func nameByte(b byte) bool {
	return b == '-' || b == '_' ||
		('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z') || ('0' <= b && b <= '9')
}
