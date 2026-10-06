package application

import (
	"context"
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/harness"
)

// The three subtrees of the embedded tree an install copies, and where each lands. Skills are the
// only part a harness finds by convention, so they go into each chosen harness's own skills
// directory; everything else is reached by a path codefall writes, so it goes to codefallDir once
// (ADR-006). A subtree not named here is not installed at all, which is how the per-harness hook
// definitions stay the hook step's and the maintainer documents stay in this repository.
const (
	skillsSource = "skills"
	hooksSource  = "hooks/shared"
	sharedSource = "shared"
)

// maintainerDocs are the files inside skills/ that a project has no reader for: the rules for
// changing a skill, and each skill's record of what it took from elsewhere. Both are written for
// someone working on codefall, and no skill, hook, or script names either, so leaving them behind
// costs a project nothing (ADR-006). They are excluded at copy time rather than dropped from the
// embedded tree, which the facade test reads whole.
var maintainerDocs = []string{skillsSource + "/AGENTS.md", "NOTES.md"}

// installed is what the extension step wrote, as the manifest records it: the files each harness's
// skills directory received, and the files .codefall/ received once for the whole run.
type installed struct {
	harnesses map[string][]string
	shared    []string
}

// extension is the second step of a run: it mirrors the embedded extension tree out of the binary,
// so it never touches a network or a foreign CLI. It runs on every run, an upgrade over a current
// install included, because a skill or a shared script deleted by hand is put back only by this
// copy. It also hands back every path it installed, which the run records in the manifest once the
// last step has succeeded.
//
// Both copies go through the skill rename, so a project that chose a prefix other than the source's
// receives every skill, and every mention of one, under that prefix (ADR-015).
//
// It copies twice, for two different reasons. Harnesses share skills directories — four of the five
// read `.agents/` — so the skills are copied once per directory, and every harness that reads that
// directory records the same files. Which directory each one is belongs to the shared harness module
// rather than to this step: doctor reports on the same directories, so neither component can be the
// one that decides where they are (ADR-003). The shared scripts and the files skills read are copied
// once, to .codefall/, because the paths that reach them are codefall's at both ends.
func (i *Initialize) extension(
	ctx context.Context, request Request,
) (domain.StepResult, installed, error) {
	dests, err := skillsDirs(request)
	if err != nil {
		return domain.StepResult{}, installed{}, err
	}

	rename, err := i.skillRename(request)
	if err != nil {
		return domain.StepResult{}, installed{}, err
	}

	// What each directory received, so a directory two harnesses share is fetched once, and what
	// each harness installed, which is what the manifest records. What the copy had to write, because
	// the file was missing or held other bytes, is what the step reports.
	copied := map[string][]string{}
	written := installed{harnesses: map[string][]string{}}

	var changed []string

	for _, name := range chosen(request) {
		dest := dests[name]

		if _, done := copied[dest]; !done {
			fetched, err := i.fetchSkills(ctx, request.Dir, dest, rename)
			if err != nil {
				return domain.StepResult{}, installed{}, err
			}

			copied[dest] = fetched.Files
			changed = append(changed, projectPaths(dest, fetched.Changed)...)
		}

		written.harnesses[name] = projectPaths(dest, copied[dest])
	}

	shared, err := i.fetchShared(ctx, request.Dir, rename)
	if err != nil {
		return domain.StepResult{}, installed{}, err
	}

	written.shared = projectPaths(codefallDir, shared.Files)
	changed = append(changed, projectPaths(codefallDir, shared.Changed)...)

	slices.Sort(changed)

	return extensionResult(directoryList(slices.Sorted(maps.Keys(copied))), changed, len(writtenPaths(written))),
		written, nil
}

// changedLimit is how many changed files the extension step names before it gives their number
// instead. A repair names the few files it put back; an upgrade across versions rewrites much of the
// tree, and a list that long would bury the lines the other steps print.
const changedLimit = 10

// extensionResult is what the extension step reports. A copy that found every file already holding
// the tree's bytes changed nothing and says so as a skip, which is what lets an upgrade over a
// current install say it is up to date. A copy that wrote every file is a fresh install and says
// where it installed. A copy that wrote some of them names what it wrote, or counts it past the
// limit.
func extensionResult(dirs string, changed []string, total int) domain.StepResult {
	if len(changed) == 0 {
		return domain.ExtensionStep.Skipped(fmt.Sprintf(
			"codefall's skills in %s and its shared files in %s/ already match this version", dirs, codefallDir))
	}

	detail := fmt.Sprintf("installed codefall's skills into %s and its shared files into %s/", dirs, codefallDir)

	switch {
	case len(changed) == total:
		return domain.ExtensionStep.Done(detail)
	case len(changed) == 1:
		return domain.ExtensionStep.Done(detail + "; wrote " + changed[0] + ", which was missing or differed")
	case len(changed) <= changedLimit:
		return domain.ExtensionStep.Done(detail + "; wrote " + sentenceList(changed) + ", which were missing or differed")
	default:
		return domain.ExtensionStep.Done(fmt.Sprintf("%s; wrote %d files that were missing or differed", detail, len(changed)))
	}
}

// skillsDirs is where each chosen harness reads skills, or the error naming a harness codefall
// cannot set up. Presentation refuses an unknown harness before the use case sees it, so that error
// is a path a run should never take.
func skillsDirs(request Request) (map[string]string, error) {
	dirs := map[string]string{}

	for _, name := range chosen(request) {
		dir, known := harness.SkillsDir(name).Get()
		if !known {
			return nil, fmt.Errorf("harness %q has no extension mechanism", name)
		}

		dirs[name] = dir
	}

	return dirs, nil
}

// projectPaths puts the destination directory back in front of what Fetch wrote. A recorded path has
// to say which directory it is in: four harnesses share one directory, and a bare "skills/…" would
// not say whether it landed under `.claude/` or `.agents/`, nor a bare "shared/…" that it landed
// under `.codefall/`.
func projectPaths(dest string, files []string) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, path.Join(dest, file))
	}

	return paths
}

// directoryList names the directories the step wrote to, as the end of the sentence it reports: one
// directory on its own, and several joined the way a person reads a list.
func directoryList(dirs []string) string {
	slashed := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		slashed = append(slashed, dir+"/")
	}

	return sentenceList(slashed)
}
