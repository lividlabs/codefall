// Package application holds create's use case. It owns the gateway interfaces the steps need and
// depends on nothing but the standard library, mo, create's own domain, and the pure shared modules.
package application

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/samber/mo"

	"github.com/lividlabs/codefall/cli/internal/create/internal/domain"
)

// gitCommand is how git is invoked. It is the only tool create needs itself; the tools init needs are
// init's to check.
const gitCommand = "git"

// FileSystem is create's view of the disk (one gateway role): whether the project's directory is
// already taken, and the directory and files a new project starts with.
type FileSystem interface {
	// Exists reports whether anything is at path. A missing path is false, not an error.
	Exists(path string) (bool, error)
	// MkdirAll creates path and every parent it needs.
	MkdirAll(path string) error
	// WriteFile writes data to path, replacing whatever was there.
	WriteFile(path string, data []byte) error
}

// CommandRunner locates and runs git (one gateway role).
type CommandRunner interface {
	// LookPath returns where a tool lives, or None when it is not on PATH (ADR-GO-03).
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

// Request is what presentation hands the use case. Dir is absolute and Title is what the README is
// headed with. A description and a remote are both optional, so each is an Option (ADR-GO-03).
type Request struct {
	Dir         string
	Title       string
	Description mo.Option[string]
	Remote      mo.Option[string]
}

// Observer watches a run step by step, so a terminal can show what is happening while it happens.
type Observer interface {
	// StepStarted is called before a step does its work.
	StepStarted(step domain.Step)
	// StepFinished is called after a step has done its work, and not at all for a step that failed.
	StepFinished(result domain.StepResult)
}

// Create starts a project from nothing: a directory, a git repository with its remote, a README and a
// .gitignore, and the commit that holds them. Setting the project up for codefall is init's job, and
// the command runs init once this is done.
type Create struct {
	files  FileSystem
	runner CommandRunner
}

// NewCreate builds the use case over its gateways.
func NewCreate(files FileSystem, runner CommandRunner) *Create {
	return &Create{files: files, runner: runner}
}

type step struct {
	domain.Step
	run func(ctx context.Context, request Request) (domain.StepResult, error)
}

// Scaffold performs every step in order, telling the observer as each one starts and finishes. The
// first step that fails ends the run, and its error names the step.
//
// What would stop the run is checked before the first step, so a directory that is already taken
// or a missing git leaves nothing behind.
func (c *Create) Scaffold(ctx context.Context, request Request, observer Observer) (domain.Report, error) {
	if observer == nil {
		observer = silentObserver{}
	}

	if err := c.preflight(request); err != nil {
		return domain.Report{}, err
	}

	steps := []step{
		{Step: domain.DirectoryStep, run: c.directory},
		{Step: domain.RepositoryStep, run: c.repository},
		{Step: domain.RemoteStep, run: c.remote},
		{Step: domain.FilesStep, run: c.writeFiles},
		{Step: domain.CommitStep, run: c.commit},
	}

	results := make([]domain.StepResult, 0, len(steps))

	for _, step := range steps {
		if err := ctx.Err(); err != nil {
			return domain.Report{}, fmt.Errorf("create: %w", err)
		}

		observer.StepStarted(step.Step)

		result, err := step.run(ctx, request)
		if err != nil {
			return domain.Report{}, fmt.Errorf("%s: %w", step.ID, err)
		}

		observer.StepFinished(result)

		results = append(results, result)
	}

	return domain.NewReport(results...), nil
}

// Push sends what the project has committed to the remote and makes it the branch's upstream. It is
// separate from Scaffold because it runs after init, so bd init's commit goes with the first one.
func (c *Create) Push(ctx context.Context, dir string) (domain.StepResult, error) {
	if err := c.git(ctx, dir, "push", "--set-upstream", domain.RemoteName, "HEAD"); err != nil {
		return domain.StepResult{}, err
	}

	return domain.PushStep.Done("pushed to " + domain.RemoteName), nil
}

// preflight reports every reason the run cannot start, not just the first.
func (c *Create) preflight(request Request) error {
	var problems []string

	if c.runner.LookPath(gitCommand).IsAbsent() {
		problems = append(problems, "git is not on PATH")
	}

	taken, err := c.files.Exists(request.Dir)
	if err != nil {
		return fmt.Errorf("check %s: %w", request.Dir, err)
	}

	if taken {
		problems = append(problems, request.Dir+" already exists; create makes a new directory")
	}

	if len(problems) == 0 {
		return nil
	}

	return errors.New(strings.Join(problems, "; "))
}

func (c *Create) directory(_ context.Context, request Request) (domain.StepResult, error) {
	if err := c.files.MkdirAll(request.Dir); err != nil {
		return domain.StepResult{}, fmt.Errorf("create %s: %w", request.Dir, err)
	}

	return domain.DirectoryStep.Done("created " + request.Dir), nil
}

func (c *Create) repository(ctx context.Context, request Request) (domain.StepResult, error) {
	if err := c.git(ctx, request.Dir, "init"); err != nil {
		return domain.StepResult{}, err
	}

	return domain.RepositoryStep.Done("initialized a git repository"), nil
}

func (c *Create) remote(ctx context.Context, request Request) (domain.StepResult, error) {
	url, ok := request.Remote.Get()
	if !ok {
		return domain.RemoteStep.Skipped("no remote given"), nil
	}

	if err := c.git(ctx, request.Dir, "remote", "add", domain.RemoteName, url); err != nil {
		return domain.StepResult{}, err
	}

	return domain.RemoteStep.Done(fmt.Sprintf("added %s: %s", domain.RemoteName, url)), nil
}

func (c *Create) writeFiles(_ context.Context, request Request) (domain.StepResult, error) {
	for _, file := range []struct {
		name string
		body string
	}{
		{domain.ReadmeName, domain.Readme(request.Title, request.Description)},
		{domain.GitIgnoreName, domain.GitIgnore},
	} {
		if err := c.files.WriteFile(filepath.Join(request.Dir, file.name), []byte(file.body)); err != nil {
			return domain.StepResult{}, fmt.Errorf("write %s: %w", file.name, err)
		}
	}

	return domain.FilesStep.Done("wrote " + domain.ReadmeName + " and " + domain.GitIgnoreName), nil
}

func (c *Create) commit(ctx context.Context, request Request) (domain.StepResult, error) {
	if err := c.git(ctx, request.Dir, "add", "--", domain.ReadmeName, domain.GitIgnoreName); err != nil {
		return domain.StepResult{}, err
	}

	if err := c.git(ctx, request.Dir, "commit", "--message", domain.InitialCommitMessage); err != nil {
		return domain.StepResult{}, err
	}

	return domain.CommitStep.Done(fmt.Sprintf("committed %s and %s as %q",
		domain.ReadmeName, domain.GitIgnoreName, domain.InitialCommitMessage)), nil
}

// git runs one git command in dir and turns a non-zero exit into an error. The detail is git's last
// line rather than its first: git puts context first and the reason last, as in "fatal: unable to
// auto-detect email address" under a commit, or "error: failed to push some refs" under a push.
func (c *Create) git(ctx context.Context, dir string, args ...string) error {
	line := strings.Join(append([]string{gitCommand}, args...), " ")

	result, err := c.runner.Run(ctx, dir, gitCommand, args...)
	if err != nil {
		return fmt.Errorf("run %s: %w", line, err)
	}

	if result.ExitCode == 0 {
		return nil
	}

	detail := lastLine(result.Stderr)
	if detail == "" {
		detail = lastLine(result.Stdout)
	}

	return fmt.Errorf("%s exited %d: %s", line, result.ExitCode, detail)
}

// lastLine is the last line of a tool's output that has anything on it, trimmed.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")

	return strings.TrimSpace(lines[len(lines)-1])
}

type silentObserver struct{}

func (silentObserver) StepStarted(domain.Step)        {}
func (silentObserver) StepFinished(domain.StepResult) {}
