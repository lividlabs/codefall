package application

import (
	"errors"
	"fmt"
	"io/fs"
)

// moveSettings moves the settings file an earlier release wrote under its former name,
// .codefall/settings.json, to the name every reader uses now, and reports whether it moved it.
// The caller says whether .codefall/project.json is already there: a project with both files keeps
// project.json, which is the file this binary owns and reads, and the old one is left for the
// person to remove — the settings step says so. A project with neither file has nothing to move and
// the step writes its settings from the run's answers.
//
// The bytes are moved rather than the file being decoded and encoded again, for the reason the
// schema and harness edits splice theirs: re-encoding would reorder every key and drop whatever a
// project had added. The edits the settings step makes after the move — a harness spelling, an
// added harness, a $schema URL — are the same ones it makes to settings that were already under the
// new name.
func (i *Initialize) moveSettings(dir string, projectExists bool) (bool, error) {
	if projectExists {
		return false, nil
	}

	data, err := i.files.ReadFile(formerSettingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("read %s: %w", formerSettingsName, err)
	}

	if err := i.files.WriteFile(settingsPath(dir), data); err != nil {
		return false, fmt.Errorf("write %s: %w", settingsName, err)
	}

	if err := i.files.Remove(formerSettingsPath(dir)); err != nil {
		return false, fmt.Errorf("remove %s: %w", formerSettingsName, err)
	}

	return true, nil
}
