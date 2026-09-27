package application

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall-cli/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/harness"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/manifest"
	"github.com/lividlabs/codefall-cli/cli/internal/shared/settings"
)

// skillsRequest is a run on a harness that reads the .agents/skills convention.
func skillsRequest() Request {
	return Request{Dir: workingDir, Tracker: settings.TrackerBeads, Harnesses: []string{harness.Codex}}
}

// sharedInstall is the .codefall/ entry every finished run records: the files a run writes once,
// whatever harnesses it is for. The paths say which directory they landed in, the same way a
// harness's do.
func sharedInstall(version string) manifest.Install {
	return manifest.Install{Version: version, Files: []string{
		".codefall/hooks/shared/codefall-block-merge-to-main.sh",
		".codefall/shared/preflight.sh",
	}}
}

// The step copies the embedded extension tree into the project's .agents/, one Fetch call, one
// destination.
func TestSkillsStepCopiesTheEmbeddedTree(t *testing.T) {
	fetcher := newFakeExtensionSource()

	report, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(t.Context(), skillsRequest(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	result := report.Results()[1]
	if result.Outcome != domain.OutcomeDone {
		t.Errorf("outcome = %v, want DONE", result.Outcome)
	}

	want := "installed codefall's skills into .agents/ and its shared files into .codefall/"
	if result.Detail != want {
		t.Errorf("detail = %q, want %q", result.Detail, want)
	}

	if len(fetcher.calls) != 2 || fetcher.calls[0].dir != filepath.Join(workingDir, ".agents") {
		t.Errorf("fetcher calls = %+v, want one fetch into %q and one into .codefall/",
			fetcher.calls, filepath.Join(workingDir, ".agents"))
	}
}

// The fetcher failing stops the run in the extension step, which is the second step of five, so the
// report carries the one step that is already done.
func TestSkillsStepStopsTheRunWhenTheCopyFails(t *testing.T) {
	fetcher := newFakeExtensionSource()
	fetcher.err = errors.New("disk full")

	_, err := NewInitialize(settled(""), toolsInstalled(), fetcher).Run(t.Context(), skillsRequest(), nil)
	if err == nil ||
		!strings.HasPrefix(err.Error(), domain.ExtensionStep.ID+": ") ||
		!strings.Contains(err.Error(), "install the embedded extension: disk full") {
		t.Errorf("Run error = %v, want it to name the extension step and the copy that failed", err)
	}
}

// A skills-directory harness runs all seven steps: the extension step installs where the mechanism
// installs, the hook step registers there too, and the rest are untouched.
func TestSkillsRunStillRunsEveryStep(t *testing.T) {
	report, err := NewInitialize(settled(""), toolsInstalled(), newFakeExtensionSource()).Run(
		t.Context(), skillsRequest(), nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	results := report.Results()
	if len(results) != 7 {
		t.Fatalf("Results() = %+v, want a result for each of the seven steps", results)
	}

	if got := results[3].Outcome; got != domain.OutcomeDone {
		t.Errorf("the hook step = %v, want DONE", got)
	}

	want := "merged codefall's hooks into .codex/hooks.json"
	if got := results[3].Detail; got != want {
		t.Errorf("the hook step's detail = %q, want %q", got, want)
	}

	steps := []domain.Step{
		domain.SettingsStep, domain.ExtensionStep, domain.BeadsStep, domain.HookStep,
		domain.AgentsStep, domain.TestingStep, domain.IgnoreStep,
	}
	got := make([]domain.Step, 0, len(results))
	for _, result := range results {
		got = append(got, result.Step)
	}
	if !slices.Equal(got, steps) {
		t.Errorf("steps = %+v, want %+v", got, steps)
	}
}

// The manifest is what the upgrade gate reads to decide a rerun has nothing to do, so it is written
// only when every step has succeeded. Written in the extension step, it made that claim with the
// hook and AGENTS.md steps still to come: a run that failed at either left a manifest saying the
// install was current, and the next run reported it was already up to date without registering the
// hooks the failed run never got to.
func TestTheManifestRecordsOnlyARunThatFinished(t *testing.T) {
	t.Run("a run that finished", func(t *testing.T) {
		files := settled("")
		request := beadsRequest()
		request.CLIVersion = "v1.2.3"

		if _, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), request, nil); err != nil {
			t.Fatalf("Run: %v", err)
		}

		var recorded manifest.Document
		if err := json.Unmarshal(files.files[filepath.Join(workingDir, manifest.Name)], &recorded); err != nil {
			t.Fatalf("decode %s: %v", manifest.Name, err)
		}

		want := manifest.Document{Harnesses: map[string]manifest.Install{
			harness.Claude: {Version: "v1.2.3", Files: []string{".claude/skills/design/SKILL.md"}},
		}, Shared: sharedInstall("v1.2.3")}
		if !reflect.DeepEqual(recorded, want) {
			t.Errorf("manifest = %+v, want %+v", recorded, want)
		}
	})

	t.Run("a run that failed a later step", func(t *testing.T) {
		files := settled(`{"hooks": "not an object"}`)

		_, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Run(t.Context(), beadsRequest(), nil)
		if err == nil {
			t.Fatal("Run = nil error, want the hook step to stop the run")
		}

		if _, wrote := files.files[filepath.Join(workingDir, manifest.Name)]; wrote {
			t.Error("a failed run recorded a manifest, want none until every step has succeeded")
		}
	})
}

// Installed is the record through the use-case boundary: the version each harness was installed at.
// The gate compares it per harness, because a current version recorded for one harness is no answer
// about another.
func TestInstalledReportsWhatFinishedRunsRecorded(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		missing bool
		want    mo.Option[Installation]
	}{
		{
			name: "a manifest recording two harnesses, each at the version that installed it",
			body: `{"harnesses": {"claude": {"version": "v1.2.3"}, "codex": {"version": "v1.1.0"}}}`,
			want: mo.Some(Installation{Versions: map[string]string{
				harness.Claude: "v1.2.3", harness.Codex: "v1.1.0"}}),
		},
		{name: "no manifest", missing: true, want: mo.None[Installation]()},
		{
			name: "a harness recorded with no version, which is nothing a comparison can use",
			body: `{"harnesses": {"codex": {"files": [".agents/skills/design/SKILL.md"]}}}`,
			want: mo.None[Installation](),
		},
		{
			// The shape written before a run recorded a version per harness. It decodes cleanly and
			// records nothing, so the next run repeats every step rather than trusting a record it
			// cannot read.
			name: "a manifest from before harnesses were recorded per install",
			body: `{"harness": "codex", "version": "v1.2.3"}`,
			want: mo.None[Installation](),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := newFakeFileSystem()
			if !tc.missing {
				files.files[filepath.Join(workingDir, manifest.Name)] = []byte(tc.body)
			}

			got, err := NewInitialize(files, toolsInstalled(), newFakeExtensionSource()).Installed(workingDir)
			if err != nil {
				t.Fatalf("Installed: %v", err)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Installed = %v, want %v", got, tc.want)
			}
		})
	}
}
