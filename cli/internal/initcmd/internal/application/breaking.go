package application

import (
	"slices"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/version"
)

// BreakingChanges reports the breaking changes a project upgrading from what its manifest records to
// the binary's own version has to hear about first: every release later than the recorded version
// and no later than the binary's that recorded any, earliest first. None means the range could not
// be determined — no manifest, a record with no version, a version that is not a release on either
// end — and the caller says so rather than printing nothing as if there were none (ADR-010).
//
// The recorded version is the earliest a finished run wrote for any harness or for .codefall/, because
// the oldest install is the one that crosses the most releases. A record carrying a version that is
// not a release, a development build for instance, makes the range undeterminable rather than being
// skipped: the run that wrote it was not a release either, and nothing says which release it was
// nearest to.
func (i *Initialize) BreakingChanges(dir, binary string) (mo.Option[[]BreakingRelease], error) {
	to, ok := version.Parse(binary)
	if !ok {
		return mo.None[[]BreakingRelease](), nil
	}

	recorded, read, err := i.recordedManifest(dir)
	if err != nil {
		return mo.None[[]BreakingRelease](), err
	}

	if !read {
		return mo.None[[]BreakingRelease](), nil
	}

	from, ok := earliestRecorded(recorded.Versions(), recorded.Shared.Version)
	if !ok {
		return mo.None[[]BreakingRelease](), nil
	}

	releases, err := i.changes.BreakingChanges()
	if err != nil {
		return mo.None[[]BreakingRelease](), err
	}

	between := make([]BreakingRelease, 0, len(releases))

	for _, release := range releases {
		at, ok := version.Parse(release.Version)
		if !ok {
			continue
		}

		if version.Compare(at, from) > 0 && version.Compare(at, to) <= 0 {
			between = append(between, release)
		}
	}

	slices.SortFunc(between, func(a, b BreakingRelease) int {
		left, _ := version.Parse(a.Version)
		right, _ := version.Parse(b.Version)

		return version.Compare(left, right)
	})

	return mo.Some(between), nil
}

// earliestRecorded is the earliest release among the versions a manifest records, or false when it
// records none or records one that is not a release.
func earliestRecorded(perHarness map[string]string, shared string) (version.Version, bool) {
	var (
		earliest version.Version
		found    bool
	)

	consider := func(text string) bool {
		if text == "" {
			return true
		}

		parsed, ok := version.Parse(text)
		if !ok {
			return false
		}

		if !found || version.Compare(parsed, earliest) < 0 {
			earliest, found = parsed, true
		}

		return true
	}

	for _, text := range perHarness {
		if !consider(text) {
			return version.Version{}, false
		}
	}

	if !consider(shared) {
		return version.Version{}, false
	}

	return earliest, found
}
