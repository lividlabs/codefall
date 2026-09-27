// Package domain holds initcmd's entities and value objects: the settings a project is initialised
// with, the harnesses codefall knows about, and the steps a run is made of. It performs no IO and
// names no delivery, encoding, or infrastructure type (ADR-BASE-01).
//
// What .codefall/settings.json may contain is not initcmd's — doctor reads the same file — so the
// format lives in the pure shared module internal/shared/settings and the value objects here are
// built from its rules (ADR-003).
package domain

import (
	"errors"
	"fmt"
	"slices"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// GitHubSettings is the tracker block for GitHub Issues. The project number is absent when the
// repository's issues are not organised into a GitHub Project.
type GitHubSettings struct {
	Repo    string
	Project mo.Option[int]
}

// ReviewSettings is the block codefall-review reads. Posting findings to a pull request is visible
// to everyone on it, so it is a decision a project makes rather than a default it inherits — and the
// answer is always recorded, never inferred from the field's absence.
type ReviewSettings struct {
	PostToPullRequest bool
}

// Settings is a complete, valid .codefall/settings.json as a value: the tracker, the block that
// tracker selects, and the review block. Exactly one tracker block is present, which is what the
// schema's oneOf says as well. Beads carries no fields of its own — bd keeps its configuration under
// .beads/ — so it needs no member here.
type Settings struct {
	Tracker string
	// Harnesses is what the project is set up for: at least one, sorted and without repeats, so the
	// file records them in a stable order however they were collected.
	Harnesses []string
	GitHub    mo.Option[GitHubSettings]
	Review    ReviewSettings
}

// NewSettings builds settings from the values a survey or a set of flags collected, and is the only
// way to obtain them: a Settings value that exists is one that may be written. Which values are
// acceptable is the format's to say; which combinations of them make sense for a run is initcmd's,
// and that is what this constructor adds.
func NewSettings(
	tracker string, harnesses []string, repo mo.Option[string], project mo.Option[int],
	review ReviewSettings,
) (Settings, error) {
	name, err := settings.ParseTracker(tracker)
	if err != nil {
		return Settings{}, err
	}

	chosen, err := newHarnesses(harnesses)
	if err != nil {
		return Settings{}, err
	}

	if name != settings.TrackerGitHub {
		if repo.IsPresent() {
			return Settings{}, fmt.Errorf("a repository is only used when the tracker is %q, not %q", settings.TrackerGitHub, name)
		}

		if project.IsPresent() {
			return Settings{}, fmt.Errorf("a project number is only used when the tracker is %q, not %q", settings.TrackerGitHub, name)
		}

		return Settings{Tracker: name, Harnesses: chosen, Review: review}, nil
	}

	github, err := newGitHubSettings(repo, project)
	if err != nil {
		return Settings{}, err
	}

	return Settings{Tracker: name, Harnesses: chosen, GitHub: mo.Some(github), Review: review}, nil
}

// newHarnesses is the set a project is set up for, sorted and without repeats. A project needs at
// least one: settings that name none would describe an install with nowhere to put anything. A name
// is kept as Parse returns it, so a former spelling is written down under the name the harness has
// now.
func newHarnesses(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, errors.New("a project needs at least one harness")
	}

	parsed := make([]string, 0, len(names))

	for _, name := range names {
		current, err := harness.Parse(name)
		if err != nil {
			return nil, err
		}

		parsed = append(parsed, current)
	}

	return slices.Compact(slices.Sorted(slices.Values(parsed))), nil
}

func newGitHubSettings(repo mo.Option[string], project mo.Option[int]) (GitHubSettings, error) {
	name, ok := repo.Get()
	if !ok {
		return GitHubSettings{}, fmt.Errorf("tracker %q needs a repository", settings.TrackerGitHub)
	}

	if err := settings.ValidateRepo(name); err != nil {
		return GitHubSettings{}, err
	}

	if number, ok := project.Get(); ok {
		if err := settings.ValidateProject(number); err != nil {
			return GitHubSettings{}, err
		}
	}

	return GitHubSettings{Repo: name, Project: project}, nil
}
