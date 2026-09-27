package application

import (
	"reflect"
	"testing"
)

// releases is a changelog with breaking changes at three releases, newest first as the file lists
// them, and a version the range has to leave out at each end.
func releases() *fakeChangeLog {
	return &fakeChangeLog{releases: []BreakingRelease{
		{Version: "0.21.0", Notes: []string{"after the binary"}},
		{Version: "0.20.0", Notes: []string{"the binary's own"}},
		{Version: "0.17.0", Notes: []string{"conceptualize is envision", "the Beads section changed"}},
		{Version: "0.14.0", Notes: []string{"the layout moved"}},
		{Version: "0.13.0", Notes: []string{"the flags were renamed"}},
	}}
}

func TestBreakingChangesListsTheReleasesBetweenTheRecordAndTheBinaryEarliestFirst(t *testing.T) {
	files := settled("")
	files.files[manifestFull] = []byte(`{"harnesses": {"claude": {"version": "0.14.0", "files": ["x"]}}}`)

	got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), releases()).BreakingChanges(workingDir, "0.20.0")
	if err != nil {
		t.Fatalf("BreakingChanges: %v", err)
	}

	want := []BreakingRelease{
		{Version: "0.17.0", Notes: []string{"conceptualize is envision", "the Beads section changed"}},
		{Version: "0.20.0", Notes: []string{"the binary's own"}},
	}

	between, ok := got.Get()
	if !ok || !reflect.DeepEqual(between, want) {
		t.Errorf("BreakingChanges() = %+v, %v, want %+v", between, ok, want)
	}
}

// The oldest install is the one that crosses the most releases, so the earliest recorded version is
// the start of the range — the shared entry counted with the harnesses' own.
func TestBreakingChangesStartsFromTheEarliestRecordedVersion(t *testing.T) {
	files := settled("")
	files.files[manifestFull] = []byte(`{
  "harnesses": {"claude": {"version": "v0.19.0", "files": ["x"]}, "codex": {"version": "0.16.0", "files": ["y"]}},
  "shared": {"version": "0.18.0", "files": ["z"]}
}`)

	got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), releases()).BreakingChanges(workingDir, "v0.20.0")
	if err != nil {
		t.Fatalf("BreakingChanges: %v", err)
	}

	between, ok := got.Get()
	if !ok || len(between) != 2 || between[0].Version != "0.17.0" || between[1].Version != "0.20.0" {
		t.Errorf("BreakingChanges() = %+v, %v, want 0.17.0 and 0.20.0", between, ok)
	}
}

// A range with no releases in it is an empty list, not an undetermined one: the caller prints
// nothing, and knows that nothing is the answer.
func TestBreakingChangesIsEmptyWhenNoReleaseBetweenRecordedAny(t *testing.T) {
	files := settled("")
	files.files[manifestFull] = []byte(`{"harnesses": {"claude": {"version": "0.17.0", "files": ["x"]}}}`)

	got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), releases()).BreakingChanges(workingDir, "0.19.0")
	if err != nil {
		t.Fatalf("BreakingChanges: %v", err)
	}

	between, ok := got.Get()
	if !ok || len(between) != 0 {
		t.Errorf("BreakingChanges() = %+v, %v, want an empty list", between, ok)
	}
}

// The range cannot be determined when either end is not a release, or when there is no record: a
// development build at either end, a manifest with no version, no manifest at all.
func TestBreakingChangesIsUndeterminedWithoutAReleaseAtBothEnds(t *testing.T) {
	for _, tc := range []struct {
		name     string
		manifest string
		binary   string
	}{
		{name: "the binary is a development build", manifest: `{"harnesses": {"claude": {"version": "0.17.0", "files": ["x"]}}}`, binary: "0.19.0-dev (abc1234)"},
		{name: "the record is a development build", manifest: `{"harnesses": {"claude": {"version": "0.17.0-dev (abc1234)", "files": ["x"]}}}`, binary: "0.19.0"},
		{name: "one of several records is a development build", manifest: `{"harnesses": {"claude": {"version": "0.17.0", "files": ["x"]}, "codex": {"version": "dev", "files": ["y"]}}}`, binary: "0.19.0"},
		{name: "the record carries no version", manifest: `{"harnesses": {"claude": {"files": ["x"]}}}`, binary: "0.19.0"},
		{name: "there is no manifest", manifest: "", binary: "0.19.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := settled("")
			if tc.manifest != "" {
				files.files[manifestFull] = []byte(tc.manifest)
			}

			got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), releases()).BreakingChanges(workingDir, tc.binary)
			if err != nil {
				t.Fatalf("BreakingChanges: %v", err)
			}

			if got.IsPresent() {
				t.Errorf("BreakingChanges() = %+v, want None", got.MustGet())
			}
		})
	}
}
