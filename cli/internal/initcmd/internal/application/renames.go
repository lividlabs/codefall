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

	"github.com/samber/mo"

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
		_, edits, err := respelledSettings(data)
		if err != nil {
			return nil, err
		}

		found = append(found, edits.formers()...)
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
// step reports about it, or "" when neither file carried one. An old spelling of a harness the file
// already names under its current name is removed rather than rewritten, so the file does not end up
// naming the harness twice.
//
// Settings are rewritten only where the step is not already writing them whole: a run that encodes
// the file from its answers has nothing old left in it. The manifest is rewritten here rather than at
// the end of the run because the rename claims nothing about an install: the entry keeps the version
// and the files it had, under the name the settings now use.
func (i *Initialize) respell(dir string, settingsToo bool) (string, error) {
	var edits []respelling

	if settingsToo {
		edited, err := i.respellSettings(dir)
		if err != nil {
			return "", err
		}

		if !edited.empty() {
			edits = append(edits, respelling{file: settingsName, harnessEdits: edited})
		}
	}

	edited, err := i.respellManifest(dir)
	if err != nil {
		return "", err
	}

	if !edited.empty() {
		edits = append(edits, respelling{file: manifest.Name, harnessEdits: edited})
	}

	return describeRespelling(edits), nil
}

// harnessEdits is what a rewrite did to one file's old spellings, each list sorted and without
// repeats: the ones it renamed to the current name, and the ones it removed because the file already
// named that harness under its current name.
type harnessEdits struct {
	renamed []string
	removed []string
}

func (e harnessEdits) empty() bool {
	return len(e.renamed) == 0 && len(e.removed) == 0
}

// formers is every old spelling the rewrite touched, sorted and without repeats.
func (e harnessEdits) formers() []string {
	return slices.Compact(slices.Sorted(slices.Values(slices.Concat(e.renamed, e.removed))))
}

func (e harnessEdits) equal(other harnessEdits) bool {
	return slices.Equal(e.renamed, other.renamed) && slices.Equal(e.removed, other.removed)
}

// respelling is one file's rewrite: the file, and what was done to the former spellings it carried.
type respelling struct {
	file string
	harnessEdits
}

// describeRespelling is the sentence a rewrite reports. Two files that were edited the same way are
// one clause for each kind of edit, which is what a project that was set up once and never edited by
// hand looks like.
func describeRespelling(edits []respelling) string {
	if len(edits) == 2 && edits[0].equal(edits[1].harnessEdits) {
		return strings.Join(editClauses(edits[0].harnessEdits, edits[0].file+" and "+edits[1].file), "; ")
	}

	var clauses []string
	for _, edit := range edits {
		clauses = append(clauses, editClauses(edit.harnessEdits, edit.file)...)
	}

	return strings.Join(clauses, "; ")
}

// editClauses is "renamed harness claude-code to claude in <files>" and "removed harness antigravity
// from <files>, already listed as agy", each only when the edit made one.
func editClauses(edits harnessEdits, files string) []string {
	var clauses []string

	if len(edits.renamed) > 0 {
		clauses = append(clauses, fmt.Sprintf("renamed harness %s in %s", renames(edits.renamed), files))
	}

	if len(edits.removed) > 0 {
		clauses = append(clauses, fmt.Sprintf("removed harness %s from %s, already listed as %s",
			sentenceList(edits.removed), files, sentenceList(currentNames(edits.removed))))
	}

	return clauses
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

// currentNames is the name each former spelling's harness has now, in the same order.
func currentNames(formers []string) []string {
	names := make([]string, 0, len(formers))
	for _, former := range formers {
		names = append(names, harness.Renamed(former).OrElse(former))
	}

	return names
}

// respellSettings rewrites the old spellings in the settings file and returns what it did to them. A
// file that is not there has none.
func (i *Initialize) respellSettings(dir string) (harnessEdits, error) {
	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return harnessEdits{}, nil
	case err != nil:
		return harnessEdits{}, fmt.Errorf("read %s: %w", settingsName, err)
	}

	body, edits, err := respelledSettings(data)
	if err != nil || edits.empty() {
		return harnessEdits{}, err
	}

	if err := i.files.WriteFile(settingsPath(dir), body); err != nil {
		return harnessEdits{}, fmt.Errorf("write %s: %w", settingsName, err)
	}

	return edits, nil
}

// respellManifest rewrites the manifest with every entry under the name its harness has now, and
// returns what it did to the old spellings. An entry under an old spelling whose harness the manifest
// also records under its current name is dropped rather than moved (manifest.Document.Current keeps
// the current one), and is reported as removed. A manifest that is not there has none.
func (i *Initialize) respellManifest(dir string) (harnessEdits, error) {
	recorded, read, err := i.recordedManifest(dir)
	if err != nil || !read {
		return harnessEdits{}, err
	}

	current, moved := recorded.Current()
	if len(moved) == 0 {
		return harnessEdits{}, nil
	}

	body, err := manifest.Encode(current)
	if err != nil {
		return harnessEdits{}, err
	}

	if err := i.files.WriteFile(filepath.Join(dir, manifest.Name), body); err != nil {
		return harnessEdits{}, fmt.Errorf("write %s: %w", manifest.Name, err)
	}

	var edits harnessEdits

	for _, former := range moved {
		if _, both := recorded.Harnesses[harness.Renamed(former).OrElse(former)]; both {
			edits.removed = append(edits.removed, former)
		} else {
			edits.renamed = append(edits.renamed, former)
		}
	}

	return edits, nil
}

