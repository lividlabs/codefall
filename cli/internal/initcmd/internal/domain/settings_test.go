package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// oneHarness is the harness every case that is not about harnesses passes, so a settings value can be
// built at all.
var oneHarness = []string{harness.Claude}

func TestNewSettingsForGitHub(t *testing.T) {
	value, err := NewSettings(settings.TrackerGitHub, oneHarness, mo.Some("lividlabs/codefall-cli"),
		mo.Some(3), ReviewSettings{})
	if err != nil {
		t.Fatalf("NewSettings: %v", err)
	}

	if value.Tracker != settings.TrackerGitHub {
		t.Errorf("Tracker = %q, want %q", value.Tracker, settings.TrackerGitHub)
	}

	github, ok := value.GitHub.Get()
	if !ok {
		t.Fatalf("GitHub = %v, want a block", value.GitHub)
	}

	if github.Repo != "lividlabs/codefall-cli" {
		t.Errorf("GitHub.Repo = %q, want %q", github.Repo, "lividlabs/codefall-cli")
	}

	if got, ok := github.Project.Get(); !ok || got != 3 {
		t.Errorf("GitHub.Project = %v, want Some(3)", github.Project)
	}
}

func TestNewSettingsForGitHubWithoutAProject(t *testing.T) {
	value, err := NewSettings(settings.TrackerGitHub, oneHarness, mo.Some("owner/name"),
		mo.None[int](), ReviewSettings{})
	if err != nil {
		t.Fatalf("NewSettings: %v", err)
	}

	github, _ := value.GitHub.Get()
	if github.Project.IsPresent() {
		t.Errorf("GitHub.Project = %v, want None", github.Project)
	}
}

func TestNewSettingsForBeadsCarriesNoGitHubBlock(t *testing.T) {
	value, err := NewSettings(settings.TrackerBeads, oneHarness, mo.None[string](), mo.None[int](),
		ReviewSettings{})
	if err != nil {
		t.Fatalf("NewSettings: %v", err)
	}

	if value.Tracker != settings.TrackerBeads {
		t.Errorf("Tracker = %q, want %q", value.Tracker, settings.TrackerBeads)
	}

	if value.GitHub.IsPresent() {
		t.Errorf("GitHub = %v, want None", value.GitHub)
	}
}

// The set is sorted and repeats are dropped, so the file records one line per harness in an order
// that does not depend on how the answers were collected.
func TestNewSettingsSortsTheHarnessesAndDropsRepeats(t *testing.T) {
	value, err := NewSettings(settings.TrackerBeads,
		[]string{harness.Codex, harness.Claude, harness.Codex},
		mo.None[string](), mo.None[int](), ReviewSettings{})
	if err != nil {
		t.Fatalf("NewSettings: %v", err)
	}

	if want := []string{harness.Claude, harness.Codex}; !slices.Equal(value.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", value.Harnesses, want)
	}
}

// A former spelling is written down under the name the harness has now, and counts as a repeat of
// it, so settings built from an old answer never carry the old name forward.
func TestNewSettingsWritesAFormerHarnessNameAsTheCurrentOne(t *testing.T) {
	value, err := NewSettings(settings.TrackerBeads,
		[]string{"claude-code", "antigravity", harness.Claude},
		mo.None[string](), mo.None[int](), ReviewSettings{})
	if err != nil {
		t.Fatalf("NewSettings: %v", err)
	}

	if want := []string{harness.Agy, harness.Claude}; !slices.Equal(value.Harnesses, want) {
		t.Errorf("Harnesses = %q, want %q", value.Harnesses, want)
	}
}

func TestNewSettingsRejects(t *testing.T) {
	for _, tc := range []struct {
		name      string
		tracker   string
		harnesses []string
		repo      mo.Option[string]
		project   mo.Option[int]
		want      string
	}{
		{
			name:      "an unknown tracker",
			tracker:   "linear",
			harnesses: oneHarness,
			repo:      mo.None[string](),
			project:   mo.None[int](),
			want:      "unknown tracker",
		},
		{
			name:      "no harness at all, which would install nowhere",
			tracker:   settings.TrackerBeads,
			harnesses: nil,
			repo:      mo.None[string](),
			project:   mo.None[int](),
			want:      "at least one harness",
		},
		{
			name:      "a harness codefall cannot set up",
			tracker:   settings.TrackerBeads,
			harnesses: []string{harness.Claude, "cursor"},
			repo:      mo.None[string](),
			project:   mo.None[int](),
			want:      `harness "cursor" is not supported yet`,
		},
		{
			name:      "github without a repository",
			tracker:   settings.TrackerGitHub,
			harnesses: oneHarness,
			repo:      mo.None[string](),
			project:   mo.None[int](),
			want:      "needs a repository",
		},
		{
			name:      "a repository that is not owner/name",
			tracker:   settings.TrackerGitHub,
			harnesses: oneHarness,
			repo:      mo.Some("codefall-cli"),
			project:   mo.None[int](),
			want:      "owner/name",
		},
		{
			name:      "a project number below one",
			tracker:   settings.TrackerGitHub,
			harnesses: oneHarness,
			repo:      mo.Some("owner/name"),
			project:   mo.Some(0),
			want:      "positive integer",
		},
		{
			name:      "a repository under beads",
			tracker:   settings.TrackerBeads,
			harnesses: oneHarness,
			repo:      mo.Some("owner/name"),
			project:   mo.None[int](),
			want:      "a repository is only used",
		},
		{
			name:      "a project number under beads",
			tracker:   settings.TrackerBeads,
			harnesses: oneHarness,
			repo:      mo.None[string](),
			project:   mo.Some(3),
			want:      "a project number is only used",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := NewSettings(tc.tracker, tc.harnesses, tc.repo, tc.project, ReviewSettings{})
			if err == nil {
				t.Fatalf("NewSettings = %+v, want an error", value)
			}

			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("NewSettings error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}
