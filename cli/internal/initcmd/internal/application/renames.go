package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// FormerHarnessNames reports the spellings .codefall/settings.json and .codefall/manifest.json still
// use for a harness that has since been named for its binary, sorted and without repeats. Presentation
// asks before it decides a rerun has nothing to do: a project whose files carry an old spelling has a
// rewrite waiting, however current the version that installed it.
//
// A file that is not there has no spelling to report. A settings file that cannot be decoded is an
// error, as it is everywhere else init reads one.
func (i *Initialize) FormerHarnessNames(dir string) ([]string, error) {
	var found []string

	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", settingsName, err)
	default:
		_, renamed, err := respelledSettings(data)
		if err != nil {
			return nil, err
		}

		found = append(found, renamed...)
	}

	recorded, read, err := i.recordedManifest(dir)
	if err != nil {
		return nil, err
	}

	if read {
		_, renamed := recorded.Current()
		found = append(found, renamed...)
	}

	return slices.Compact(slices.Sorted(slices.Values(found))), nil
}

// respell rewrites the old spellings of a harness in .codefall/settings.json and
// .codefall/manifest.json to the names the harnesses have now, and returns the sentence the settings
// step reports about it, or "" when neither file carried one.
//
// Settings are rewritten only where the step is not already writing them whole: a run that encodes
// the file from its answers has nothing old left in it. The manifest is rewritten here rather than at
// the end of the run because the rename claims nothing about an install: the entry keeps the version
// and the files it had, under the name the settings now use.
func (i *Initialize) respell(dir string, settingsToo bool) (string, error) {
	var edits []respelling

	if settingsToo {
		renamed, err := i.respellSettings(dir)
		if err != nil {
			return "", err
		}

		if len(renamed) > 0 {
			edits = append(edits, respelling{file: settingsName, formers: renamed})
		}
	}

	renamed, err := i.respellManifest(dir)
	if err != nil {
		return "", err
	}

	if len(renamed) > 0 {
		edits = append(edits, respelling{file: manifest.Name, formers: renamed})
	}

	return describeRespelling(edits), nil
}

// respelling is one file's rename: the former spellings it carried.
type respelling struct {
	file    string
	formers []string
}

// describeRespelling is the sentence a rename reports. Two files that carried the same spellings are
// one clause, which is what a project that was set up once and never edited by hand looks like.
func describeRespelling(edits []respelling) string {
	if len(edits) == 2 && slices.Equal(edits[0].formers, edits[1].formers) {
		return fmt.Sprintf("renamed harness %s in %s and %s", renames(edits[0].formers), edits[0].file, edits[1].file)
	}

	clauses := make([]string, 0, len(edits))
	for _, edit := range edits {
		clauses = append(clauses, fmt.Sprintf("renamed harness %s in %s", renames(edit.formers), edit.file))
	}

	return strings.Join(clauses, "; ")
}

// renames is "claude-code to claude and antigravity to agy": each former spelling and the name the
// harness has now.
func renames(formers []string) string {
	pairs := make([]string, 0, len(formers))
	for _, former := range formers {
		pairs = append(pairs, former+" to "+harness.Renamed(former).OrElse(former))
	}

	return sentenceList(pairs)
}

// respellSettings rewrites the old spellings in the settings file and returns the ones it rewrote. A
// file that is not there has none.
func (i *Initialize) respellSettings(dir string) ([]string, error) {
	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read %s: %w", settingsName, err)
	}

	body, renamed, err := respelledSettings(data)
	if err != nil || len(renamed) == 0 {
		return nil, err
	}

	if err := i.files.WriteFile(settingsPath(dir), body); err != nil {
		return nil, fmt.Errorf("write %s: %w", settingsName, err)
	}

	return renamed, nil
}

// respellManifest rewrites the manifest with every entry under the name its harness has now, and
// returns the old spellings it moved. A manifest that is not there has none.
func (i *Initialize) respellManifest(dir string) ([]string, error) {
	recorded, read, err := i.recordedManifest(dir)
	if err != nil || !read {
		return nil, err
	}

	current, renamed := recorded.Current()
	if len(renamed) == 0 {
		return nil, nil
	}

	body, err := manifest.Encode(current)
	if err != nil {
		return nil, err
	}

	if err := i.files.WriteFile(filepath.Join(dir, manifest.Name), body); err != nil {
		return nil, fmt.Errorf("write %s: %w", manifest.Name, err)
	}

	return renamed, nil
}

// respelledSettings is the settings file's text with every old spelling in the harnesses list
// replaced by the name the harness has now, and the old spellings it replaced, sorted and without
// repeats.
//
// The names are spliced into the file's text rather than the document being decoded and encoded
// again, for the reason the testing step splices its block (declareTestRoot): re-encoding would
// reorder every key and drop whatever a project had added. Only the bytes of the names change. A
// harnesses field that is not a list has nothing to rename, and validation is what says so.
func respelledSettings(data []byte) ([]byte, []string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	if start, err := decoder.Token(); err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return data, nil, nil
	}

	var spans []nameSpan

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		if key != settings.FieldHarnesses {
			continue
		}

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		end := int(decoder.InputOffset())

		found, err := formerNameSpans(value, end-len(value))
		if err != nil {
			return nil, nil, err
		}

		spans = append(spans, found...)
	}

	if len(spans) == 0 {
		return data, nil, nil
	}

	body := slices.Clone(data)
	renamed := make([]string, 0, len(spans))

	// From the end backwards, so a replacement of a different length leaves the spans before it
	// where they were.
	for _, span := range slices.Backward(spans) {
		quoted, err := json.Marshal(span.current)
		if err != nil {
			return nil, nil, fmt.Errorf("encode %q: %w", span.current, err)
		}

		body = slices.Concat(body[:span.from], quoted, body[span.to:])
		renamed = append(renamed, span.former)
	}

	return body, slices.Compact(slices.Sorted(slices.Values(renamed))), nil
}

// nameSpan is where one old spelling sits in the settings file's text: the bytes of the quoted
// string, from its opening quote to just past its closing one.
type nameSpan struct {
	from, to int
	former   string
	current  string
}

// formerNameSpans finds the old spellings in the harnesses value, which starts at offset in the file.
func formerNameSpans(value json.RawMessage, offset int) ([]nameSpan, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))

	if start, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '[' {
		return nil, nil
	}

	var spans []nameSpan

	for decoder.More() {
		before := int(decoder.InputOffset())

		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		var name string
		if err := json.Unmarshal(element, &name); err != nil {
			// Not a name: validation says so, and there is nothing here to rename.
			continue
		}

		current, former := harness.Renamed(name).Get()
		if !former {
			continue
		}

		// Between the previous token and this element there is only space and a comma, so the
		// element's own bytes end where the decoder now stands.
		end := int(decoder.InputOffset())
		if end-len(element) < before {
			return nil, fmt.Errorf("decode %s: harness %q is not where it was read", settingsName, name)
		}

		spans = append(spans, nameSpan{
			from: offset + end - len(element), to: offset + end, former: name, current: current,
		})
	}

	return spans, nil
}
