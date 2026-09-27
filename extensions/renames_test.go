package extensions

import (
	"io/fs"
	"testing"
)

// Every name the table renames to is a skill the tree ships, and every name it renames from is not:
// a row pointing at a directory that does not exist would report a removal as a rename to nothing,
// and a former name still shipped would be removed and reinstalled on every upgrade.
func TestSkillRenamesPointFromRetiredNamesToShippedSkills(t *testing.T) {
	for former, current := range SkillRenames() {
		if _, err := fs.Stat(Files(), "skills/"+current+"/SKILL.md"); err != nil {
			t.Errorf("%s is renamed to %s, which the tree does not ship: %v", former, current, err)
		}

		if _, err := fs.Stat(Files(), "skills/"+former); err == nil {
			t.Errorf("%s is recorded as a former name, and the tree still ships it", former)
		}
	}
}
