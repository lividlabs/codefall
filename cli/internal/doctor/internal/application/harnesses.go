package application

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/doctor/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// initRemedy is what to do about a harness the project chose and codefall was never run for.
const initRemedy = "codefall init"

// renameRemedy is what to do about a harness recorded under the spelling it had before it was named
// for its binary: init rewrites both files on its next run.
const renameRemedy = "run codefall init again, which rewrites them"

// settingsName is the settings file as a person reads it in a report.
const settingsName = ".codefall/settings.json"

// harnesses runs checks 9 to 11, all of which read .codefall/manifest.json, so the file is read once
// here and handed to each of them.
//
// The two checks after the first read both files under the names the harnesses have now. An old
// spelling is the first check's to report, and read as written it would also look like a harness the
// project chose and never installed, or an install left over from one it dropped.
func (d *Diagnose) harnesses(_ context.Context, dir string, results []domain.Result) []domain.Result {
	// Settings doctor has already complained about say nothing about which harnesses were chosen, and
	// a second complaint about the same file would be noise. The checks are absent from the report
	// rather than present with a status of their own, which is what every other skipped check does.
	if !settingsAreComplete(results) {
		return results
	}

	written, declared := d.chosenHarnesses(dir)
	if !declared {
		return results
	}

	chosen, settingsFormers := currentNames(written)

	recorded, read := d.recordedManifest(dir)
	current, manifestFormers := recorded.Current()

	results = append(results, namesAreCurrent(settingsFormers, manifestFormers))

	return d.leftOver(current, read, chosen, d.installed(dir, current, chosen, results))
}

// currentNames is the harnesses the settings name, each under the name it has now and once, in the
// order the settings give them, and the former spellings among them, sorted.
func currentNames(names []string) ([]string, []string) {
	current := make([]string, 0, len(names))

	var formers []string

	for _, name := range names {
		renamed, former := harness.Renamed(name).Get()
		if former {
			formers = append(formers, name)
			name = renamed
		}

		if !slices.Contains(current, name) {
			current = append(current, name)
		}
	}

	return current, slices.Compact(slices.Sorted(slices.Values(formers)))
}

// namesAreCurrent is check 9: the settings and the manifest name each harness the way codefall does
// now. A harness is named for its binary, and a project set up before two of them were renamed still
// records the spellings they had.
//
// It warns rather than fails, because codefall still reads the old spellings and nothing stops
// working. What goes wrong is that a script reading the settings finds a name that is not a command
// it can run, and init rewrites both files the next time it runs.
func namesAreCurrent(settingsFormers, manifestFormers []string) domain.Result {
	var found []string

	switch {
	case len(settingsFormers) > 0 && slices.Equal(settingsFormers, manifestFormers):
		found = append(found, fmt.Sprintf("%s and %s name %s", settingsName, manifest.Name, renames(settingsFormers)))
	default:
		if len(settingsFormers) > 0 {
			found = append(found, fmt.Sprintf("%s names %s", settingsName, renames(settingsFormers)))
		}

		if len(manifestFormers) > 0 {
			found = append(found, fmt.Sprintf("%s names %s", manifest.Name, renames(manifestFormers)))
		}
	}

	if len(found) == 0 {
		return domain.HarnessNames.Pass()
	}

	return domain.HarnessNames.Warn(strings.Join(found, "; "), mo.Some(renameRemedy))
}

// renames is "claude-code (now claude) and antigravity (now agy)": each former spelling, and the name
// the harness has now.
func renames(formers []string) string {
	pairs := make([]string, 0, len(formers))
	for _, former := range formers {
		pairs = append(pairs, fmt.Sprintf("%s (now %s)", former, harness.Renamed(former).OrElse(former)))
	}

	if len(pairs) < 2 {
		return strings.Join(pairs, "")
	}

	return strings.Join(pairs[:len(pairs)-1], ", ") + " and " + pairs[len(pairs)-1]
}