// respelledSettings is the settings file's text with every old spelling in the harnesses list
// replaced by the name the harness has now, and what it did to the old spellings. An old spelling of
// a harness the list already names under its current name is removed rather than replaced, so the
// rewrite never leaves the list naming one harness twice.
//
// The names are spliced into the file's text rather than the document being decoded and encoded
// again, for the reason the testing step splices its block (declareTestRoot): re-encoding would
// reorder every key and drop whatever a project had added. Only the bytes of the names change, and a
// removed name takes one separator with it. A harnesses field that is not a list has nothing to
// rename, and validation is what says so.
func respelledSettings(data []byte) ([]byte, harnessEdits, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	if start, err := decoder.Token(); err != nil {
		return nil, harnessEdits{}, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '{' {
		return data, harnessEdits{}, nil
	}

	var spans []nameSpan

	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, harnessEdits{}, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, harnessEdits{}, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		if key != settings.FieldHarnesses {
			continue
		}

		// A raw value is the source's own bytes, so the offset it ends at locates where it starts.
		end := int(decoder.InputOffset())

		found, err := formerNameSpans(value, end-len(value))
		if err != nil {
			return nil, harnessEdits{}, err
		}

		spans = append(spans, found...)
	}

	if len(spans) == 0 {
		return data, harnessEdits{}, nil
	}

	slices.SortFunc(spans, func(a, b nameSpan) int { return a.from - b.from })

	body := slices.Clone(data)

	var edits harnessEdits

	// From the end backwards, so a replacement of a different length leaves the spans before it
	// where they were.
	for _, span := range slices.Backward(spans) {
		if span.remove {
			body = slices.Concat(body[:span.from], body[span.to:])
			edits.removed = append(edits.removed, span.former)

			continue
		}

		quoted, err := json.Marshal(span.current)
		if err != nil {
			return nil, harnessEdits{}, fmt.Errorf("encode %q: %w", span.current, err)
		}

		body = slices.Concat(body[:span.from], quoted, body[span.to:])
		edits.renamed = append(edits.renamed, span.former)
	}

	edits.renamed = slices.Compact(slices.Sorted(slices.Values(edits.renamed)))
	edits.removed = slices.Compact(slices.Sorted(slices.Values(edits.removed)))

	return body, edits, nil
}

// nameSpan is one edit to the settings file's text. A rename covers the quoted string, from its
// opening quote to just past its closing one, and puts the current name there. A removal covers
// the element and one separator beside it, and puts nothing there.
type nameSpan struct {
	from, to int
	former   string
	current  string
	remove   bool
}

// listElement is one element of the harnesses list: where its bytes sit in the file, and the name it
// holds when it is a string.
type listElement struct {
	from, to int
	name     mo.Option[string]
}

// formerNameSpans finds the edits the old spellings in the harnesses value need, the value starting
// at offset in the file.
//
// An old spelling is renamed unless its harness is already in the list under the current name, or
// an earlier old spelling of the same harness is being renamed to it; then it is removed. A removed
// element takes the separator before it, so the elements before it keep the layout they had. At the
// head of the list there is no element before, so a removed element there takes the separator after
// it instead, up to the next element's first byte.
func formerNameSpans(value json.RawMessage, offset int) ([]nameSpan, error) {
	elements, err := listElements(value, offset)
	if err != nil {
		return nil, err
	}

	listed := map[string]bool{}

	for _, element := range elements {
		if name, ok := element.name.Get(); ok && harness.Renamed(name).IsAbsent() {
			listed[name] = true
		}
	}

	var spans []nameSpan

	removed := make([]bool, len(elements))

	for index, element := range elements {
		name, ok := element.name.Get()
		if !ok {
			continue
		}

		current, former := harness.Renamed(name).Get()
		if !former {
			continue
		}

		if listed[current] {
			removed[index] = true

			continue
		}

		listed[current] = true

		spans = append(spans, nameSpan{from: element.from, to: element.to, former: name, current: current})
	}

	// An element is removed only beside one that names the same harness and stays, so some element
	// always stays.
	first := slices.Index(removed, false)

	for index, element := range elements {
		if !removed[index] {
			continue
		}

		span := nameSpan{former: element.name.MustGet(), remove: true}

		if index < first {
			span.from, span.to = element.from, elements[index+1].from
		} else {
			span.from, span.to = elements[index-1].to, element.to
		}

		spans = append(spans, span)
	}

	return spans, nil
}

// listElements reads the elements of the harnesses value, which starts at offset in the file. A value
// that is not a list has none.
func listElements(value json.RawMessage, offset int) ([]listElement, error) {
	decoder := json.NewDecoder(bytes.NewReader(value))

	if start, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("decode %s: %w", settingsName, err)
	} else if delim, ok := start.(json.Delim); !ok || delim != '[' {
		return nil, nil
	}

	var elements []listElement

	for decoder.More() {
		before := int(decoder.InputOffset())

		var element json.RawMessage
		if err := decoder.Decode(&element); err != nil {
			return nil, fmt.Errorf("decode %s: %w", settingsName, err)
		}

		// Between the previous token and this element there is only space and a comma, so the
		// element's own bytes end where the decoder now stands.
		end := int(decoder.InputOffset())
		if end-len(element) < before {
			return nil, fmt.Errorf("decode %s: harness list element is not where it was read", settingsName)
		}

		read := listElement{from: offset + end - len(element), to: offset + end, name: mo.None[string]()}

		// An element that is not a name is left where it is: validation says so.
		var name string
		if err := json.Unmarshal(element, &name); err == nil {
			read.name = mo.Some(name)
		}

		elements = append(elements, read)
	}

	return elements, nil
}
