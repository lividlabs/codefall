// Package manifest is the one definition of the .codefall/manifest.json format: what finished init
// runs recorded, an entry per harness and an entry for what a run writes once. Both components need
// it — initcmd writes the file and doctor
// reports on what it says — so it lives here rather than in either one's domain (ADR-003).
//
// It is a pure shared module: it imports the standard library and the pure harness module, which is
// what lets domain/ and application/ name it. Adding a dependency here breaks that permission and fails the
// pure-shared-modules rule in .golangci.yml.
//
// What belongs here is the format. What a run installs, and what to do about an entry the settings no
// longer name, belong to the component that decides it.
package manifest

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
)

// Name is where the record lives, in .codefall/ beside settings.json. It is written once every step
// of a run has succeeded, so what it says is an install that finished.
const Name = ".codefall/manifest.json"

// Document is a decoded manifest: one entry per harness a finished run installed for, and one entry
// for what that run wrote once for the whole project.
//
// The split follows the install layout (ADR-006). The skills are copied into each harness's own
// skills directory, so which harness they were written for is a fact worth recording; the files
// reached by a path codefall writes go to .codefall/ once, whatever harnesses the run was for, and
// belong to no harness at all. Recording them in every harness's entry would say each install wrote
// them, and the record would then disagree with itself about who owns the copy.
type Document struct {
	Harnesses map[string]Install `json:"harnesses"`
	// Shared is .codefall/: the files the most recent finished run wrote there. An entry carrying no
	// version is a file written before this field existed, and says nothing.
	Shared Install `json:"shared"`
}

// Install is one entry: the binary that wrote the files, and the files, relative to the directory
// init installed in.
type Install struct {
	Version string   `json:"version"`
	Files   []string `json:"files"`
}

// Decode reads a manifest. A file written before harnesses were recorded per install names none of
// them and decodes cleanly to an empty record rather than an error: a reader then knows nothing about
// what is installed, which is the same position no file at all leaves it in.
func Decode(data []byte) (Document, error) {
	var document Document

	if err := json.Unmarshal(data, &document); err != nil {
		return Document{}, fmt.Errorf("decode %s: %w", Name, err)
	}

	return document, nil
}

// Encode renders a manifest as the bytes that go in the file, at the indent a person would have
// written by hand so a diff reads a line at a time.
func Encode(document Document) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", Name, err)
	}

	return data, nil
}

// Current returns the record with every entry filed under a spelling a harness had before it was
// named for its binary moved to the name it has now, and the former spellings it moved, sorted. The
// receiver is left as it was.
//
// When a record holds both spellings of one harness, the entry under the current name is kept: only
// a binary that already knew the new name could have written it, so it is the later of the two.
func (d Document) Current() (Document, []string) {
	current := Document{Harnesses: make(map[string]Install, len(d.Harnesses)), Shared: d.Shared}

	var moved []string

	for _, name := range slices.Sorted(maps.Keys(d.Harnesses)) {
		renamed, former := harness.Renamed(name).Get()
		if !former {
			current.Harnesses[name] = d.Harnesses[name]

			continue
		}

		moved = append(moved, name)

		if _, recorded := d.Harnesses[renamed]; !recorded {
			current.Harnesses[renamed] = d.Harnesses[name]
		}
	}

	if d.Harnesses == nil {
		current.Harnesses = nil
	}

	return current, moved
}

// Recorded returns the harnesses the manifest names, sorted. An entry carrying no version records
// nothing a reader can act on — which is what a hand-edited file looks like — so it is left out.
func (d Document) Recorded() []string {
	recorded := make([]string, 0, len(d.Harnesses))

	for name, install := range d.Harnesses {
		if install.Version != "" {
			recorded = append(recorded, name)
		}
	}

	slices.Sort(recorded)

	return recorded
}

// Versions is the version each harness was installed at, for a caller holding them against a binary's
// own tag. An entry carrying no version is left out, for the reason Recorded leaves it out.
func (d Document) Versions() map[string]string {
	versions := make(map[string]string, len(d.Harnesses))

	for name, install := range d.Harnesses {
		if install.Version != "" {
			versions[name] = install.Version
		}
	}

	return versions
}
