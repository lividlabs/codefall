package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// settings runs checks 1 to 4, and hands on to the four ignore-entry checks. Each of the first four
// is the prerequisite of the next, so the first failure ends the group and the remaining checks are
// absent from the report.
//
// Every detail is a whole sentence, because the report prints it under a category header without the
// check's title in front of it.
//
// Decoding happens here rather than in infrastructure, which returns bytes, or in the domain, which
// must not name an encoding. It decodes into the generic document so the shared field tables stay
// the single definition of the settings shape and every problem is reported at once.
func (d *Diagnose) settings(_ context.Context, dir string, results []domain.Result) []domain.Result {
	codefallDir := filepath.Join(dir, ".codefall")
	settingsPath := filepath.Join(codefallDir, "settings.json")
	createRemedy := mo.Some("create .codefall/settings.json; schema: " + settings.SchemaID)

	exists, err := d.files.DirExists(codefallDir)

	switch {
	case err != nil:
		return append(results, domain.CodefallDir.Fail(
			fmt.Sprintf("Cannot stat %s: %v", codefallDir, err), mo.None[string]()))
	case !exists:
		return append(results, domain.CodefallDir.Fail(".codefall/ not found", createRemedy))
	}

	results = append(results, domain.CodefallDir.Pass())

	data, err := d.files.ReadFile(settingsPath)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return append(results, domain.SettingsFile.Fail(".codefall/settings.json not found", createRemedy))
	case err != nil:
		return append(results, domain.SettingsFile.Fail(
			fmt.Sprintf("Cannot read .codefall/settings.json: %v", err), mo.None[string]()))
	}

	results = append(results, domain.SettingsFile.Pass())

	fixRemedy := mo.Some("fix the fields above; schema: " + settings.SchemaID)

	var doc settings.Document

	if err := json.Unmarshal(data, &doc); err != nil {
		var syntaxErr *json.SyntaxError
		if errors.As(err, &syntaxErr) {
			return append(results, domain.SettingsJSON.Fail(
				fmt.Sprintf("settings.json is not valid JSON at byte %d: %v", syntaxErr.Offset, syntaxErr),
				mo.None[string]()))
		}

		// Well-formed JSON that is not an object parses cleanly and is simply the wrong shape, so
		// check 3 passes and check 4 carries the complaint.
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			results = append(results, domain.SettingsJSON.Pass())

			return append(results, domain.SettingsComplete.Fail(
				"settings.json's top level must be a JSON object", fixRemedy))
		}

		return append(results, domain.SettingsJSON.Fail(
			fmt.Sprintf("settings.json is not valid JSON: %v", err), mo.None[string]()))
	}

	results = append(results, domain.SettingsJSON.Pass())

	if problems := settings.Validate(doc); len(problems) > 0 {
		return append(results, domain.SettingsComplete.Fail(
			"settings.json is incomplete: "+strings.Join(problems, "; "), fixRemedy))
	}

	results = append(results, domain.SettingsComplete.Pass())

	return d.ignored(dir, results)
}

// ignoredEntries is what each of the three files has to name, one check each and in the order
// doctor reports them. The .ignore entries are what codefall commits and nobody greps: review
// findings, and the report an agentic test run leaves (ADR-007). The .gitignore entry is the refresh
// stamp, which belongs to one machine (ADR-005). The .gitattributes entry is the union merge for
// bd's append-only interaction log, which is committed and would otherwise conflict on every merge
// of two branches that both appended to it.
//
// A run's own output under the testing root is git-ignored too, but the entry names a path the
// project chose, and doctor's report is about what codefall can check without knowing it.
var ignoredEntries = []struct {
	check domain.Check
	file  string
	entry string
}{
	{domain.ReviewsIgnored, settings.IgnoreName, settings.IgnoreEntry},
	{domain.TestsIgnored, settings.IgnoreName, settings.IgnoreEntryTests},
	{domain.StampIgnored, settings.GitIgnoreName, settings.RefreshStamp},
	{domain.InteractionsMerged, settings.GitAttributesName, settings.InteractionsAttribute},
}

// ignored is checks 5 to 8: each entry codefall needs in an ignore or attributes file is there.
//
// They warn rather than fail. Nothing stops working without an entry — findings and reports are
// still written and still tracked, and every other verb behaves identically. What goes wrong is
// quieter: agents searching the codebase start reading old findings as if they were code, and a
// committed stamp tells every other clone it was current at a commit it never refreshed at, and a
// merge of the interaction log stops on a conflict that has only one right answer. That is worth
// reporting and is not worth an exit status.
//
// Each check is independent of the ones beside it, so all four run whatever any of them found.
func (d *Diagnose) ignored(dir string, results []domain.Result) []domain.Result {
	for _, want := range ignoredEntries {
		results = append(results, d.entryIgnored(dir, want.check, want.file, want.entry))
	}

	return results
}

// entryIgnored is one of those checks: the file names the entry, or it says which file is missing
// which line.
func (d *Diagnose) entryIgnored(dir string, check domain.Check, file, entry string) domain.Result {
	remedy := mo.Some("add " + entry + " to " + file + ", or run codefall upgrade")

	data, err := d.files.ReadFile(filepath.Join(dir, file))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return check.Warn(file+" not found", remedy)
	case err != nil:
		return check.Warn(fmt.Sprintf("Cannot read %s: %v", file, err), mo.None[string]())
	case settings.NamesEntry(string(data), entry):
		return check.Pass()
	default:
		return check.Warn(file+" does not name "+entry, remedy)
	}
}
