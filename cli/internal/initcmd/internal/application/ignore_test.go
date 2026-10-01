package application

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
	"github.com/lividlabs/codefall/cli/internal/shared/userfile"
)

var (
	ignoreFull        = filepath.Join(workingDir, settings.IgnoreName)
	gitIgnoreFull     = filepath.Join(workingDir, settings.GitIgnoreName)
	gitAttributesFull = filepath.Join(workingDir, settings.GitAttributesName)
	// The testing root the fixture's request declares, which is what the artifacts entry names.
	artifactsEntry = settings.TestArtifacts(settings.DefaultTestDir)
)

// gitIgnored is a .gitignore that already names all three of its entries and a .gitattributes that
// already names its one, for the tests about the other file.
func gitIgnored(files *fakeFileSystem) *fakeFileSystem {
	files.files[gitIgnoreFull] = []byte(settings.RefreshStamp + "\n" + userfile.Name + "\n" + artifactsEntry + "\n")
	files.files[gitAttributesFull] = []byte(settings.InteractionsAttribute + "\n")

	return files
}

// ignoreResult is the last result of a run, which is the ignore step's: it runs last so that
// .ignore is the author's file to commit rather than one bd init sweeps up.
func ignoreResult(t *testing.T, files *fakeFileSystem) domain.StepResult {
	t.Helper()

	report, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(
		t.Context(), beadsRequest(), nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	results := report.Results()

	return results[len(results)-1]
}

func TestIgnoreStepWritesTheFilesWhenThereAreNone(t *testing.T) {
	files := settled("{}")

	result := ignoreResult(t, files)

	if result.Step != domain.IgnoreStep {
		t.Fatalf("the last step = %+v, want %+v", result.Step, domain.IgnoreStep)
	}

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	if want := "wrote .ignore, wrote .gitignore and wrote .gitattributes"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	// Each file is written once with every entry it needs, each under the comment that says why it
	// is there and with a blank line between them.
	want := settings.IgnoreComment + "\n" + settings.IgnoreEntry + "\n\n" +
		settings.IgnoreTestsComment + "\n" + settings.IgnoreEntryTests + "\n"
	if got := string(files.files[ignoreFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.IgnoreName, got, want)
	}

	want = settings.GitIgnoreComment + "\n" + settings.RefreshStamp + "\n\n" +
		userfile.GitIgnoreComment + "\n" + userfile.Name + "\n\n" +
		settings.TestArtifactsComment + "\n" + artifactsEntry + "\n"
	if got := string(files.files[gitIgnoreFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.GitIgnoreName, got, want)
	}

	want = settings.GitAttributesComment + "\n" + settings.InteractionsAttribute + "\n"
	if got := string(files.files[gitAttributesFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.GitAttributesName, got, want)
	}
}

// A project that already declares attributes — line endings, linguist overrides — keeps them and
// gains the merge driver after them.
func TestIgnoreStepAddsTheMergeDriverToAGitattributesThatIsAlreadyThere(t *testing.T) {
	files := settled("{}")
	files.files[ignoreFull] = []byte(settings.IgnoreEntry + "\n" + settings.IgnoreEntryTests + "\n")
	files.files[gitIgnoreFull] = []byte(settings.RefreshStamp + "\n" + userfile.Name + "\n" + artifactsEntry + "\n")
	files.files[gitAttributesFull] = []byte("* text=auto\n")

	result := ignoreResult(t, files)

	if want := "added " + settings.InteractionsAttribute + " to .gitattributes"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want := "* text=auto\n\n" + settings.GitAttributesComment + "\n" + settings.InteractionsAttribute + "\n"
	if got := string(files.files[gitAttributesFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.GitAttributesName, got, want)
	}
}

// The run output of a testing root the project moved is ignored at the root it declared, not at the
// default one.
func TestIgnoreStepNamesTheDeclaredTestingRoot(t *testing.T) {
	files := settled("{}")

	request := beadsRequest()
	request.TestDir = "packages/web/e2e"

	if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource(), noChanges()).Run(
		t.Context(), request, nil,
	); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := string(files.files[gitIgnoreFull]); !strings.Contains(got, "packages/web/e2e/.artifacts/") {
		t.Errorf("%s =\n%q\nwant it to name the declared root's artifacts", settings.GitIgnoreName, got)
	}
}

// A project's .gitignore is nearly always its own already, so the entries are appended to it, and
// the .ignore file that is already right is left alone and left out of the sentence.
func TestIgnoreStepAddsTheEntriesToAGitignoreThatIsAlreadyThere(t *testing.T) {
	files := settled("{}")
	files.files[ignoreFull] = []byte(settings.IgnoreEntry + "\n" + settings.IgnoreEntryTests + "\n")
	files.files[gitIgnoreFull] = []byte("node_modules/\ndist/\n")
	files.files[gitAttributesFull] = []byte(settings.InteractionsAttribute + "\n")

	result := ignoreResult(t, files)

	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "added " + settings.RefreshStamp + ", " + userfile.Name + " and " + artifactsEntry + " to .gitignore"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want = "node_modules/\ndist/\n\n" + settings.GitIgnoreComment + "\n" + settings.RefreshStamp + "\n\n" +
		userfile.GitIgnoreComment + "\n" + userfile.Name + "\n\n" +
		settings.TestArtifactsComment + "\n" + artifactsEntry + "\n"
	if got := string(files.files[gitIgnoreFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.GitIgnoreName, got, want)
	}

	if got := string(files.files[ignoreFull]); got != settings.IgnoreEntry+"\n"+settings.IgnoreEntryTests+"\n" {
		t.Errorf("%s =\n%q\nwant it untouched", settings.IgnoreName, got)
	}
}

