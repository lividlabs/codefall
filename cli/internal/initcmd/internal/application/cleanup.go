package application

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
)

// cleanup is the step an upgrade runs after the extension step: it removes what the previous finished
// run recorded writing and this run did not write, because a file the binary shipped then and does
// not ship now is a skill or a script the project no longer has a reader for (ADR-010).
//
// It reads the record, never the disk: only a path the manifest lists is a candidate, and only for
// the harnesses this run installs for, so a harness the settings have dropped keeps what codefall
// wrote for it and doctor keeps reporting that. A path this run wrote for any harness is never
// removed, whichever entry listed it, because four harnesses share one skills directory and a file
// one entry has stopped naming may be another's. A record with nothing to compare — an entry with no
// file list, a harness with no entry — removes nothing for that entry and says so.
func (i *Initialize) cleanup(request Request, previous manifest.Document, written installed) (domain.StepResult, error) {
	dests, err := skillsDirs(request)
	if err != nil {
		return domain.StepResult{}, err
	}

	// The record under the names the harnesses have now, which are the names this run wrote under.
	recorded, _ := previous.Current()

	kept := writtenPaths(written)

	var (
		candidates []removal
		unread     []string
	)

	for _, name := range chosen(request) {
		entry, listed := recorded.Harnesses[name]

		switch {
		case !listed:
			unread = append(unread, "no earlier install is recorded for "+name)
		case len(entry.Files) == 0:
			unread = append(unread, "the record for "+name+" lists no files")
		default:
			candidates = append(candidates, stale(entry.Files, kept, dests[name]+"/")...)
		}
	}

	switch {
	case recorded.Shared.Version == "" && len(recorded.Shared.Files) == 0:
		unread = append(unread, "no earlier install is recorded for "+codefallDir+"/")
	case len(recorded.Shared.Files) == 0:
		unread = append(unread, "the record for "+codefallDir+"/ lists no files")
	default:
		candidates = append(candidates, stale(recorded.Shared.Files, kept, codefallDir+"/")...)
	}

	removed, err := i.remove(request.Dir, dedupe(candidates))
	if err != nil {
		return domain.StepResult{}, err
	}

	notes := strings.Join(unread, "; ")

	if len(removed) == 0 {
		detail := "nothing to remove"
		if notes != "" {
			detail += "; " + notes + ", so nothing was compared there"
		}

		return domain.CleanupStep.Skipped(detail), nil
	}

	detail := "removed " + sentenceList(i.describeRemovals(removed, written, dests))
	if notes != "" {
		detail += "; " + notes + ", so nothing was compared there"
	}

	return domain.CleanupStep.Done(detail), nil
}

// removal is one path the record lists and this run did not write, and the directory the install it
// belonged to owns, which is the boundary an emptied parent is pruned up to.
type removal struct {
	path string
	root string
}

// writtenPaths is every path this run wrote, for any harness and for .codefall/, as a set.
func writtenPaths(written installed) map[string]bool {
	kept := map[string]bool{}

	for _, files := range written.harnesses {
		for _, file := range files {
			kept[file] = true
		}
	}

	for _, file := range written.shared {
		kept[file] = true
	}

	return kept
}

// stale is the recorded paths this run did not write, kept to the ones a run could have written: a
// clean relative path under root. A manifest is a file a person can edit, and a path that climbs out
// of the install directory is not one codefall wrote, whatever the record says.
func stale(recorded []string, kept map[string]bool, root string) []removal {
	var found []removal

	for _, file := range recorded {
		clean := path.Clean(file)

		if kept[file] || kept[clean] {
			continue
		}

		if clean == "." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") ||
			!strings.HasPrefix(clean, root) {
			continue
		}

		found = append(found, removal{path: clean, root: strings.TrimSuffix(root, "/")})
	}

	return found
}

// dedupe drops a path listed by two entries, which is what two harnesses sharing a skills directory
// record, and sorts what is left so two runs report the same order.
func dedupe(candidates []removal) []removal {
	slices.SortFunc(candidates, func(a, b removal) int { return strings.Compare(a.path, b.path) })

	return slices.CompactFunc(candidates, func(a, b removal) bool { return a.path == b.path })
}

// remove deletes each candidate and prunes the directories it leaves empty, up to but never
// including the install's own root, and returns the candidates it removed.
func (i *Initialize) remove(dir string, candidates []removal) ([]removal, error) {
	removed := make([]removal, 0, len(candidates))

	for _, candidate := range candidates {
		if err := i.files.Remove(path.Join(dir, candidate.path)); err != nil {
			return nil, fmt.Errorf("remove %s: %w", candidate.path, err)
		}

		removed = append(removed, candidate)

		for parent := path.Dir(candidate.path); parent != candidate.root && parent != "." && parent != "/"; parent = path.Dir(parent) {
			empty, err := i.files.DirIsEmpty(path.Join(dir, parent))
			if err != nil {
				return nil, fmt.Errorf("read %s/: %w", parent, err)
			}

			if !empty {
				break
			}

			if err := i.files.Remove(path.Join(dir, parent)); err != nil {
				return nil, fmt.Errorf("remove %s/: %w", parent, err)
			}
		}
	}

	return removed, nil
}

// describeRemovals is what the step reports: a skill directory this run wrote nothing into is named
// once, as a rename when the tree renamed it and as no longer shipped otherwise, and every other file
// is named on its own.
func (i *Initialize) describeRemovals(removed []removal, written installed, dests map[string]string) []string {
	kept := writtenPaths(written)

	var (
		described []string
		seen      = map[string]bool{}
	)

	for _, candidate := range removed {
		skill, isSkill := skillDirOf(candidate.path, dests)

		if !isSkill || survives(skill, kept) {
			described = append(described, candidate.path+" (no longer shipped)")

			continue
		}

		if seen[skill] {
			continue
		}

		seen[skill] = true

		if current, renamed := i.source.RenamedSkill(path.Base(skill)).Get(); renamed {
			described = append(described, skill+"/ (renamed to "+current+")")
		} else {
			described = append(described, skill+"/ (no longer shipped)")
		}
	}

	return described
}

// skillDirOf is the skill directory a path sits under — `.agents/skills/graft` for
// `.agents/skills/graft/SKILL.md` — when it sits under one of the skills directories this run
// installs into.
func skillDirOf(file string, dests map[string]string) (string, bool) {
	for _, dest := range dests {
		prefix := dest + "/" + skillsSource + "/"
		if !strings.HasPrefix(file, prefix) {
			continue
		}

		rest := strings.TrimPrefix(file, prefix)

		name, _, nested := strings.Cut(rest, "/")
		if !nested {
			return "", false
		}

		return prefix + name, true
	}

	return "", false
}

// survives reports whether this run wrote anything into the skill directory, in which case a removed
// file under it is a file the skill dropped rather than the skill going.
func survives(skill string, kept map[string]bool) bool {
	for file := range kept {
		if strings.HasPrefix(file, skill+"/") {
			return true
		}
	}

	return false
}
