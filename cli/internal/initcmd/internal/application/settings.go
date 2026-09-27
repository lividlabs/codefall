package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// Where settings live, relative to the directory init is run in. The display form is what a person
// reads in a report; the path form is what the file system is given. codefallDir is the project's
// own codefall directory, which also holds the manifest and, since the install layout, the files
// every harness reaches by a path codefall writes (ADR-006).
const (
	codefallDir  = ".codefall"
	settingsFile = "settings.json"
	settingsName = codefallDir + "/" + settingsFile
)

// SettingsExist reports whether dir already has settings. Presentation asks before it prompts: there
// is no point surveying a project for answers that will not be written.
func (i *Initialize) SettingsExist(dir string) (bool, error) {
	_, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("read %s: %w", settingsName, err)
	}
}

// ChosenHarnesses reports the harnesses .codefall/settings.json records, so a rerun installs for what
// the project already chose rather than asking again. None means there is no settings file, or the
// file records none — which is what a file written before the field existed looks like (ADR-GO-03).
//
// A harness recorded under a spelling it had before it was named for its binary is reported under the
// name it has now, which is the name every step installs for and the settings step writes back.
//
// It decodes the one field it needs rather than the whole document: what the rest of the file may
// hold is the format's business, and this is a question about one answer the project gave.
func (i *Initialize) ChosenHarnesses(dir string) (mo.Option[[]string], error) {
	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return mo.None[[]string](), nil
	case err != nil:
		return mo.None[[]string](), fmt.Errorf("read %s: %w", settingsName, err)
	}

	var document struct {
		Harnesses []string `json:"harnesses"`
	}

	if err := json.Unmarshal(data, &document); err != nil {
		return mo.None[[]string](), fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if len(document.Harnesses) == 0 {
		return mo.None[[]string](), nil
	}

	names := make([]string, 0, len(document.Harnesses))
	for _, name := range document.Harnesses {
		names = append(names, harness.Renamed(name).OrElse(name))
	}

	return mo.Some(names), nil
}

// DeclaredTestDir reports the testing root .codefall/settings.json records, so a rerun works with
// the root the project already declared rather than asking for it again. None means there is no
// settings file, or the file declares no root — which is what a file written before the block
// existed looks like (ADR-GO-03).
//
// Like ChosenHarnesses, it decodes the one field it needs: what the rest of the file may hold is the
// format's business, and this is a question about one answer the project gave.
func (i *Initialize) DeclaredTestDir(dir string) (mo.Option[string], error) {
	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return mo.None[string](), nil
	case err != nil:
		return mo.None[string](), fmt.Errorf("read %s: %w", settingsName, err)
	}

	var document struct {
		Test struct {
			Dir string `json:"dir"`
		} `json:"test"`
	}

	if err := json.Unmarshal(data, &document); err != nil {
		return mo.None[string](), fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if document.Test.Dir == "" {
		return mo.None[string](), nil
	}

	return mo.Some(document.Test.Dir), nil
}

// settings is the first step of a run: it writes .codefall/settings.json, the file doctor checks.
//
// Settings that are already there are left alone unless the run asked for them to be rewritten,
// because init is safe to run again — the steps after this one still have work to do in a project
// that is half set up. The one edit it makes to them is the harness names: an old spelling in the
// settings or the manifest is rewritten to the name the harness has now, and nothing else in either
// file changes.
func (i *Initialize) settings(_ context.Context, request Request) (domain.StepResult, error) {
	exists, err := i.SettingsExist(request.Dir)
	if err != nil {
		return domain.StepResult{}, err
	}

	if exists && !request.Force {
		renamed, err := i.respell(request.Dir, true)
		if err != nil {
			return domain.StepResult{}, err
		}

		if renamed != "" {
			return domain.SettingsStep.Done(renamed), nil
		}

		return domain.SettingsStep.Skipped(settingsName + " already exists (use --force to rewrite it)"), nil
	}

	chosen, err := domain.NewSettings(request.Tracker, request.Harnesses, request.IssuesRepo,
		request.IssuesProject,
		domain.ReviewSettings{PostToPullRequest: request.ReviewPostToPullRequest.OrElse(false)})
	if err != nil {
		return domain.StepResult{}, err
	}

	data, err := encodeSettings(chosen)
	if err != nil {
		return domain.StepResult{}, err
	}

	if err := i.files.MkdirAll(filepath.Join(request.Dir, codefallDir)); err != nil {
		return domain.StepResult{}, fmt.Errorf("create %s/: %w", codefallDir, err)
	}

	if err := i.files.WriteFile(settingsPath(request.Dir), data); err != nil {
		return domain.StepResult{}, fmt.Errorf("write %s: %w", settingsName, err)
	}

	wrote := fmt.Sprintf("wrote %s (%s)", settingsName, describe(chosen))

	// The settings were just written from the run's answers, so only the manifest can still carry an
	// old spelling.
	renamed, err := i.respell(request.Dir, false)
	if err != nil {
		return domain.StepResult{}, err
	}

	if renamed != "" {
		wrote += "; " + renamed
	}

	return domain.SettingsStep.Done(wrote), nil
}

