package domain

import (
	"strings"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// shipped is a tree of skills the way the embedded tree names them, including one verb that is a
// prefix of another's spelling, so the longest-first rule has something to decide.
var shipped = []string{"codefall-design", "codefall-mock-up", "codefall-mock", "codefall-implement", "codefall-upgrade"}

func renameOnto(t *testing.T, prefix string) SkillRename {
	t.Helper()

	rename, err := NewSkillRename(prefix, shipped)
	if err != nil {
		t.Fatalf("NewSkillRename(%q): %v", prefix, err)
	}

	return rename
}

// Every form a skill is named in — the directory, the frontmatter, the slash command, the relative
// path into another skill, the sentence — is one whole name, and the rewrite finds each of them.
func TestSkillRenameRewritesWholeNamesInEveryForm(t *testing.T) {
	rename := renameOnto(t, settings.SkillPrefixCf)

	for _, tc := range []struct {
		in, want string
	}{
		{"skills/codefall-design/SKILL.md", "skills/cf-design/SKILL.md"},
		{"name: codefall-design\n", "name: cf-design\n"},
		{"run `/codefall-implement DESIGN-003`", "run `/cf-implement DESIGN-003`"},
		{"../codefall-upgrade/templates/x.md", "../cf-upgrade/templates/x.md"},
		{"`codefall-design`, then `codefall-implement`.", "`cf-design`, then `cf-implement`."},
		{"codefall-design/codefall-implement", "cf-design/cf-implement"},
		{"codefall-mock-up and codefall-mock", "cf-mock-up and cf-mock"},
		{"codefall-mock-upstream", "codefall-mock-upstream"},
		{"(codefall-design)", "(cf-design)"},
		{"codefall-design", "cf-design"},
	} {
		if got := rename.Text(tc.in); got != tc.want {
			t.Errorf("Text(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A name that is not a skill is left alone whatever it looks like: the install's own directory, the
// binary, the guard scripts, a word that happens to contain a skill name, and a skill the tree does
// not ship.
func TestSkillRenameLeavesWhatIsNotASkillName(t *testing.T) {
	rename := renameOnto(t, settings.SkillPrefixCf)

	for _, in := range []string{
		".codefall/shared/preflight.sh",
		"the codefall binary",
		"codefall-block-merge-to-main.sh",
		"codefall-session-notice.sh",
		"codefall-land-documents.yml",
		"codefall-merge-guard",
		"codefall-designs",
		"my-codefall-design",
		"codefall_design",
		"codefall-lineage",
		"codefall-conceptualize",
		"xcodefall-design",
		"codefall-design2",
		"codefall-",
	} {
		if got := rename.Text(in); got != in {
			t.Errorf("Text(%q) = %q, want it unchanged", in, got)
		}
	}
}

// With the default prefix the source comes out as it went in, byte for byte: that is what keeps every
// project that set nothing on the install it has.
func TestSkillRenameOntoTheDefaultIsTheIdentityOnTheSource(t *testing.T) {
	rename := renameOnto(t, settings.SkillPrefixCodefall)

	source := "---\nname: codefall-design\n---\nRun `/codefall-implement` next; see ../codefall-upgrade/ and `.codefall/`.\n"
	if got := rename.Text(source); got != source {
		t.Errorf("Text onto the default changed the source:\n%s\nwant\n%s", got, source)
	}
}

// A skill under any prefix a project may choose is recognised, so a project moving from cf back to
// codefall, or on to cfall, sees each old directory as the rename it is.
func TestSkillRenameRecognisesEveryPrefixAProjectMayChoose(t *testing.T) {
	onto := renameOnto(t, settings.SkillPrefixCfall)

	for _, tc := range []struct{ in, want string }{
		{"codefall-design", "cfall-design"},
		{"cf-design", "cfall-design"},
		{"cfall-design", "cfall-design"},
		{".agents/skills/cf-design/SKILL.md", ".agents/skills/cfall-design/SKILL.md"},
	} {
		if got := onto.Text(tc.in); got != tc.want {
			t.Errorf("Text(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	back := renameOnto(t, settings.SkillPrefixCodefall)

	if got, want := back.Text("cf-design"), "codefall-design"; got != want {
		t.Errorf("Text(%q) onto the default = %q, want %q", "cf-design", got, want)
	}
}

// The zero value renames nothing, and a rewrite with no skills is the same: nothing to find.
func TestSkillRenameZeroValueRenamesNothing(t *testing.T) {
	if got := (SkillRename{}).Text("codefall-design"); got != "codefall-design" {
		t.Errorf("zero value Text = %q, want it unchanged", got)
	}

	empty, err := NewSkillRename(settings.SkillPrefixCf, nil)
	if err != nil {
		t.Fatalf("NewSkillRename with no skills: %v", err)
	}

	if got := empty.Text("codefall-design"); got != "codefall-design" {
		t.Errorf("Text with no skills = %q, want it unchanged", got)
	}

	if got := (SkillRename{}).Prefix(); got != "" {
		t.Errorf("zero value Prefix() = %q, want \"\"", got)
	}

	if got := empty.Prefix(); got != settings.SkillPrefixCf {
		t.Errorf("Prefix() = %q, want %q", got, settings.SkillPrefixCf)
	}
}

// A prefix outside the closed set is the validator's refusal, and a skill the tree names any other
// way is a tree the authoring rule forbids: both are errors naming what was wrong.
func TestNewSkillRenameRefusesAnUnknownPrefixAndAnUnprefixedSkill(t *testing.T) {
	if _, err := NewSkillRename("code", shipped); err == nil || !strings.Contains(err.Error(), `unknown skill prefix "code"`) {
		t.Errorf("NewSkillRename(code) error = %v, want the unknown prefix named", err)
	}

	for _, skill := range []string{"design", "cf-design", "codefall-"} {
		_, err := NewSkillRename(settings.SkillPrefixCf, []string{skill})
		if err == nil || !strings.Contains(err.Error(), `skill "`+skill+`" is not named codefall-<verb>`) {
			t.Errorf("NewSkillRename with skill %q error = %v, want the skill named", skill, err)
		}
	}
}

// Names finds what a text calls the skills, under any prefix, bounded the way Text bounds a rename,
// and in the order written.
func TestSkillRenameNamesEverySkillNameInAText(t *testing.T) {
	rename := renameOnto(t, settings.SkillPrefixCf)

	got := rename.Names("Run `/codefall-implement`, then cf-design; not codefall-designs, codefall-lineage, or .codefall/. See ../cfall-upgrade/.")
	want := []string{"codefall-implement", "cf-design", "cfall-upgrade"}

	if len(got) != len(want) {
		t.Fatalf("Names() = %q, want %q", got, want)
	}

	for at := range want {
		if got[at] != want[at] {
			t.Errorf("Names()[%d] = %q, want %q", at, got[at], want[at])
		}
	}

	if got := (SkillRename{}).Names("codefall-design"); len(got) != 0 {
		t.Errorf("zero value Names() = %q, want none", got)
	}
}
