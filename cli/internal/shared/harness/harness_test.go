package harness

import (
	"slices"
	"testing"

	"github.com/samber/mo"
)

func TestAllIsSortedAndCopied(t *testing.T) {
	want := []string{Agy, Claude, Codex, Muse, OpenCode}

	if got := All(); !slices.Equal(got, want) {
		t.Errorf("All() = %q, want %q", got, want)
	}

	// The caller gets a copy: mutating it must not change what the next caller sees.
	All()[0] = "mutated"

	if got := All()[0]; got != Agy {
		t.Errorf("All()[0] after a caller mutated its copy = %q, want %q", got, Agy)
	}
}

func TestParse(t *testing.T) {
	if got, err := Parse(Codex); err != nil || got != Codex {
		t.Errorf("Parse(%q) = %q, %v, want %q, nil", Codex, got, err, Codex)
	}

	err := errorFrom(Parse("cursor"))

	want := `harness "cursor" is not supported yet (supported: agy, claude, codex, muse, opencode)`
	if err == nil || err.Error() != want {
		t.Errorf("Parse(%q) error = %v, want %q", "cursor", err, want)
	}
}

// A project's checked-in files may still carry the spelling a harness had before it was named for
// its binary. Parse reads it as the harness it names now, so that project keeps working, and never
// hands the old spelling back for a caller to write down again.
func TestParseAcceptsAFormerSpellingAndReturnsTheCurrentName(t *testing.T) {
	for _, tc := range []struct{ former, want string }{
		{"claude-code", Claude},
		{"antigravity", Agy},
	} {
		t.Run(tc.former, func(t *testing.T) {
			if got, err := Parse(tc.former); err != nil || got != tc.want {
				t.Errorf("Parse(%q) = %q, %v, want %q, nil", tc.former, got, err, tc.want)
			}
		})
	}
}

// The former spellings are accepted and never offered: All is what a person is shown, and it names
// only the harnesses as they are called now.
func TestAllOffersNoFormerSpelling(t *testing.T) {
	for _, former := range []string{"claude-code", "antigravity"} {
		if slices.Contains(All(), former) {
			t.Errorf("All() = %q, which offers the former spelling %q", All(), former)
		}
	}
}

func TestRenamedNamesTheCurrentHarnessForAFormerSpellingOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		want mo.Option[string]
	}{
		{"claude-code", mo.Some(Claude)},
		{"antigravity", mo.Some(Agy)},
		{Claude, mo.None[string]()},
		{Codex, mo.None[string]()},
		{"cursor", mo.None[string]()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Renamed(tc.name); got != tc.want {
				t.Errorf("Renamed(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// errorFrom drops the value of a (T, error) pair so a table can assert on the error alone.
func errorFrom(_ string, err error) error {
	return err
}

// Every harness codefall can set up has somewhere to install, which is what makes the table the
// roster: a row that went missing would take the name out of All and Parse as well, rather than
// leaving a name the steps could accept and then have nowhere to write.
func TestEveryHarnessHasSomewhereToInstall(t *testing.T) {
	for _, name := range All() {
		dir, known := SkillsDir(name).Get()

		switch {
		case !known:
			t.Errorf("SkillsDir(%q) = None, want a directory", name)
		case dir == "":
			t.Errorf("SkillsDir(%q) = %q, want a directory", name, dir)
		}
	}
}

func TestSkillsDirNamesTheConventionEachHarnessFollows(t *testing.T) {
	for _, tc := range []struct {
		harness string
		want    string
	}{
		{Claude, ".claude"},
		{Agy, ".agents"},
		{Codex, ".agents"},
		{Muse, ".agents"},
		{OpenCode, ".agents"},
	} {
		t.Run(tc.harness, func(t *testing.T) {
			if got := SkillsDir(tc.harness).OrEmpty(); got != tc.want {
				t.Errorf("SkillsDir(%q) = %q, want %q", tc.harness, got, tc.want)
			}
		})
	}
}

// A harness codefall cannot set up has no directory to report, and the absence is the answer rather
// than an empty string that reads like one.
func TestSkillsDirIsAbsentForAHarnessCodefallCannotSetUp(t *testing.T) {
	if got := SkillsDir("aider"); got.IsPresent() {
		t.Errorf("SkillsDir(%q) = %v, want None", "aider", got)
	}
}
