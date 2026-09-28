package application

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/config/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/userfile"
)

// SetPersona records who the person at the keyboard works as in .codefall/user.json, and makes sure
// .gitignore keeps that file out of the repository. It returns one write per file, the user file's
// first.
//
// A missing user file is created holding the version and the persona. One that is there keeps every
// other key it holds and the layout it has; only the persona's value changes, and the version is
// added when the file has none. A result the user file module would not accept is refused and
// nothing is written.
//
// The .gitignore line is written first, so a failure between the two writes never leaves a user file
// git would pick up. It is the line init writes, by the same helper, and it is added whenever it is
// missing, whether or not the persona changed.
func (c *Config) SetPersona(dir, name string) ([]domain.Write, error) {
	persona, err := userfile.ParsePersona(name)
	if err != nil {
		return nil, err
	}

	if _, err := c.readSettings(dir); err != nil {
		return nil, err
	}

	existing, doc, err := c.readUserFile(dir)
	if err != nil {
		return nil, err
	}

	body, userWrite, err := userFileWith(existing, doc, persona)
	if err != nil {
		return nil, err
	}

	ignoreWrite, err := c.ignoreUserFile(dir)
	if err != nil {
		return nil, err
	}

	if userWrite.Changed {
		if err := c.files.WriteFile(userFilePath(dir), body); err != nil {
			return nil, fmt.Errorf("write %s: %w", userfile.Name, err)
		}
	}

	writes := []domain.Write{userWrite}
	if ignored, ok := ignoreWrite.Get(); ok {
		writes = append(writes, ignored)
	}

	return writes, nil
}

// userFileWith is the user file's new text and what writing it would do: a new file, a changed
// persona, or nothing when the file already says so.
func userFileWith(existing mo.Option[[]byte], doc userfile.Document, persona string) ([]byte, domain.Write, error) {
	data, present := existing.Get()
	if !present {
		body, err := encode(map[string]any{}, true, "")
		if err != nil {
			return nil, domain.Write{}, err
		}

		// The two keys are spliced in rather than encoded from a map, which would put persona first.
		body, err = withVersion(body)
		if err != nil {
			return nil, domain.Write{}, err
		}

		body, err = withPersona(body, persona)
		if err != nil {
			return nil, domain.Write{}, err
		}

		return append(body, '\n'), domain.Changed(fmt.Sprintf("wrote %s (persona: %s)", userfile.Name, persona)), nil
	}

	if namesPersona(doc) && userfile.Persona(doc) == persona && len(userfile.Validate(doc)) == 0 {
		return nil, domain.Unchanged(fmt.Sprintf("%s already says persona: %s", userfile.Name, persona)), nil
	}

	// A version that is null reads as missing, as the module has it, and gets the value a missing
	// one gets.
	hasVersion := doc[userfile.FieldVersion] != nil

	body, err := withPersona(data, persona)
	if err != nil {
		return nil, domain.Write{}, fmt.Errorf("decode %s: %w", userfile.Name, err)
	}

	if !hasVersion {
		if body, err = withVersion(body); err != nil {
			return nil, domain.Write{}, fmt.Errorf("decode %s: %w", userfile.Name, err)
		}
	}

	written, err := decodeUserFile(body)
	if err != nil {
		return nil, domain.Write{}, err
	}

	if problems := userfile.Validate(written); len(problems) > 0 {
		return nil, domain.Write{}, fmt.Errorf("%s is not valid, so nothing was changed: %s",
			userfile.Name, strings.Join(problems, "; "))
	}

	return body, domain.Changed(fmt.Sprintf("set persona to %s in %s", persona, userfile.Name)), nil
}

// namesPersona reports whether the document spells a persona out, as opposed to leaving the default
// to apply.
func namesPersona(doc userfile.Document) bool {
	named, _ := doc[userfile.FieldPersona].(string)

	return named != ""
}

func withPersona(data []byte, persona string) ([]byte, error) {
	return withField(data, userfile.FieldPersona, func(string) ([]byte, error) {
		return encode(persona, false, "")
	})
}

func withVersion(data []byte) ([]byte, error) {
	return withField(data, userfile.FieldVersion, func(string) ([]byte, error) {
		return encode(userfile.Version, false, "")
	})
}

// ignoreUserFile adds the user file's line to .gitignore when the file does not already name it, and
// says what that took; None means the line was already there, which is not worth a line of output.
func (c *Config) ignoreUserFile(dir string) (mo.Option[domain.Write], error) {
	existing := mo.None[string]()

	data, err := c.files.ReadFile(gitIgnorePath(dir))

	switch {
	case err == nil:
		existing = mo.Some(string(data))
	case !errors.Is(err, fs.ErrNotExist):
		return mo.None[domain.Write](), fmt.Errorf("read %s: %w", settings.GitIgnoreName, err)
	}

	body, added := settings.WithIgnoreLines(existing, []settings.IgnoreLine{
		{Entry: userfile.Name, Comment: userfile.GitIgnoreComment},
	})
	if len(added) == 0 {
		return mo.None[domain.Write](), nil
	}

	if err := c.files.WriteFile(gitIgnorePath(dir), []byte(body)); err != nil {
		return mo.None[domain.Write](), fmt.Errorf("write %s: %w", settings.GitIgnoreName, err)
	}

	if existing.IsAbsent() {
		return mo.Some(domain.Changed("wrote " + settings.GitIgnoreName)), nil
	}

	return mo.Some(domain.Changed(fmt.Sprintf("added %s to %s", userfile.Name, settings.GitIgnoreName))), nil
}
