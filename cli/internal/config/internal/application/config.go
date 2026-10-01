// Package application holds `codefall config`'s one use case: read the effective configuration, and
// change the agents list, its narrower orders, or the persona one command at a time. It owns the
// gateway interface it needs and depends on nothing but the standard library, mo, config's own
// domain, and the pure shared modules that define the two files' formats.
package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/shared/settings"
	"github.com/lividlabs/codefall/cli/internal/shared/userfile"
)

// FileSystem is config's view of the working directory (one gateway role): it reads the settings,
// the user file, and .gitignore, and writes them back.
type FileSystem interface {
	// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist).
	ReadFile(path string) ([]byte, error)
	// WriteFile writes data to path, replacing whatever was there.
	WriteFile(path string, data []byte) error
}

// settingsName is the settings file as a person reads its path, relative to the directory that holds
// .codefall/.
const settingsName = ".codefall/settings.json"

// errNotSetUp is what every write refuses a directory with no settings with. The settings are what
// init writes first, and a project without them has nothing for config to change.
var errNotSetUp = errors.New(settingsName + " not found; run codefall init first")

// Config reads and writes the two files a person configures codefall through: the checked-in
// .codefall/settings.json, for the agents, and the per-user .codefall/user.json, for the persona.
// It never prompts; every value arrives as an argument, so it runs the same under a script.
type Config struct {
	files FileSystem
}

// NewConfig builds the use case over its gateway.
func NewConfig(files FileSystem) *Config {
	return &Config{files: files}
}

// projectSettings is the settings file as it was read: its text, which a write splices into, and its
// decoded document, which the settings module reads and validates.
type projectSettings struct {
	data []byte
	doc  settings.Document
}

// readSettings reads and decodes .codefall/settings.json, refusing a directory that has none.
func (c *Config) readSettings(dir string) (projectSettings, error) {
	data, err := c.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return projectSettings{}, errNotSetUp
	case err != nil:
		return projectSettings{}, fmt.Errorf("read %s: %w", settingsName, err)
	}

	doc, err := decodeSettings(data)
	if err != nil {
		return projectSettings{}, err
	}

	return projectSettings{data: data, doc: doc}, nil
}

// decodeSettings decodes a settings file's text into the generic document the settings module reads.
func decodeSettings(data []byte) (settings.Document, error) {
	var doc settings.Document

	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if doc == nil {
		return nil, fmt.Errorf("decode %s: %w", settingsName, errNotObject)
	}

	return doc, nil
}

// validSettings refuses settings the settings module would not accept, in its own words. A write to
// a file that is already invalid would be built on a reading of it that every other reader disagrees
// with, so the person fixes the file first; doctor reports the same problems.
func validSettings(doc settings.Document, what string) error {
	problems := settings.Validate(doc)
	if len(problems) == 0 {
		return nil
	}

	return fmt.Errorf("%s: %s", what, strings.Join(problems, "; "))
}

// writeSettings writes a settings file's new text, only when it is settings the module accepts. A
// result it would refuse leaves the file exactly as it was and says why in the module's words.
func (c *Config) writeSettings(dir string, body []byte) error {
	doc, err := decodeSettings(body)
	if err != nil {
		return err
	}

	if err := validSettings(doc, "that change would leave "+settingsName+" invalid, so nothing was changed"); err != nil {
		return err
	}

	if err := c.files.WriteFile(settingsPath(dir), body); err != nil {
		return fmt.Errorf("write %s: %w", settingsName, err)
	}

	return nil
}

// readUserFile reads and decodes .codefall/user.json, or None when there is none.
func (c *Config) readUserFile(dir string) (mo.Option[[]byte], userfile.Document, error) {
	data, err := c.files.ReadFile(userFilePath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return mo.None[[]byte](), nil, nil
	case err != nil:
		return mo.None[[]byte](), nil, fmt.Errorf("read %s: %w", userfile.Name, err)
	}

	doc, err := decodeUserFile(data)
	if err != nil {
		return mo.None[[]byte](), nil, err
	}

	return mo.Some(data), doc, nil
}

// decodeUserFile decodes a user file's text into the generic document the user file module reads.
func decodeUserFile(data []byte) (userfile.Document, error) {
	var doc userfile.Document

	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode %s: %w", userfile.Name, err)
	}

	if doc == nil {
		return nil, fmt.Errorf("decode %s: %w", userfile.Name, errNotObject)
	}

	return doc, nil
}

func settingsPath(dir string) string {
	return filepath.Join(dir, filepath.FromSlash(settingsName))
}

func userFilePath(dir string) string {
	return filepath.Join(dir, filepath.FromSlash(userfile.Name))
}

func gitIgnorePath(dir string) string {
	return filepath.Join(dir, settings.GitIgnoreName)
}
