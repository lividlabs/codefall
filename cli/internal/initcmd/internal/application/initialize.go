// Package application holds initcmd's one use case. It owns the gateway interfaces the steps need
// and depends on nothing but the standard library, mo, and initcmd's own domain.
package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/manifest"
)

// FileSystem is initcmd's view of the working directory (one gateway role). It reads what is already
// there and writes what init creates; the permissions those writes use are infrastructure's
// decision, not the use case's.
type FileSystem interface {
	// ReadFile returns a file's bytes. A missing file satisfies errors.Is(err, fs.ErrNotExist).
	ReadFile(path string) ([]byte, error)
	// MkdirAll creates path and every parent it needs, and does nothing when it already exists.
	MkdirAll(path string) error
	// WriteFile writes data to path, replacing whatever was there.
	WriteFile(path string, data []byte) error
	// Remove deletes one file or one empty directory. A path that is not there is success.
	Remove(path string) error
	// DirIsEmpty reports whether path is a directory holding nothing. A path that is not there, or
	// that is a file, is not an empty directory and not an error either.
	DirIsEmpty(path string) (bool, error)
}

// CommandRunner locates and runs the external tools codefall depends on (one gateway role). It is
// the same role doctor declares, because it is the same job: initcmd asks gh what repository this
// directory belongs to, asks git about the directory, and runs the harness and Beads.
type CommandRunner interface {
	// LookPath returns where a tool lives, or None when it is not on PATH. There is no useful "why"
	// behind a missing binary, so this is an Option and not an error (ADR-GO-03).
	LookPath(name string) mo.Option[string]
	// Run executes a tool and waits for it. A non-zero exit is a normal result; the error is
	// reserved for a command that could not be started at all.
	Run(ctx context.Context, dir, name string, args ...string) (CommandResult, error)
}

// CommandResult is what one external command produced.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExtensionSource is the extension's embedded payload (one gateway role): a tree in the binary, so
// both operations are local reads. Fetch mirrors it for the extension step; Read takes one file, for
// the hook step's per-harness definitions.
type ExtensionSource interface {
	// Fetch mirrors the named subtrees of the extension's tree onto destDir, each file keeping the
	// path it has in the tree, and returns every path it installed, relative to destDir, so the
	// caller can put it on record, and the ones that were missing or held other bytes, so the caller
	// can say whether the copy changed anything. A file that already holds the tree's bytes is left
	// as it is. A path under exclude is left behind: an entry holding a slash names a path in the
	// tree, and an entry that is a bare file name matches that file wherever it sits.
	Fetch(ctx context.Context, destDir string, sources, exclude []string) (Fetched, error)
	// Read returns one file from the tree.
	Read(path string) ([]byte, error)
	// RenamedSkill returns the name a skill directory has now when former is a name it used to have,
	// or None when former is no skill the tree has ever renamed. It is what lets an upgrade report a
	// removed directory as a rename rather than a deletion.
	RenamedSkill(former string) mo.Option[string]
}

// Fetched is what one Fetch did: every path the tree holds under the subtrees it was asked for,
// which is what an install records, and the subset it had to write because the file was missing or
// held other bytes, which is what an install reports. Both are relative to the destination and
// sorted.
type Fetched struct {
	Files   []string
	Changed []string
}

// ChangeLog is the record of what each release changed (one gateway role): the repository's
// CHANGELOG.md, embedded in the binary, read for the one section a project upgrading has to hear
// about before anything moves.
type ChangeLog interface {
	// BreakingChanges returns every release that recorded breaking changes, with the notes each
	// recorded, in the order the changelog lists them.
	BreakingChanges() ([]BreakingRelease, error)
}

// BreakingRelease is one release's breaking changes: its version as the changelog spells it, and one
// note per change, as release-please wrote them from the commits' BREAKING CHANGE footers.
type BreakingRelease struct {
	Version string
	Notes   []string
}

// Request is what presentation hands the use case: every answer a survey or a set of flags could
// collect, already parsed. Absence is an Option, so "no project number" and "project number zero"
// cannot be confused (ADR-GO-03). Tracker may be empty when settings already exist, because then
// nothing is built from it — so a step that comes to depend on which tracker the project uses must
// read it out of the settings file that is already there, not out of this field.
//
// Both commands hand over the same contract: init builds it from a survey or from flags, upgrade
// from what the settings and the manifest already record. The steps do not know which command is
// running them, and have no reason to.
type Request struct {
	Dir        string
	Tracker    string
	IssuesRepo mo.Option[string]
	// CLIVersion is the binary version that will get stamped into .codefall/manifest.json when
	// this run writes one. Presentation reads it from the build's own info.
	CLIVersion string

	// TestDir is the testing root the project declares: where its test cases live, relative to the
	// directory holding .codefall/ (ADR-007). Empty means nobody was asked — a rerun of a project
	// settled before the block existed — and the run takes the format's default, which is the same
	// answer the survey offers.
	TestDir string

	IssuesProject mo.Option[int]
	// ReviewPostToPullRequest is whether codefall-review may post its findings to a pull request.
	// None means nobody was asked — a scripted run that gave no flag — and the file records the
	// default rather than leaving the block out.
	ReviewPostToPullRequest mo.Option[bool]
	// Harnesses is every harness this run installs for, in whatever order presentation collected
	// them and possibly with repeats. Every step reads it through chosen, which sorts it and drops
	// the repeats, so a name given twice is one install and each step reports the same order. A run
	// that names none is refused by preflight rather than quietly installing nothing.
	Harnesses []string
	// Current is true when the manifest records every harness at this binary's version and there is
	// nothing left to declare or rewrite. The run then skips the Beads step, which the install at this
	// version has already been through, and so needs no bd on PATH. Every other step still runs, the
	// extension copy and the cleanup after it included, because a person can delete a skill, a shared
	// script, a line, or a section between two runs of the same version, and doctor's remedy for each
	// is `codefall upgrade`. Each step reports a skip when it found its work already done, which is
	// how the command knows whether to say the install is up to date.
	Current bool
}

