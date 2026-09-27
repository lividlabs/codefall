package application

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// ignoreEntry is one line codefall needs in an ignore file: the entry, and the comment that says why
// it is there for whoever finds the file later.
type ignoreEntry struct {
	entry   string
	comment string
}

// ignoreFile is one file and every entry codefall needs in it, in the order the step writes them.
type ignoreFile struct {
	file    string
	entries []ignoreEntry
}

// ignoreFiles is what the step writes. The .ignore entries keep what codefall commits and nobody
// greps — review findings, and the report an agentic test run leaves (ADR-007) — out of every search
// that goes through ripgrep. The .gitignore entries keep out what belongs to one machine, one person,
// or one run: the refresh stamp (ADR-005), the user file, and everything a test run produces that is
// not its report. The
// .gitattributes entry is the one line that is not about ignoring: bd's interaction log is
// append-only and committed, and a union merge is what keeps two branches' appends from conflicting.
//
// It is a function because the .gitignore's last entry names the testing root, which the project
// chose.
func ignoreFiles(root string) []ignoreFile {
	return []ignoreFile{
		{file: settings.IgnoreName, entries: []ignoreEntry{
			{entry: settings.IgnoreEntry, comment: settings.IgnoreComment},
			{entry: settings.IgnoreEntryTests, comment: settings.IgnoreTestsComment},
		}},
		{file: settings.GitIgnoreName, entries: []ignoreEntry{
			{entry: settings.RefreshStamp, comment: settings.GitIgnoreComment},
			{entry: userfile.Name, comment: userfile.GitIgnoreComment},
			{entry: settings.TestArtifacts(root), comment: settings.TestArtifactsComment},
		}},
		{file: settings.GitAttributesName, entries: []ignoreEntry{
			{entry: settings.InteractionsAttribute, comment: settings.GitAttributesComment},
		}},
	}
}

// ignore is the step that writes the ignore entries, and the attributes entry beside them.
//
// Any of the files may already be the project's own, holding entries that have nothing to do with
// codefall, so this step appends rather than writes: overwriting a file that has drifted is the one
// thing the extension's rules never allow. A file that already names its entries is left exactly as
// it is, which is what makes a rerun a no-op rather than a growing list of duplicates.
func (i *Initialize) ignore(_ context.Context, request Request) (domain.StepResult, error) {
	files := ignoreFiles(testRoot(request))

	var done []string

	for _, file := range files {
		did, err := i.ensureIgnored(request.Dir, file)
		if err != nil {
			return domain.StepResult{}, err
		}

		if did != "" {
			done = append(done, did)
		}
	}

	if len(done) == 0 {
		return domain.IgnoreStep.Skipped(sentenceList(alreadyNamed(files))), nil
	}

	return domain.IgnoreStep.Done(sentenceList(done)), nil
}

// alreadyNamed is the skip's sentence, one clause per file: what it already names, so a reader can
// see the step had nothing to do rather than that it did nothing.
func alreadyNamed(files []ignoreFile) []string {
	clauses := make([]string, 0, len(files))

	for _, file := range files {
		entries := make([]string, 0, len(file.entries))
		for _, entry := range file.entries {
			entries = append(entries, entry.entry)
		}

		clauses = append(clauses, file.file+" already names "+sentenceList(entries))
	}

	return clauses
}

// ensureIgnored puts every entry one file is missing into it, in one read and one write, and says
// what that took — or "" when the file already had all of them.
func (i *Initialize) ensureIgnored(dir string, file ignoreFile) (string, error) {
	path := filepath.Join(dir, file.file)

	existing, err := i.files.ReadFile(path)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		var body strings.Builder

		for at, entry := range file.entries {
			if at > 0 {
				body.WriteString("\n")
			}

			body.WriteString(entry.comment + "\n" + entry.entry + "\n")
		}

		if err := i.files.WriteFile(path, []byte(body.String())); err != nil {
			return "", fmt.Errorf("write %s: %w", file.file, err)
		}

		return "wrote " + file.file, nil

	case err != nil:
		return "", fmt.Errorf("read %s: %w", file.file, err)
	}

	body := string(existing)

	var added []string

	for _, entry := range file.entries {
		if settings.NamesEntry(body, entry.entry) {
			continue
		}

		// A file that does not end in a newline would otherwise have its last entry joined to ours.
		if !strings.HasSuffix(body, "\n") {
			body += "\n"
		}

		body += "\n" + entry.comment + "\n" + entry.entry + "\n"
		added = append(added, entry.entry)
	}

	if len(added) == 0 {
		return "", nil
	}

	if err := i.files.WriteFile(path, []byte(body)); err != nil {
		return "", fmt.Errorf("write %s: %w", file.file, err)
	}

	return fmt.Sprintf("added %s to %s", sentenceList(added), file.file), nil
}
