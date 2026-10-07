// Package buildinfo says which build of codefall is running. A release reports the version
// GoReleaser stamped into it. Any other build reports the release it is based on, marked as a
// development build, with the commit it was built from, so a binary built from a checkout is never
// mistaken for a release — not even one built at a release tag.
//
// It is imported by the composition root, which hands the string to Fang for --version, by init's
// presentation layer, which records it in .codefall/manifest.json, and by update's infrastructure,
// which reads whether the running binary is a release.
package buildinfo

import (
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

// GoReleaser sets these at link time, from the -X flags in .goreleaser.yml; every other build leaves
// them empty. The linker ignores a -X flag that names no variable without saying so, which is how
// releases once shipped reporting no version at all — buildinfo_test.go holds the two in step.
var (
	version string
	commit  string
)

// shortCommit is how much of a commit hash is shown: git's own abbreviation, and Fang's.
const shortCommit = 7

// Version is what init records as the version that installed a project's files: the version of a
// release, and for any other build the same string --version prints, so rebuilding at another commit
// reads to init as a version change.
func Version() string {
	return read().version()
}

// Display is what --version prints: Version, plus the commit a release was built from.
func Display() string {
	return read().display()
}

// build is what the binary knows about how it was built.
type build struct {
	// release is the version of a release build, and empty for a development build.
	release string
	// base is the release a development build is based on, and empty when Go found no tag below it.
	base string
	// commit is the full hash of the commit the binary was built from, and empty when nothing
	// recorded it — `go run` does not.
	commit string
	// modified reports that the working tree had uncommitted changes when the binary was built.
	modified bool
}

func (b build) version() string {
	if b.release != "" {
		return b.release
	}

	name := "dev"
	if b.base != "" {
		name = b.base + "-dev"
	}

	var notes []string
	if b.commit != "" {
		notes = append(notes, abbreviate(b.commit))
	}

	if b.modified {
		notes = append(notes, "uncommitted changes")
	}

	if len(notes) == 0 {
		return name
	}

	return name + " (" + strings.Join(notes, ", ") + ")"
}

func (b build) display() string {
	if b.release != "" && b.commit != "" {
		return b.release + " (" + abbreviate(b.commit) + ")"
	}

	return b.version()
}

func abbreviate(hash string) string {
	if len(hash) > shortCommit {
		return hash[:shortCommit]
	}

	return hash
}

func read() build {
	info, _ := debug.ReadBuildInfo()

	return describe(version, commit, info)
}

// describe works out the build from what was stamped at link time and what the go command recorded.
// info is nil when the binary carries no build information.
func describe(stampedVersion, stampedCommit string, info *debug.BuildInfo) build {
	if stampedVersion != "" {
		return build{release: strings.TrimPrefix(stampedVersion, "v"), commit: stampedCommit}
	}

	if info == nil {
		return build{}
	}

	// A module sum means the go command fetched the module at a version rather than building a
	// checkout — `go install github.com/lividlabs/codefall/cli/cmd/codefall@v0.14.0` — so the
	// version is a published one, and nothing about a working tree was recorded.
	if info.Main.Sum != "" {
		return build{release: strings.TrimPrefix(info.Main.Version, "v")}
	}

	b := build{base: baseRelease(info.Main.Version)}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			b.commit = setting.Value
		case "vcs.modified":
			b.modified = setting.Value == "true"
		}
	}

	return b
}

// The version the go command records for a checkout is the tag at the commit, or a pseudo-version
// naming the tag below it (https://go.dev/ref/mod#pseudo-versions). The pseudo-version forms are
// matched first, because each is also a valid tag with a prerelease suffix.
var (
	// v0.14.1-0.20260921120000-abcdef123456: a commit after v0.14.0.
	afterRelease = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)-0\.\d{14}-[0-9a-f]+$`)
	// v0.15.0-rc.1.0.20260921120000-abcdef123456: a commit after v0.15.0-rc.1.
	afterPrerelease = regexp.MustCompile(`^v(\d+\.\d+\.\d+-.+)\.0\.\d{14}-[0-9a-f]+$`)
	// v0.0.0-20260921120000-abcdef123456: a commit with no tag below it.
	untagged = regexp.MustCompile(`^v\d+\.0\.0-\d{14}-[0-9a-f]+$`)
	// v0.14.0: the commit is the tag.
	tagged = regexp.MustCompile(`^v(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$`)
)

// baseRelease is the release a checkout's recorded version names, without its leading v, or empty
// when it names none — no tag below the commit, or "(devel)" when the go command recorded nothing.
func baseRelease(recorded string) string {
	// The +dirty suffix says what vcs.modified says, and that is read from there.
	recorded, _, _ = strings.Cut(recorded, "+")

	if m := afterRelease.FindStringSubmatch(recorded); m != nil {
		// The pseudo-version increments the patch number of the tag below it.
		patch, err := strconv.Atoi(m[3])
		if err != nil || patch == 0 {
			return ""
		}

		return m[1] + "." + m[2] + "." + strconv.Itoa(patch-1)
	}

	if m := afterPrerelease.FindStringSubmatch(recorded); m != nil {
		return m[1]
	}

	if untagged.MatchString(recorded) {
		return ""
	}

	if m := tagged.FindStringSubmatch(recorded); m != nil {
		return m[1]
	}

	return ""
}
