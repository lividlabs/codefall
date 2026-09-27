// Package version reads and orders the release versions codefall records: the tag GoReleaser stamps
// into a binary and the string init writes into .codefall/manifest.json. It is what upgrade uses to
// say which releases lie between an install and the binary running now.
//
// It is a pure shared module: it imports the standard library and nothing else, which is what lets
// domain/ and application/ name it (ADR-003). Adding a dependency here breaks that permission and
// fails the pure-shared-modules rule in .golangci.yml.
//
// Only a release is a version here. A development build reports `0.19.0-dev (abc1234)`, and a
// checkout with no tag below it reports `dev`; neither names a point in the release history, so
// neither parses. That is deliberate: a range with a development build at either end has no
// releases to list, and a caller is told so rather than handed a guess.
package version

import (
	"strconv"
	"strings"
)

// Version is one release: the three numbers of its tag.
type Version struct {
	Major, Minor, Patch int
}

// Parse reads a release version — `0.19.0`, or `v0.19.0` as a tag is spelled — and reports whether
// it read one. Anything after the three numbers, a prerelease, a build note, a development marker,
// is not a release and does not parse.
func Parse(text string) (Version, bool) {
	text = strings.TrimPrefix(strings.TrimSpace(text), "v")

	parts := strings.Split(text, ".")
	if len(parts) != 3 {
		return Version{}, false
	}

	numbers := make([]int, 0, len(parts))

	for _, part := range parts {
		if part == "" || strings.TrimLeft(part, "0123456789") != "" {
			return Version{}, false
		}

		number, err := strconv.Atoi(part)
		if err != nil {
			return Version{}, false
		}

		numbers = append(numbers, number)
	}

	return Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}, true
}

// Compare orders two versions: negative when a is the earlier release, zero when they are the same,
// positive when a is the later one.
func Compare(a, b Version) int {
	switch {
	case a.Major != b.Major:
		return a.Major - b.Major
	case a.Minor != b.Minor:
		return a.Minor - b.Minor
	default:
		return a.Patch - b.Patch
	}
}

// String is the version as a tag is spelled without its prefix, which is how the manifest and the
// changelog both write it.
func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor) + "." + strconv.Itoa(v.Patch)
}
