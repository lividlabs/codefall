package manifest

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

const twoHarnesses = `{
  "harnesses": {
    "codex": {"version": "v1.1.0", "files": [".agents/skills/design/SKILL.md"]},
    "claude": {"version": "v1.2.3", "files": [".claude/skills/design/SKILL.md"]}
  }
}`

func TestDecodeReadsEveryEntry(t *testing.T) {
	document, err := Decode([]byte(twoHarnesses))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got, want := document.Recorded(), []string{"claude", "codex"}; !slices.Equal(got, want) {
		t.Errorf("Recorded() = %q, want %q", got, want)
	}

	want := map[string]string{"claude": "v1.2.3", "codex": "v1.1.0"}
	if got := document.Versions(); !maps.Equal(got, want) {
		t.Errorf("Versions() = %v, want %v", got, want)
	}

	if got, want := document.Harnesses["codex"].Files,
		[]string{".agents/skills/design/SKILL.md"}; !slices.Equal(got, want) {
		t.Errorf("codex files = %q, want %q", got, want)
	}
}

// A manifest written before a harness was named for its binary files it under the spelling it had.
// Current moves the entry to the name the harness has now and says which spellings it moved, so a
// reader can report the rename and compare the install under the name the settings use.
func TestCurrentMovesAFormerSpellingToTheNameTheHarnessHasNow(t *testing.T) {
	document, err := Decode([]byte(`{"harnesses": {
  "claude-code": {"version": "v1.2.3", "files": [".claude/skills/design/SKILL.md"]},
  "antigravity": {"version": "v1.2.3", "files": [".agents/skills/design/SKILL.md"]},
  "codex": {"version": "v1.2.3", "files": [".agents/skills/design/SKILL.md"]}
}}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	current, moved := document.Current()

	if want := []string{"antigravity", "claude-code"}; !slices.Equal(moved, want) {
		t.Errorf("Current() moved %q, want %q", moved, want)
	}

	if got, want := current.Recorded(), []string{"agy", "claude", "codex"}; !slices.Equal(got, want) {
		t.Errorf("Current().Recorded() = %q, want %q", got, want)
	}

	if got, want := current.Harnesses["claude"].Files,
		[]string{".claude/skills/design/SKILL.md"}; !slices.Equal(got, want) {
		t.Errorf("claude files = %q, want %q", got, want)
	}

	// The receiver is a value a caller may still be holding, so it keeps the spellings it had.
	if got, want := document.Recorded(), []string{"antigravity", "claude-code", "codex"}; !slices.Equal(got, want) {
		t.Errorf("Recorded() after Current = %q, want %q", got, want)
	}
}

// Both spellings on record means a newer binary has already written the current one, so its entry
// is the one kept, and the former spelling is still reported as moved so a rewrite drops it.
func TestCurrentKeepsTheEntryUnderTheCurrentNameWhenBothAreRecorded(t *testing.T) {
	document, err := Decode([]byte(`{"harnesses": {
  "claude-code": {"version": "v1.0.0", "files": ["old"]},
  "claude": {"version": "v1.2.3", "files": ["new"]}
}}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	current, moved := document.Current()

	if want := []string{"claude-code"}; !slices.Equal(moved, want) {
		t.Errorf("Current() moved %q, want %q", moved, want)
	}

	want := map[string]string{"claude": "v1.2.3"}
	if got := current.Versions(); !maps.Equal(got, want) {
		t.Errorf("Current().Versions() = %v, want %v", got, want)
	}
}

func TestCurrentMovesNothingInARecordThatUsesTheCurrentNames(t *testing.T) {
	document, err := Decode([]byte(twoHarnesses))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	current, moved := document.Current()
	if len(moved) != 0 {
		t.Errorf("Current() moved %q, want nothing", moved)
	}

	if !maps.Equal(current.Versions(), document.Versions()) {
		t.Errorf("Current().Versions() = %v, want %v", current.Versions(), document.Versions())
	}
}

// A file written before harnesses were recorded per install names none of them. It decodes rather
// than failing, and records nothing — the same position no file at all leaves a reader in.
func TestDecodeReadsAnOlderRecordAsEmpty(t *testing.T) {
	document, err := Decode([]byte(`{"harness": "codex", "version": "v1.2.3"}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got := document.Recorded(); len(got) != 0 {
		t.Errorf("Recorded() = %q, want none", got)
	}

	if got := document.Versions(); len(got) != 0 {
		t.Errorf("Versions() = %v, want none", got)
	}
}

// An entry with no version says an install was recorded without saying what installed it, which is
// nothing a reader can hold against a binary's tag or report as present.
func TestAnEntryWithNoVersionIsLeftOut(t *testing.T) {
	document, err := Decode([]byte(
		`{"harnesses": {"codex": {"files": ["x"]}, "muse": {"version": "v1.0.0"}}}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got, want := document.Recorded(), []string{"muse"}; !slices.Equal(got, want) {
		t.Errorf("Recorded() = %q, want %q", got, want)
	}

	if _, left := document.Versions()["codex"]; left {
		t.Errorf("Versions() = %v, want codex left out", document.Versions())
	}
}

// The files a run writes once belong to no harness, so they are their own entry. Recorded and
// Versions answer about harnesses and say nothing about it: a project has installs for harnesses,
// and .codefall/ is not one of them.
func TestTheSharedEntryIsItsOwnRecord(t *testing.T) {
	document, err := Decode([]byte(`{
  "harnesses": {"codex": {"version": "v1.2.3", "files": [".agents/skills/design/SKILL.md"]}},
  "shared": {"version": "v1.2.3", "files": [".codefall/shared/preflight.sh"]}
}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got, want := document.Shared.Files, []string{".codefall/shared/preflight.sh"}; !slices.Equal(got, want) {
		t.Errorf("Shared.Files = %q, want %q", got, want)
	}

	if got, want := document.Recorded(), []string{"codex"}; !slices.Equal(got, want) {
		t.Errorf("Recorded() = %q, want %q — the shared entry is not a harness", got, want)
	}

	// A file written before the entry existed decodes to an entry that records nothing, which is the
	// same position a reader is in with no file at all.
	older, err := Decode([]byte(twoHarnesses))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if older.Shared.Version != "" || len(older.Shared.Files) != 0 {
		t.Errorf("Shared = %+v, want an entry recording nothing", older.Shared)
	}
}

func TestDecodeNamesTheFileItCouldNotRead(t *testing.T) {
	_, err := Decode([]byte("{ not json"))
	if err == nil || !strings.Contains(err.Error(), Name) {
		t.Errorf("Decode error = %v, want it to name %s", err, Name)
	}
}

// What Encode writes, Decode reads back: the two halves of the format are one definition, and a
// change to either that the other cannot follow fails here.
func TestEncodeAndDecodeAgree(t *testing.T) {
	document := Document{Harnesses: map[string]Install{
		"claude": {Version: "v1.2.3", Files: []string{".claude/skills/design/SKILL.md"}},
	}}

	data, err := Encode(document)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	read, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got, want := read.Versions()["claude"], "v1.2.3"; got != want {
		t.Errorf("version after a round trip = %q, want %q", got, want)
	}

	if !strings.Contains(string(data), "\n  \"harnesses\"") {
		t.Errorf("Encode wrote %s, want it indented the way a person would write it", data)
	}
}
