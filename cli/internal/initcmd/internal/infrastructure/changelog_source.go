package infrastructure

import (
	"bufio"
	"bytes"
	"regexp"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/application"
)

// ChangeLogSource reads the breaking changes out of the changelog release-please writes. It parses
// one shape and one shape only: a release opens with `## [x.y.z](…)` and its breaking changes sit as
// `* ` bullets under a `### ⚠ BREAKING CHANGES` heading, until the next heading. A heading the tool
// does not write is ignored, and the hand-written entries above the first release carry no such
// section, so they contribute nothing.
type ChangeLogSource struct {
	text []byte
}

// NewChangeLogSource builds the gateway over the changelog's bytes. The text is injected so tests
// can drive a fixture, and so the file the binary embeds is the composition root's to hand over.
func NewChangeLogSource(text []byte) *ChangeLogSource {
	return &ChangeLogSource{text: text}
}

// A release heading, as release-please writes it: the version in brackets, linked to the compare
// view. The prefix a tag carries is not in the heading, and the version is what the manifest records.
var releaseHeading = regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)\]`)

// The heading release-please gives the section, warning sign included.
const breakingHeading = "### ⚠ BREAKING CHANGES"

// BreakingChanges returns every release that recorded breaking changes, with its notes, in the order
// the changelog lists them, which is newest first. A release with the heading and no bullets under
// it recorded none and is left out.
func (c *ChangeLogSource) BreakingChanges() ([]application.BreakingRelease, error) {
	var (
		releases []application.BreakingRelease
		current  *application.BreakingRelease
		inBlock  bool
	)

	flush := func() {
		if current != nil && len(current.Notes) > 0 {
			releases = append(releases, *current)
		}

		current = nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(c.text))
	// A note can run long: release-please writes each footer as one line however long it is.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case releaseHeading.MatchString(line):
			flush()

			current = &application.BreakingRelease{Version: releaseHeading.FindStringSubmatch(line)[1]}
			inBlock = false
		case strings.HasPrefix(line, "## "):
			// A heading at the release level that is not a release ends whatever release was open.
			flush()

			inBlock = false
		case strings.TrimSpace(line) == breakingHeading:
			inBlock = current != nil
		case strings.HasPrefix(line, "### "):
			inBlock = false
		case inBlock && strings.HasPrefix(line, "* "):
			current.Notes = append(current.Notes, strings.TrimSpace(strings.TrimPrefix(line, "* ")))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	flush()

	return releases, nil
}