// installed is check 10: the files a finished run recorded for each chosen harness are still where it
// wrote them.
//
// What this used to stat was hooks/shared/ under each harness's own skills directory, which the
// install layout moved: those scripts are written once, to .codefall/, whatever harnesses a run was
// for, so nothing under a skills directory says which harness it was installed for any more
// (ADR-006). What is still per harness is the manifest's entry — a finished run records, for each
// harness it installed for, every file it wrote into the directory that harness reads.
//
// Reading the record is also what keeps the check clear of a skill's name. The names come out of the
// file the last run wrote rather than out of this code, so a skill renamed between two releases
// changes both at once, where a literal here would fail on a project that is set up correctly.
//
// It fails rather than warns. A harness the project chose and codefall never installed for has no
// skills there, so every codefall verb is missing in a harness somebody is using. A manifest that
// cannot be read records no install and reports every chosen harness, which is the same answer and
// the same remedy: init rewrites the file it could not read.
func (d *Diagnose) installed(
	dir string, recorded manifest.Document, chosen []string, results []domain.Result,
) []domain.Result {
	var missing []string

	for _, name := range chosen {
		files := recorded.Harnesses[name].Files
		if len(files) == 0 {
			missing = append(missing, name)

			continue
		}

		gone, refused, err := d.firstMissing(dir, files)
		if err != nil {
			return append(results, domain.HarnessesInstalled.Fail(
				fmt.Sprintf("Cannot stat %s: %v", refused, err), mo.None[string]()))
		}

		if gone {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return append(results, domain.HarnessesInstalled.Fail(
			"codefall is not installed for "+strings.Join(missing, ", "), mo.Some(initRemedy)))
	}

	return append(results, domain.HarnessesInstalled.PassWithDetail(strings.Join(chosen, ", ")))
}

// firstMissing reports whether any of the recorded files has gone from the project. A half-written
// install is not an install: the run that wrote the record wrote all of them, and `codefall init` is
// the remedy either way. A file system that refuses a stat returns the path it refused, so the
// report can name it.
func (d *Diagnose) firstMissing(dir string, files []string) (bool, string, error) {
	for _, file := range files {
		path := filepath.Join(dir, file)

		there, err := d.files.Exists(path)
		if err != nil {
			return false, path, err
		}

		if !there {
			return true, "", nil
		}
	}

	return false, "", nil
}

// leftOver is check 11: nothing codefall installed is still sitting there for a harness the settings
// no longer name. A project set up for two harnesses that later drops one keeps everything codefall
// wrote for it, because codefall only ever writes what it owns and never deletes.
//
// It warns rather than fails, because nothing stops working. What goes wrong is quieter: an agent
// reading the skills directory finds codefall's skills for a harness the project has dropped, and the
// hook codefall registered for that harness still runs.
//
// The remedy points at the manifest rather than naming a directory to delete. The manifest is what
// says which files codefall wrote for that harness, and harnesses share directories — four of the
// five read .agents/ — so the directory a dropped harness read may still be another's. Which of those
// files are safe to remove is the reader's judgement, not doctor's.
func (d *Diagnose) leftOver(
	recorded manifest.Document, read bool, chosen []string, results []domain.Result,
) []domain.Result {
	// A manifest that is missing, unreadable, or not a manifest any more records no install to be
	// left over. Init fails on the same file the next time it writes one, so nothing goes unsaid.
	if !read {
		return results
	}

	var dropped []string

	for _, name := range recorded.Recorded() {
		if !slices.Contains(chosen, name) {
			dropped = append(dropped, name)
		}
	}

	if len(dropped) == 0 {
		return append(results, domain.HarnessesLeftOver.Pass())
	}

	return append(results, domain.HarnessesLeftOver.Warn(
		fmt.Sprintf("codefall is still installed for %s, which the settings no longer name",
			strings.Join(dropped, ", ")),
		mo.Some(fmt.Sprintf("remove the files %s lists for %s; codefall never deletes what it wrote",
			manifest.Name, strings.Join(dropped, ", ")))))
}

// recordedManifest is what a finished run left on record. Anything the reader cannot make sense of is
// no record rather than a second complaint: doctor has no check for the manifest's own shape, and the
// install check above has already reported whatever is missing on disk.
func (d *Diagnose) recordedManifest(dir string) (manifest.Document, bool) {
	data, err := d.files.ReadFile(filepath.Join(dir, manifest.Name))
	if err != nil {
		return manifest.Document{}, false
	}

	recorded, err := manifest.Decode(data)
	if err != nil {
		return manifest.Document{}, false
	}

	return recorded, true
}

// settingsAreComplete reports whether the settings group found the file complete. A check whose
// prerequisite failed is skipped, and the settings this check reads are that prerequisite.
func settingsAreComplete(results []domain.Result) bool {
	for _, result := range results {
		if result.Check.ID == domain.SettingsComplete.ID {
			return result.Status == domain.StatusPass
		}
	}

	return false
}

// chosenHarnesses is the harnesses the settings record. A document that cannot be read again, or
// names none, is no answer rather than a second complaint, and the check is skipped.
func (d *Diagnose) chosenHarnesses(dir string) ([]string, bool) {
	doc, read := d.settingsDocument(dir)
	if !read {
		return nil, false
	}

	chosen := settings.Harnesses(doc)
	if len(chosen) == 0 {
		return nil, false
	}

	return chosen, true
}