// Observer watches a run step by step, so a terminal can show what is happening while it happens.
// It is declared here because the use case is what has something to say; what a listener does with
// it — a spinner, a log line, nothing at all — belongs to the caller.
type Observer interface {
	// StepStarted is called before a step does its work.
	StepStarted(step domain.Step)
	// StepFinished is called after a step has done its work, and not at all for a step that failed.
	StepFinished(result domain.StepResult)
}

// Initialize sets a project up for codefall: it writes .codefall/settings.json, installs the harness
// extension, initialises Beads, registers codefall's hooks with the harness, writes the sections of
// AGENTS.md that say how the project uses it, and makes the tree its test cases live in. It is the
// one use case behind two commands: `codefall init` runs it once, on a project with no manifest, and
// `codefall upgrade` runs it again for the harnesses the settings record, replacing what it owns and
// touching nothing else.
type Initialize struct {
	files   FileSystem
	runner  CommandRunner
	source  ExtensionSource
	changes ChangeLog
}

// NewInitialize builds the use case over its gateways.
func NewInitialize(files FileSystem, runner CommandRunner, source ExtensionSource, changes ChangeLog) *Initialize {
	return &Initialize{files: files, runner: runner, source: source, changes: changes}
}

// step is one unit of work in a run. It returns what it did, or an error that stops the run: a step
// that could not finish leaves the project half set up, and the steps after it would be building on
// something that is not there.
type step struct {
	domain.Step
	run func(ctx context.Context, request Request) (domain.StepResult, error)
}

// Run performs every step in order, telling the observer as each one starts and finishes. The first
// step that fails ends the run, and its error names the step so the reader knows how far init got.
//
// The tools the run needs are checked before the first step starts, so a missing one leaves the
// project untouched rather than half set up. That error already says what is wrong and what to do
// about it, so it is returned as it is, with no step to name in front of it.
func (i *Initialize) Run(ctx context.Context, request Request, observer Observer) (domain.Report, error) {
	if observer == nil {
		observer = silentObserver{}
	}

	if err := i.preflight(ctx, request); err != nil {
		return domain.Report{}, err
	}

	// What the extension step copied, for the manifest the run writes at the end. It is a local of
	// this run rather than a field of the use case, which every run of the process shares.
	written := installed{}

	// What the previous finished run recorded, read before any step can rewrite the file: the
	// cleanup step compares it with what this run writes, and a run with no record — the first run —
	// has nothing to compare and no cleanup step.
	previous, upgrading, err := i.recordedManifest(request.Dir)
	if err != nil {
		return domain.Report{}, err
	}

	steps := []step{
		{Step: domain.SettingsStep, run: i.settings},
		{Step: domain.ExtensionStep, run: func(ctx context.Context, request Request) (domain.StepResult, error) {
			result, files, err := i.extension(ctx, request)
			written = files

			return result, err
		}},
	}

	if upgrading {
		steps = append(steps, step{Step: domain.CleanupStep, run: func(_ context.Context, request Request) (domain.StepResult, error) {
			return i.cleanup(request, previous, written)
		}})
	}

	// An install already current at this version has a Beads it has already initialised. The steps
	// after this one repair a project's own files, and they run either way.
	if !request.Current {
		steps = append(steps, step{Step: domain.BeadsStep, run: i.beads})
	}

	steps = append(steps,
		step{Step: domain.HookStep, run: i.hook},
		step{Step: domain.AgentsStep, run: i.agents},
		step{Step: domain.TestingStep, run: i.testing},
		step{Step: domain.IgnoreStep, run: i.ignore},
	)

	results := make([]domain.StepResult, 0, len(steps))

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return domain.Report{}, fmt.Errorf("initialize: %w", err)
		}

		observer.StepStarted(step.Step)

		result, err := step.run(ctx, request)
		if err != nil {
			return domain.Report{}, fmt.Errorf("%s: %w", step.ID, err)
		}

		observer.StepFinished(result)

		results = append(results, result)
	}

	// The manifest is the last thing a run writes, because it says the install is complete and the
	// upgrade gate believes it: written any earlier, a run that failed a later step would leave a
	// record claiming work it never did, and the next run would report there was nothing to do.
	//
	// Every run writes it, a run over a current install included. There it comes out byte for byte
	// as it went in unless the cleanup or the settings step changed the record, and the file is not a
	// step, so writing it says nothing.
	if err := i.writeManifest(request.Dir, request.CLIVersion, written); err != nil {
		return domain.Report{}, fmt.Errorf("record the installation to %s: %w", manifest.Name, err)
	}

	return domain.NewReport(results...), nil
}

// chosen is the harnesses a run installs for, sorted and without repeats. Every step reads the set
// through this, so the order a flag or a survey happened to collect them in cannot change what a
// step does or what it reports.
func chosen(request Request) []string {
	return slices.Compact(slices.Sorted(slices.Values(request.Harnesses)))
}

// sentenceList joins names the way a person reads a list, for the sentences the steps report about
// themselves.
func sentenceList(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}

// silentObserver stands in for a caller that has nothing to show, so Run has no nil check in its
// loop.
type silentObserver struct{}

func (silentObserver) StepStarted(domain.Step)        {}
func (silentObserver) StepFinished(domain.StepResult) {}
