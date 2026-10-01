package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/text"
)

// beadsCommand is how Beads is invoked, and the arguments init gives it.
//
// --non-interactive because init has already asked its questions. --skip-agents because bd would
// otherwise append its own section to AGENTS.md and CLAUDE.md and write .claude/settings.json,
// .codex/, and .agents/ — codefall owns those files, and the context bd's section points at reaches
// a session through the hook step instead.
const beadsCommand = "bd"

var beadsInitArgs = []string{"init", "--non-interactive", "--skip-agents"}

// beadsAuditOffArgs turns bd's interaction log off explicitly in the database's config.yaml, right
// after bd init. bd 1.3 leaves the key commented out and off; writing it makes the project's choice
// visible in the file, so a teammate who finds .beads/interactions.jsonl appearing in a pull request
// can see it was turned on deliberately and where. bd config set is asked rather than the file
// edited because bd knows where the database is (BEADS_DIR relocates it) and codefall does not.
// It runs only on a database this step just made: a project that already has Beads may have
// turned the log on, and that is its decision to keep.
var beadsAuditOffArgs = []string{"config", "set", "audit.enabled", "false"}

// skipHooksArg is what an install below the repository root adds. bd init points the clone's
// core.hooksPath at its own .beads/hooks, and that setting is one value for the whole repository:
// claiming it from a subdirectory would take the repository's git hooks away from everyone working
// in that clone, and bd points it at the root's .beads/hooks even when it wrote the database
// somewhere below, so the path it names need not exist at all (bd 1.2.2). At the root, where Beads
// is the whole project's tracker, the setting is bd's to make.
const skipHooksArg = "--skip-hooks"

// What bd says when it committed what it wrote, and the message it committed under. bd commits
// unconditionally and has no flag to stop it, so the report says so rather than letting a commit
// nobody asked for turn up in the log unexplained. What went into it is left unnamed: bd stages
// .beads/ and .gitignore, and also every one of the files preflight guards, so the list depends on
// what the directory had.
const (
	beadsCommitted     = "Committed beads files to git"
	beadsCommitMessage = "bd init: initialize beads issue tracking"
)

// beads is the third step of a run: it creates the issue database that codefall's agents work
// against. Preflight has already established that bd is on PATH, that this is a git work tree, and
// that nothing is staged — the three things bd init would otherwise decide for itself.
func (i *Initialize) beads(ctx context.Context, request Request) (domain.StepResult, error) {
	// Preflight asked bd this same question, to decide whether what the directory had waiting was
	// any of its business. It is asked again rather than carried across, because a step takes no
	// state from preflight: what preflight leaves behind is a run that was allowed to start.
	initialized, err := i.beadsInitialized(ctx, request.Dir)
	if err != nil {
		return domain.StepResult{}, err
	}

	if initialized {
		return domain.BeadsStep.Skipped("Beads is already initialized here"), nil
	}

	prefix, err := i.repositoryPrefix(ctx, request.Dir)
	if err != nil {
		return domain.StepResult{}, err
	}

	args := beadsInitArgs
	done := "initialized Beads"

	if prefix != "" {
		args = append(slices.Clone(args), skipHooksArg)
		done += " without its git hooks, which belong to the whole repository"
	}

	result, err := i.probe(ctx, request.Dir, beadsCommand, args...)
	if err != nil {
		return domain.StepResult{}, err
	}

	if result.ExitCode != 0 {
		detail := text.FirstLine(result.Stderr)
		if detail == "" {
			detail = text.FirstLine(result.Stdout)
		}

		return domain.StepResult{}, fmt.Errorf("%s exited %d: %s",
			commandLine(beadsCommand, args), result.ExitCode, detail)
	}

	// bd warns on stderr about things a fresh database has not got yet — no Dolt remote, for one —
	// and it does that on a run that worked. A successful run has nothing to complain about, so what
	// it said there is dropped.
	if strings.Contains(result.Stdout, beadsCommitted) {
		done = fmt.Sprintf("%s (bd init committed what it wrote as %q)", done, beadsCommitMessage)
	}

	note, err := i.beadsAuditOff(ctx, request.Dir)
	if err != nil {
		return domain.StepResult{}, err
	}

	return domain.BeadsStep.Done(done + note), nil
}

// beadsAuditOff writes the interaction-log default into the database bd init just made, and says so
// as a clause of the step's sentence. bd refusing is reported in the same clause rather than failing
// the step: the log is off by default anyway, so the database works either way, and what the
// project loses is only the explicit line in its config.yaml.
func (i *Initialize) beadsAuditOff(ctx context.Context, dir string) (string, error) {
	result, err := i.probe(ctx, dir, beadsCommand, beadsAuditOffArgs...)
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		detail := text.FirstLine(result.Stderr)
		if detail == "" {
			detail = text.FirstLine(result.Stdout)
		}

		return fmt.Sprintf("; %s exited %d (%s), so the interaction log is at bd's default",
			commandLine(beadsCommand, beadsAuditOffArgs), result.ExitCode, detail), nil
	}

	return ", with the interaction log off in its config.yaml", nil
}

// beadsInitialized asks bd whether this directory already has a database. `bd info` is the question
// doctor asks for the same reason: BEADS_DIR relocates .beads/, so looking for the directory is not
// the same question, and re-running bd init where it has already run is an error rather than a
// no-op.
func (i *Initialize) beadsInitialized(ctx context.Context, dir string) (bool, error) {
	result, err := i.probe(ctx, dir, beadsCommand, "info")
	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}
