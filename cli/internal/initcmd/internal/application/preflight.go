package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lividlabs/codefall-cli/cli/internal/shared/text"
)

// gitCommand is how git is invoked. Init never asks git to change anything: it asks it two questions
// about the directory, because bd init answers them for itself in ways nobody asked for.
const gitCommand = "git"

// requiredTool is an external tool a run cannot do without, and what a reader should do about it
// being missing. A tool with nothing useful to suggest has no remedy, and the complaint is then just
// the missing tool.
type requiredTool struct {
	name   string
	remedy string
}

// preflight checks that every tool this run will reach for is on PATH and that the directory is one
// the run can work in, before any step has done anything. A tool that turns up missing halfway
// through leaves the project half set up, which is the one state init exists to avoid — so the whole
// run refuses to start instead.
//
// Every problem is reported, not just the first: someone clearing prerequisites wants the whole list
// in one go. A missing tool is the exception to that only in one direction — the questions that tool
// would have answered are not asked, so it is one complaint rather than a cascade of them.
func (i *Initialize) preflight(ctx context.Context, request Request) error {
	var problems []error

	add := func(problem error) {
		if problem != nil {
			problems = append(problems, problem)
		}
	}

	// A run for no harness has nowhere to install, and every step after this one would report doing
	// nothing rather than failing. It is refused here, where a run that cannot finish has not
	// started either.
	if len(chosen(request)) == 0 {
		add(errors.New("no harness to set up"))
	}

	for _, tool := range requiredTools(request) {
		if i.runner.LookPath(tool.name).IsAbsent() {
			add(missingTool(tool))
		}
	}

	// git answers both questions below, so they are asked only when git is there.
	if i.runner.LookPath(gitCommand).IsPresent() {
		worktree, problem := i.worktreeProblem(ctx, request.Dir)
		add(problem)

		// The hook step writes the install directory's path into the commands it registers, and a
		// path it cannot quote is better refused before the first step than after the third.
		if worktree {
			_, problem := i.repositoryPrefix(ctx, request.Dir)
			add(problem)
		}

		// bd init commits, so what it would sweep into that commit only matters when bd init is going
		// to run — which needs bd itself to say whether Beads is already initialised, and needs a
		// work tree to commit in. A run over a current install has no Beads step at all.
		if worktree && !request.Current && i.runner.LookPath(beadsCommand).IsPresent() {
			for _, problem := range i.beadsCommitProblems(ctx, request.Dir) {
				add(problem)
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}

	return problemList{problems: problems}
}

// problemList is everything preflight found, as one error. It joins the complaints with "; " rather
// than the newlines errors.Join writes, because Fang renders an error as a sentence — and it unwraps
// to the list it was built from, so a context.Canceled inside any of them is still one errors.Is
// away.
type problemList struct {
	problems []error
}

func (p problemList) Error() string {
	messages := make([]string, 0, len(p.problems))
	for _, problem := range p.problems {
		messages = append(messages, problem.Error())
	}

	return strings.Join(messages, "; ")
}

func (p problemList) Unwrap() []error { return p.problems }

// missingTool is the complaint about a tool that is not installed.
func missingTool(tool requiredTool) error {
	if tool.remedy == "" {
		return fmt.Errorf("%s is not on PATH", tool.name)
	}

	return fmt.Errorf("%s is not on PATH (%s)", tool.name, tool.remedy)
}

// worktreeProblem reports whether dir is inside a git work tree, and the complaint when it is not.
// It matters because bd init run outside a repository silently runs git init first, which is a
// decision about the project that init has no business making on someone's behalf.
//
// The answer is the word git printed, not the exit code: inside a bare repository, and inside a
// .git directory itself, `rev-parse --is-inside-work-tree` prints false and exits 0. Neither is
// somewhere bd init could commit what it wrote.
func (i *Initialize) worktreeProblem(ctx context.Context, dir string) (bool, error) {
	result, err := i.probe(ctx, dir, gitCommand, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, err
	}

	if result.ExitCode != 0 || strings.TrimSpace(result.Stdout) != "true" {
		return false, errors.New("not a git work tree (bd init would create a repository here; " +
			"run git init first)")
	}

	return true, nil
}

// beadsCommitProblems is everything about the directory that bd init's own commit would take with
// it. Beads that is already initialised means bd init will not run, and then none of it is init's
// business — or bd's — so the questions are not even asked.
//
// The two problems are reported independently. A staged file is caught by the first and, if it is
// one of the paths bd stages by name, by the second as well: they are two ways of losing authorship
// of a change, with two different things to do about it, and someone clearing prerequisites wants
// both at once rather than the second one after fixing the first.
func (i *Initialize) beadsCommitProblems(ctx context.Context, dir string) []error {
	initialized, err := i.beadsInitialized(ctx, dir)
	if err != nil {
		return []error{err}
	}

	if initialized {
		return nil
	}

	return []error{i.stagedChangesProblem(ctx, dir), i.uncommittedFilesProblem(ctx, dir)}
}

// stagedChangesProblem reports what to say when the index has changes waiting and bd init is about
// to run. bd init makes its own commit with no pathspec, so anything already staged is swept into a
// commit that says it initialised Beads — which is why a run with a dirty index refuses to start.
func (i *Initialize) stagedChangesProblem(ctx context.Context, dir string) error {
	result, err := i.probe(ctx, dir, gitCommand, "diff", "--cached", "--quiet")
	if err != nil {
		return err
	}

	switch result.ExitCode {
	case 0:
		return nil
	case 1:
		return errors.New("the git index has staged changes; bd init commits the whole index, " +
			"so commit or unstage them first")
	default:
		return fmt.Errorf("git diff --cached --quiet exited %d: %s",
			result.ExitCode, text.FirstLine(result.Stderr))
	}
}

// The paths bd init stages by name before its pathspec-less commit, when they exist. An uncommitted
// change to any of them — staged, unstaged, or untracked — would be committed under bd's message
// rather than the author's, which is a change of authorship nobody asked for.
//
// The extension step's own write to .claude/settings.json is not one of these. It happens after
// preflight, and bd committing it is what should happen: the file is codefall's to write. What this
// guard is for is somebody else's edit to the same file, made before init ran.
var beadsCommitPaths = []string{
	".gitignore", "AGENTS.md", "CLAUDE.md", ".claude/settings.json", ".codex", ".agents",
}

// uncommittedFilesProblem reports what to say when one of the paths bd init stages has a change that
// is not committed yet. It names the paths git reported rather than all six, because the list a
// reader needs is the one they have to do something about.
func (i *Initialize) uncommittedFilesProblem(ctx context.Context, dir string) error {
	args := append([]string{"status", "--porcelain", "--"}, beadsCommitPaths...)

	result, err := i.probe(ctx, dir, gitCommand, args...)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("%s exited %d: %s",
			commandLine(gitCommand, args), result.ExitCode, text.FirstLine(result.Stderr))
	}

	changed := changedPaths(result.Stdout)
	if len(changed) == 0 {
		return nil
	}

	return fmt.Errorf("bd init would commit uncommitted changes to %s; commit or stash them first",
		strings.Join(changed, ", "))
}

// changedPaths is the path out of every line `git status --porcelain` wrote: two status characters,
// a space, and then the path. A rename is reported as "old -> new", and the new name is the one that
// would be committed.
func changedPaths(stdout string) []string {
	var paths []string

	for _, line := range strings.Split(stdout, "\n") {
		if len(line) < 4 {
			continue
		}

		path := strings.TrimSpace(line[3:])

		if _, renamed, found := strings.Cut(path, " -> "); found {
			path = renamed
		}

		paths = append(paths, path)
	}

	return paths
}

// probe asks a tool a question in the project's directory. A non-zero exit is the answer, not a
// failure; a tool that could not be started at all is the error, phrased the way extension.go phrases
// the same thing.
func (i *Initialize) probe(ctx context.Context, dir, name string, args ...string) (CommandResult, error) {
	result, err := i.runner.Run(ctx, dir, name, args...)
	if err != nil {
		return CommandResult{}, fmt.Errorf("run %s: %w", commandLine(name, args), err)
	}

	return result, nil
}

// commandLine is how a command reads back to the person who has to run it themselves.
func commandLine(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

// requiredTools is what this particular run needs, and all it ever needs are bd and git, because
// a run initialises Beads and Beads sits inside a git repository. The extension step copies the
// embedded tree to the harness's own skills directory, so no foreign CLI is needed. A run over a
// current install has no Beads step, so it needs git alone, which the hook step's paths come from.
func requiredTools(request Request) []requiredTool {
	if request.Current {
		return []requiredTool{{name: gitCommand}}
	}

	return []requiredTool{
		{name: beadsCommand, remedy: "brew install beads"},
		{name: gitCommand},
	}
}