// The file may be the project's own, so the step appends: overwriting a file that has drifted is the
// one thing the extension's rules never allow.
func TestIgnoreStepAppendsToAFileThatAlreadySaysSomething(t *testing.T) {
	for _, tc := range []struct {
		name     string
		existing string
	}{
		{name: "the file ends with a newline", existing: "vendor/\nnode_modules/\n"},
		{name: "the file does not", existing: "vendor/\nnode_modules/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := gitIgnored(settled("{}"))
			files.files[ignoreFull] = []byte(tc.existing)

			result := ignoreResult(t, files)

			if result.Outcome != domain.OutcomeDone {
				t.Errorf("outcome = %v, want DONE", result.Outcome)
			}

			got := string(files.files[ignoreFull])

			if !strings.HasPrefix(got, "vendor/\nnode_modules/\n") {
				t.Errorf("%s =\n%q\nwant it to keep what was there", settings.IgnoreName, got)
			}

			if !strings.HasSuffix(got, settings.IgnoreEntryTests+"\n") {
				t.Errorf("%s =\n%q\nwant it to end with %q", settings.IgnoreName, got, settings.IgnoreEntryTests)
			}

			// The entry a file did not end with a newline before must not be joined to its last line.
			if strings.Contains(got, "node_modules/"+settings.IgnoreEntry) {
				t.Errorf("%s =\n%q\nwant the entry on a line of its own", settings.IgnoreName, got)
			}
		})
	}
}

// A file that names one entry and not the other gains the one it is missing and keeps the one it
// has, which is how the second entry reaches a project set up before it existed.
func TestIgnoreStepAddsOnlyTheEntryAFileIsMissing(t *testing.T) {
	files := gitIgnored(settled("{}"))
	files.files[ignoreFull] = []byte(settings.IgnoreComment + "\n" + settings.IgnoreEntry + "\n")

	result := ignoreResult(t, files)

	if want := "added " + settings.IgnoreEntryTests + " to .ignore"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	got := string(files.files[ignoreFull])

	if strings.Count(got, settings.IgnoreEntry+"\n") != 1 {
		t.Errorf("%s =\n%q\nwant the entry it had written once", settings.IgnoreName, got)
	}

	if !strings.HasSuffix(got, settings.IgnoreEntryTests+"\n") {
		t.Errorf("%s =\n%q\nwant the missing entry added", settings.IgnoreName, got)
	}
}

// A rerun must not grow a file. The step compares whole lines, so an indented entry counts as
// present and a commented-out one does not — ripgrep does not read the comment either.
func TestIgnoreStepLeavesAFileThatAlreadyNamesItsEntries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		existing string
		skipped  bool
	}{
		{
			name:     "written plainly",
			existing: settings.IgnoreEntry + "\n" + settings.IgnoreEntryTests + "\n",
			skipped:  true,
		},
		{
			name:     "indented",
			existing: "  " + settings.IgnoreEntry + "  \n\t" + settings.IgnoreEntryTests + "\n",
			skipped:  true,
		},
		{
			name:     "commented out",
			existing: "# " + settings.IgnoreEntry + "\n# " + settings.IgnoreEntryTests + "\n",
			skipped:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := gitIgnored(settled("{}"))
			files.files[ignoreFull] = []byte(tc.existing)

			result := ignoreResult(t, files)

			if tc.skipped {
				if result.Outcome != domain.OutcomeSkipped {
					t.Errorf("outcome = %v, want SKIPPED", result.Outcome)
				}

				want := ".ignore already names " + settings.IgnoreEntry + " and " + settings.IgnoreEntryTests +
					", .gitignore already names " + settings.RefreshStamp + ", " + userfile.Name + " and " +
					artifactsEntry +
					" and .gitattributes already names " + settings.InteractionsAttribute
				if result.Detail != want {
					t.Errorf("detail = %q, want %q", result.Detail, want)
				}

				if got := string(files.files[ignoreFull]); got != tc.existing {
					t.Errorf("%s =\n%q\nwant it untouched", settings.IgnoreName, got)
				}

				return
			}

			if result.Outcome != domain.OutcomeDone {
				t.Errorf("outcome = %v, want DONE", result.Outcome)
			}

			if !strings.HasSuffix(string(files.files[ignoreFull]), settings.IgnoreEntryTests+"\n") {
				t.Errorf("%s =\n%q\nwant the entries added", settings.IgnoreName, files.files[ignoreFull])
			}
		})
	}
}

// A project set up before the user file existed has a .gitignore naming the stamp and the run output
// and not the user file, and an upgrade adds that one line under its comment and nothing else.
func TestIgnoreStepAddsTheUserFileToAGitignoreFromBeforeItExisted(t *testing.T) {
	files := settled("{}")
	before := settings.GitIgnoreComment + "\n" + settings.RefreshStamp + "\n\n" +
		settings.TestArtifactsComment + "\n" + artifactsEntry + "\n"
	files.files[ignoreFull] = []byte(settings.IgnoreEntry + "\n" + settings.IgnoreEntryTests + "\n")
	files.files[gitIgnoreFull] = []byte(before)
	files.files[gitAttributesFull] = []byte(settings.InteractionsAttribute + "\n")

	result := ignoreResult(t, files)

	if want := "added " + userfile.Name + " to .gitignore"; result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	want := before + "\n" + userfile.GitIgnoreComment + "\n" + userfile.Name + "\n"
	if got := string(files.files[gitIgnoreFull]); got != want {
		t.Errorf("%s =\n%q\nwant\n%q", settings.GitIgnoreName, got, want)
	}
}