func settingsPath(dir string) string {
	return filepath.Join(dir, codefallDir, settingsFile)
}

// describe is what the step reports it wrote, in the terms the person answering the survey used.
//
// A settings value is named `chosen` throughout this file because `settings` is now the shared
// module that defines the format (ADR-003), and a local of that name would hide it.
func describe(chosen domain.Settings) string {
	parts := []string{
		"tracker: " + chosen.Tracker,
		"harnesses: " + sentenceList(chosen.Harnesses),
	}

	if github, ok := chosen.GitHub.Get(); ok {
		parts = append(parts, "issues repo: "+github.Repo)

		if project, ok := github.Project.Get(); ok {
			parts = append(parts, fmt.Sprintf("issues project: %d", project))
		}
	}

	if chosen.Review.PostToPullRequest {
		parts = append(parts, "review posts to pull requests")
	}

	return strings.Join(parts, ", ")
}

// settingsDocument is the settings file's JSON shape. Encoding lives here rather than in the domain,
// which must not name an encoding, or in infrastructure, which takes bytes — the same reason doctor
// decodes in its application layer. Field order is key order, and it is chosen so the file opens
// with what identifies it and closes with the block the tracker selects.
//
// The test block is not here: it is written by the testing step, into whatever settings the project
// has, because most projects get theirs on a rerun over a file this step skipped (ADR-007).
type settingsDocument struct {
	Schema    string                    `json:"$schema"`
	Version   int                       `json:"version"`
	Tracker   string                    `json:"tracker"`
	Harnesses []string                  `json:"harnesses"`
	Beads     mo.Option[beadsDocument]  `json:"beads,omitzero"`
	GitHub    mo.Option[gitHubDocument] `json:"github,omitzero"`
	Review    reviewDocument            `json:"review"`
}

// reviewDocument is the review block. It is always written, including when the answer is the
// default: a file that states the setting shows there is something to change, where an absent block
// reads as a feature nobody has heard of.
type reviewDocument struct {
	PostToPullRequest bool `json:"postToPullRequest"`
}

// beadsDocument is the empty block the schema requires for the Beads tracker: bd keeps its own
// configuration under .beads/, and the block exists so the "exactly one tracker block" rule reads
// the same for both trackers.
//
// IsZero is what keeps it in the file. `omitzero` asks mo.Option, which asks the value it holds, and
// a struct with no fields is always the zero value — so without this the present block would be
// dropped as if it were absent.
type beadsDocument struct{}

func (beadsDocument) IsZero() bool { return false }

// gitHubDocument is the GitHub tracker's block. The project number is omitted rather than written as
// null when the repository's issues are not organised into a project.
type gitHubDocument struct {
	Repo    string         `json:"issuesRepo"`
	Project mo.Option[int] `json:"issuesProject,omitzero"`
}

func (gitHubDocument) IsZero() bool { return false }

// encodeSettings renders settings as the bytes that go in the file: two-space indent and a trailing
// newline, so the result is what a person would have written by hand and diffs a line at a time.
func encodeSettings(chosen domain.Settings) ([]byte, error) {
	document := settingsDocument{
		Schema:    settings.SchemaID,
		Version:   settings.Version,
		Tracker:   chosen.Tracker,
		Harnesses: chosen.Harnesses,
		Review:    reviewDocument{PostToPullRequest: chosen.Review.PostToPullRequest},
	}

	// One case per tracker, so adding a tracker is a case here and a block above rather than a
	// silent omission. NewSettings has already rejected any tracker the format does not know, which
	// is what makes the default unreachable.
	switch chosen.Tracker {
	case settings.TrackerGitHub:
		github, ok := chosen.GitHub.Get()
		if !ok {
			return nil, fmt.Errorf("tracker %q has no github block", chosen.Tracker)
		}

		document.GitHub = mo.Some(gitHubDocument{Repo: github.Repo, Project: github.Project})
	case settings.TrackerBeads:
		document.Beads = mo.Some(beadsDocument{})
	default:
		return nil, fmt.Errorf("tracker %q has no block to write", chosen.Tracker)
	}

	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", settingsName, err)
	}

	return append(data, '\n'), nil
}
