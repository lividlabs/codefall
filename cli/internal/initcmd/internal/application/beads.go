package application

import (
	"context"
	"encoding/json"
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

// beadsRemotesArgs asks bd which Dolt remotes the database has, and beadsFirstPushArgs publishes the
// database to the default one. bd init wires a remote named origin at the git origin whenever the
// repository has one (bd 1.3), and pushes nothing to it. A remote that is wired and empty fails
// every pull: `bd sync` and `bd dolt pull` exit 1 with "no branches found in remote", so the
// project's first /codefall-refresh stops at its beads step. The first push is what gives the remote
// a branch, so the step makes it. Like the interaction-log default, it runs only on a database this
// step just made.
//
// The remotes are asked for rather than read out of what bd init printed, because the JSON is what
// bd offers a program. The push carries no --yes: that flag consents to adopting a remote where
// none is wired, and a push that follows a listed remote adopts nothing.
var (
	beadsRemotesArgs   = []string{"dolt", "remote", "list", "--json"}
	beadsFirstPushArgs = []string{"dolt", "push"}
)

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
	// state from preflight: all preflight passes on is that the run was allowed to start.
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
		return domain.StepResult{}, fmt.Errorf("%s exited %d: %s",
			commandLine(beadsCommand, args), result.ExitCode, firstLineOf(result))
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

	pushed, err := i.beadsFirstPush(ctx, request.Dir)
	if err != nil {
		return domain.StepResult{}, err
	}

	return domain.BeadsStep.Done(done + note + pushed), nil
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
		return fmt.Sprintf("; %s exited %d (%s), so the interaction log is at bd's default",
			commandLine(beadsCommand, beadsAuditOffArgs), result.ExitCode, firstLineOf(result)), nil
	}

	return ", with the interaction log off in its config.yaml", nil
}

// beadsFirstPush publishes the database bd init just made to the Dolt remote bd init wired, and says
// so as a clause of the step's sentence. A database with no remote has nothing to publish and adds
// nothing to the sentence: the project works on this machine alone, which is what bd init already
// said on stderr. A push bd refuses is reported in the same clause rather than failing the step,
// with the command that mends it: the database works without it, and what the project is left with
// is a remote that every pull fails against until that command has run. A second clone, which
// bd init fills from the remote, pushes nothing and exits 0.
func (i *Initialize) beadsFirstPush(ctx context.Context, dir string) (string, error) {
	wired, note, err := i.beadsRemoteWired(ctx, dir)
	if err != nil {
		return "", err
	}

	if !wired {
		return note, nil
	}

	result, err := i.probe(ctx, dir, beadsCommand, beadsFirstPushArgs...)
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return fmt.Sprintf("; %s exited %d (%s), so its Dolt remote is wired and empty: run `%s` before "+
			"the first /codefall-refresh", commandLine(beadsCommand, beadsFirstPushArgs), result.ExitCode,
			firstLineOf(result), commandLine(beadsCommand, beadsFirstPushArgs)), nil
	}

	return ", and pushed to its Dolt remote", nil
}

// beadsRemoteWired reports whether the database has a Dolt remote. bd answers with a JSON array, one
// object per remote, and an empty array when there are none. An answer that cannot be read is not a
// reason to push blind, so it is reported as a clause and the push is left to the person.
func (i *Initialize) beadsRemoteWired(ctx context.Context, dir string) (bool, string, error) {
	result, err := i.probe(ctx, dir, beadsCommand, beadsRemotesArgs...)
	if err != nil {
		return false, "", err
	}

	unread := fmt.Sprintf("; %s could not be read, so run `%s` before the first /codefall-refresh if "+
		"the database has a Dolt remote", commandLine(beadsCommand, beadsRemotesArgs),
		commandLine(beadsCommand, beadsFirstPushArgs))

	if result.ExitCode != 0 {
		return false, unread, nil
	}

	var remotes []json.RawMessage
	if err := json.Unmarshal([]byte(result.Stdout), &remotes); err != nil {
		return false, unread, nil //nolint:nilerr // an unreadable answer is a clause of the report, not a failed step
	}

	return len(remotes) > 0, "", nil
}

// firstLineOf is the one line of a failed command worth repeating: what it said on stderr, or on
// stdout when stderr was empty.
func firstLineOf(result CommandResult) string {
	if detail := text.FirstLine(result.Stderr); detail != "" {
		return detail
	}

	return text.FirstLine(result.Stdout)
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
