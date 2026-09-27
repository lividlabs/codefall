package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// equipRemedy is what to do about a project whose local commands are not declared: the verb that
// drafts them, or declares the ones the project already has (ADR-005).
const equipRemedy = "run /codefall-equip to declare the project's start and update commands"

// assignment is a leading VAR=value on a command line, which sets an environment variable for the
// program that follows rather than naming the program.
var assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// local runs checks 14 and 15: the project's local start and update commands are declared, and the
// program each one runs can be found (ADR-005).
//
// It reads the same settings the settings group validated, so whatever skipped those checks skips
// these, and the checks are absent from the report rather than present with a status of their own.
func (d *Diagnose) local(_ context.Context, dir string, results []domain.Result) []domain.Result {
	if !settingsAreComplete(results) {
		return results
	}

	doc, read := d.settingsDocument(dir)
	if !read {
		return results
	}

	commands, declared := settings.LocalCommands(doc).Get()
	if !declared {
		// A warning, not a failure: every other verb behaves the same without the block. What is
		// missing is refresh, which is the thing that brings a teammate's environment current, and
		// the remedy is the verb that fills the block in.
		return append(results, domain.LocalDeclared.Warn(
			"no local start and update commands are declared in settings.json", mo.Some(equipRemedy)))
	}

	results = append(results, domain.LocalDeclared.Pass())

	return d.runnable(dir, commands, results)
}

// runnable is check 15: each declared command names a program that exists. A command that names a
// path is looked for in the project; one that names a bare program is looked for on PATH. This is
// as far as doctor can go without running anything, which it never does.
//
// It fails rather than warns. The project declared these commands, refresh will run them, and a
// program that is not there is a declaration nothing can act on.
func (d *Diagnose) runnable(dir string, commands settings.Local, results []domain.Result) []domain.Result {
	// One line per missing program, naming every field that runs it: start and update usually
	// name the same script, and saying it is missing twice would say nothing more.
	var order []string

	missing := map[string][]string{}

	for _, command := range []struct {
		field string
		line  string
	}{
		{settings.FieldLocalStart, commands.Start},
		{settings.FieldLocalUpdate, commands.Update},
	} {
		field := settings.BlockLocal + "." + command.field

		program, named := programOf(command.line)
		if !named {
			return append(results, domain.LocalRunnable.Fail(
				field+" runs nothing", mo.Some(equipRemedy)))
		}

		found, where, err := d.programExists(dir, program)
		if err != nil {
			return append(results, domain.LocalRunnable.Fail(
				fmt.Sprintf("Cannot stat %s: %v", program, err), mo.None[string]()))
		}

		if found {
			continue
		}

		key := program + " is not " + where
		if _, seen := missing[key]; !seen {
			order = append(order, key)
		}

		missing[key] = append(missing[key], field)
	}

	if len(order) == 0 {
		return append(results, domain.LocalRunnable.Pass())
	}

	lines := make([]string, 0, len(order))
	for _, key := range order {
		lines = append(lines, key+" ("+strings.Join(missing[key], ", ")+")")
	}

	return append(results, domain.LocalRunnable.Fail(strings.Join(lines, "; "), mo.Some(equipRemedy)))
}

// programOf is the program a command line runs: its first word, after any leading environment
// assignments. A line with no such word runs nothing.
func programOf(line string) (string, bool) {
	for _, word := range strings.Fields(line) {
		if assignment.MatchString(word) {
			continue
		}

		return word, true
	}

	return "", false
}

// programExists looks for a program where the shell would: in the project when the name carries
// a path, on PATH when it is bare. The second return is where it looked, for the sentence that says
// it was not there.
func (d *Diagnose) programExists(dir, program string) (bool, string, error) {
	if !strings.Contains(program, "/") {
		return d.runner.LookPath(program).IsPresent(), "on PATH", nil
	}

	path := program
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, program)
	}

	found, err := d.files.Exists(path)

	return found, "in the project", err
}

// settingsDocument is the settings file decoded as the generic document the shared field tables
// read. The file has already been read and found complete by the settings group, so anything
// unexpected here is nothing to report a second time — it is simply no answer, and the check that
// asked is skipped.
func (d *Diagnose) settingsDocument(dir string) (settings.Document, bool) {
	data, err := d.files.ReadFile(filepath.Join(dir, ".codefall", "settings.json"))
	if err != nil {
		return nil, false
	}

	var doc settings.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, false
	}

	return doc, true
}
