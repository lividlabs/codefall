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
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// settings runs checks 1 to 4, and hands on to the five ignore-entry checks and the persona check.
// Each of the first four is the prerequisite of the next, so the first failure ends the group and the remaining checks are
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

	return append(d.ignored(dir, results), d.persona(dir))
}

// ignoredEntries is what each of the three files has to name, one check each and in the order
// doctor reports them. The .ignore entries are what codefall commits and nobody greps: review
// findings, and the report an agentic test run leaves (ADR-007). The .gitignore entries are the
// refresh stamp, which belongs to one machine (ADR-005), and the user file, which belongs to one
// person. The .gitattributes entry is the union merge for
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
	{domain.UserIgnored, settings.GitIgnoreName, userfile.Name},
	{domain.InteractionsMerged, settings.GitAttributesName, settings.InteractionsAttribute},
}

// ignored is checks 5 to 9: each entry codefall needs in an ignore or attributes file is there.
//
// They warn rather than fail. Nothing stops working without an entry — findings and reports are
// still written and still tracked, and every other verb behaves identically. What goes wrong is
// quieter: agents searching the codebase start reading old findings as if they were code, a
// committed stamp tells every other clone it was current at a commit it never refreshed at, a
// committed user file hands one person's persona to everyone who clones the project, and a merge of
// the interaction log stops on a conflict that has only one right answer. That is worth reporting
// and is not worth an exit status.
//
// Each check is independent of the ones beside it, so all five run whatever any of them found.
func (d *Diagnose) ignored(dir string, results []domain.Result) []domain.Result {
	for _, want := range ignoredEntries {
		remedy := "add " + want.entry + " to " + want.file + ", or run " + d.setupCommand(dir)
		if want.check.ID == domain.UserIgnored.ID {
			remedy = d.userIgnoredRemedy(dir)
		}

		results = append(results, d.entryIgnored(dir, want.check, want.file, want.entry, mo.Some(remedy)))
	}

	return results
}

// userIgnoredRemedy is what puts the user file's line back: `codefall config persona` writes it
// whenever it is missing, and is given the persona the person already has so that nothing else
// changes, or the setup command writes it with everything else codefall installs.
func (d *Diagnose) userIgnoredRemedy(dir string) string {
	return "run codefall config persona " + d.currentPersona(dir) + ", which adds the line, or run " +
		d.setupCommand(dir)
}

// currentPersona is the persona the skills use for this person: the user file's, when the file is
// there and valid, and the default otherwise, which is what the preflight script falls back to.
func (d *Diagnose) currentPersona(dir string) string {
	data, err := d.files.ReadFile(filepath.Join(dir, userfile.Name))
	if err != nil {
		return userfile.DefaultPersona
	}

	var doc userfile.Document
	if err := json.Unmarshal(data, &doc); err != nil || len(userfile.Validate(doc)) > 0 {
		return userfile.DefaultPersona
	}

	return userfile.Persona(doc)
}

// entryIgnored is one of those checks: the file names the entry, or it says which file is missing
// which line.
func (d *Diagnose) entryIgnored(dir string, check domain.Check, file, entry string,
	remedy mo.Option[string]) domain.Result {
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

// persona is check 10: the user file, when there is one, is valid and names a persona codefall knows.
// No file means the default persona, which is what every project had before the file existed, so it
// passes and says which persona that is.
//
// It fails rather than warns. The skills read the persona through the shared preflight script, which
// falls back to the default on a file it cannot read, so a person who wrote the file to change how
// the skills talk to them would get the default without being told. The file is one person's and is
// never checked in, so the remedy is theirs: set the persona with `codefall config persona`, or remove
// the file. The command rewrites a file it can parse and refuses one it cannot, so a file that is not
// a JSON object is removed first.
func (d *Diagnose) persona(dir string) domain.Result {
	choices := "codefall config persona <" + strings.Join(userfile.Personas(), "|") + ">"
	fixRemedy := mo.Some("run " + choices + ", or remove " + userfile.Name + " to use the " +
		userfile.DefaultPersona + " persona")
	removeRemedy := mo.Some("remove " + userfile.Name + " to use the " + userfile.DefaultPersona +
		" persona, then run " + choices + " to choose another")

	data, err := d.files.ReadFile(filepath.Join(dir, userfile.Name))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return domain.Persona.PassWithDetail("persona: " + userfile.DefaultPersona + " by default")
	case err != nil:
		return domain.Persona.Fail(
			fmt.Sprintf("Cannot read %s: %v", userfile.Name, err), mo.None[string]())
	}

	var doc userfile.Document

	if err := json.Unmarshal(data, &doc); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return domain.Persona.Fail(userfile.Name+"'s top level must be a JSON object", removeRemedy)
		}

		return domain.Persona.Fail(fmt.Sprintf("%s is not valid JSON: %v", userfile.Name, err), removeRemedy)
	}

	if problems := userfile.Validate(doc); len(problems) > 0 {
		return domain.Persona.Fail(userfile.Name+" is not valid: "+strings.Join(problems, "; "), fixRemedy)
	}

	return domain.Persona.PassWithDetail("persona: " + userfile.Persona(doc))
}
