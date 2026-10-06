package extensions

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The merge guard runs as a PreToolUse hook in every harness. These tests run the shipped script
// against a throwaway repository, with the payload a harness sends, and read its answer the way a
// harness does: exit 2 with the reason on stderr for Claude Code, Codex, and OpenCode, and a JSON
// decision on stdout for Antigravity.

const guardScript = "hooks/shared/codefall-block-merge-to-main.sh"

// A stand-in for gh: `gh pr view` prints the base branch a case chooses, and fails when it
// chooses none, which is what gh does for a PR it cannot find.
const fakeGh = `#!/bin/sh
[ -n "${FAKE_GH_BASE:-}" ] || exit 1
printf '%s\n' "$FAKE_GH_BASE"
`

type guardRepo struct {
	script string
	dir    string
	env    []string
}

// newGuardRepo makes a repository whose origin's default branch is protected and whose checked-out
// branch is current.
func newGuardRepo(t *testing.T, protected, current string) *guardRepo {
	t.Helper()

	for _, tool := range []string{"bash", "git", "python3"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("the merge guard needs %s: %v", tool, err)
		}
	}

	script, err := filepath.Abs(guardScript)
	if err != nil {
		t.Fatal(err)
	}

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(fakeGh), 0o755); err != nil {
		t.Fatal(err)
	}

	r := &guardRepo{
		script: script,
		dir:    t.TempDir(),
		env: append(os.Environ(),
			"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
			"GIT_CONFIG_GLOBAL="+os.DevNull,
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=codefall", "GIT_AUTHOR_EMAIL=codefall@example.com",
			"GIT_COMMITTER_NAME=codefall", "GIT_COMMITTER_EMAIL=codefall@example.com",
		),
	}

	r.git(t, "init", "-q")
	r.git(t, "symbolic-ref", "HEAD", "refs/heads/"+current)
	r.git(t, "commit", "-q", "--allow-empty", "-m", "init")
	r.git(t, "update-ref", "refs/remotes/origin/"+protected, "HEAD")
	r.git(t, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/"+protected)

	return r
}

func (r *guardRepo) git(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = r.env

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// run hands the guard a Claude Code payload carrying command, with gh answering base for any PR.
// It returns the exit code and stderr.
func (r *guardRepo) run(t *testing.T, command, base string, flags ...string) (int, string, string) {
	t.Helper()

	payload := map[string]any{"tool_input": map[string]any{"command": command}}
	if len(flags) > 0 && flags[0] == "--antigravity" {
		payload = map[string]any{"toolCall": map[string]any{"args": map[string]any{"CommandLine": command}}}
	}

	in, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.Command("bash", append([]string{r.script}, flags...)...)
	cmd.Dir = r.dir
	cmd.Env = append(r.env, "FAKE_GH_BASE="+base)
	cmd.Stdin = bytes.NewReader(in)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	var exit *exec.ExitError

	switch {
	case err == nil:
		return 0, stdout.String(), stderr.String()
	case errors.As(err, &exit):
		return exit.ExitCode(), stdout.String(), stderr.String()
	default:
		t.Fatalf("running the guard: %v", err)
		return 0, "", ""
	}
}

type guardCase struct {
	command string
	base    string // what gh reports as a PR's base; empty means gh cannot say
}

func (r *guardRepo) expectAllowed(t *testing.T, cases []guardCase) {
	t.Helper()

	for _, c := range cases {
		if code, _, stderr := r.run(t, c.command, c.base); code != 0 {
			t.Errorf("denied, want allowed (exit %d): %q\n%s", code, c.command, stderr)
		}
	}
}

func (r *guardRepo) expectDenied(t *testing.T, cases []guardCase, reason string) {
	t.Helper()

	for _, c := range cases {
		code, _, stderr := r.run(t, c.command, c.base)
		if code != 2 {
			t.Errorf("allowed, want denied (exit %d): %q", code, c.command)
			continue
		}

		if !strings.Contains(stderr, reason) {
			t.Errorf("denied %q with %q, want a reason containing %q", c.command, stderr, reason)
		}
	}
}

// Issue #177: a push was denied whenever the protected branch's name appeared anywhere in the
// line. A push is judged by its own refspecs, so these all go through.
func TestMergeGuardAllowsPushesToOtherBranches(t *testing.T) {
	r := newGuardRepo(t, "main", "feat")

	r.expectAllowed(t, []guardCase{
		{command: `git commit -q -m "VISION-001: forfeit" && git push -q -u origin vision/VISION-001-forfeit && gh pr create --base main --title "VISION-001" --body "Lands on main."`},
		{command: `git push -u origin docs/vision-001-active`},
		{command: `git push -u origin docs/vision-001-active && gh pr create --base main --fill`},
		{command: `git push --set-upstream origin feat`},
		{command: `git push --force-with-lease origin feat; echo main`},
		{command: `git push origin HEAD`},
		{command: `git push origin HEAD:feat`},
		{command: `git push origin main:feat`},
		{command: `git push origin feat:maintenance`},
		{command: `git push origin domain`},
		{command: `git push -o main origin feat`},
		{command: `git push origin tag main`},
		{command: `git push origin feat 2>&1 | tee main.log`},
		{command: `git fetch origin main && git rebase origin/main && git push --force-with-lease origin feat`},
		{command: `git push`},
		{command: `git push origin`},
		{command: `git status`},
		{command: `git merge main`},
	})
}

func TestMergeGuardDeniesPushesToTheProtectedBranch(t *testing.T) {
	r := newGuardRepo(t, "main", "feat")

	r.expectDenied(t, []guardCase{
		{command: `git push origin main`},
		{command: `git push -u origin main`},
		{command: `git push --set-upstream origin main`},
		{command: `git push -q --force origin main`},
		{command: `git push origin HEAD:main`},
		{command: `git push origin +main`},
		{command: `git push origin +HEAD:main`},
		{command: `git push origin feat:refs/heads/main`},
		{command: `git push origin refs/heads/main`},
		{command: `git push origin :main`},
		{command: `git push origin --delete main`},
		{command: `git push origin feat main`},
		{command: `git push origin -- main`},
		{command: `git push -o ci.skip origin main`},
		{command: `git push origin "HEAD:main"`},
		{command: `git push origin 'refs/heads/*:refs/heads/*'`},
		{command: `git push origin :`},
		{command: `git -C . push origin main`},
		{command: `/usr/bin/git push origin main`},
		{command: `GIT_TRACE=0 git push origin main`},
	}, "denied: 'git push' targeting 'main'.")

	r.expectDenied(t, []guardCase{
		{command: `git push --all origin`},
		{command: `git push --mirror origin`},
	}, "pushes every branch, 'main' among them.")
}

func TestMergeGuardJudgesEachCommandInACompoundLine(t *testing.T) {
	r := newGuardRepo(t, "main", "feat")

	r.expectDenied(t, []guardCase{
		{command: `git commit -m x && git push origin feat && git push origin main`},
		{command: `npm test; git push origin HEAD:main`},
		{command: `npm test || git push origin main`},
		{command: `git push origin main 2>&1 | tail -1`},
		{command: `git push origin main & wait`},
		{command: "git status\ngit push origin main"},
		{command: `(cd sub && git push origin main)`},
		{command: `bash -c "git fetch && git push origin main"`},
		{command: `sh -ec 'git push origin HEAD:main'`},
		{command: `eval git push origin main`},
		// An unbalanced quote defeats shlex; the line is split on whitespace instead.
		{command: `git push origin main && echo "unterminated`},
		// gh pr merge into an epic is allowed, and the push after it is still read.
		{command: `gh pr merge 12 --squash && git push origin main`, base: "epic/E-1"},
	}, "denied: 'git push' targeting 'main'.")
}

func TestMergeGuardOnTheProtectedBranch(t *testing.T) {
	r := newGuardRepo(t, "main", "main")

	r.expectDenied(t, []guardCase{
		{command: `git push`},
		{command: `git push origin`},
		{command: `git push -u origin`},
		{command: `git push --force-with-lease`},
		{command: `git push && echo pushed`},
		{command: `git add -A && git commit -m x && git push`},
	}, "denied: bare 'git push' while on 'main'.")

	r.expectDenied(t, []guardCase{
		{command: `git push origin HEAD`},
		{command: `git push origin @`},
	}, "denied: 'git push' targeting 'main'.")

	r.expectDenied(t, []guardCase{
		{command: `git merge feat`},
		{command: `git fetch && git merge --ff-only origin/feat`},
	}, "denied: 'git merge' while on 'main'.")

	r.expectAllowed(t, []guardCase{
		{command: `git push origin feat`},
		{command: `git push -u origin HEAD:feat`},
		{command: `git status`},
	})
}

// The protected branch is origin's default, read from refs/remotes/origin/HEAD, and only falls
// back to main when that is unset.
func TestMergeGuardProtectsOriginsDefaultBranch(t *testing.T) {
	r := newGuardRepo(t, "trunk", "feat")

	r.expectDenied(t, []guardCase{
		{command: `git push origin trunk`},
		{command: `git push origin HEAD:refs/heads/trunk`},
	}, "denied: 'git push' targeting 'trunk'.")

	r.expectAllowed(t, []guardCase{
		{command: `git push origin main`},
		{command: `git push origin feat && gh pr create --base trunk --fill`},
	})
}

func TestMergeGuardDeniesMergesIntoTheProtectedBranch(t *testing.T) {
	r := newGuardRepo(t, "main", "feat")

	r.expectDenied(t, []guardCase{
		{command: `gh stack merge`},
		{command: `git fetch && gh stack merge --yes`},
	}, "denied: 'gh stack merge' lands a stack on 'main'.")

	r.expectDenied(t, []guardCase{
		{command: `gh pr merge 12 --squash`, base: "main"},
		{command: `gh pr merge --squash`, base: "main"},
		{command: `gh pr merge 12 --squash`},
	}, "denied: 'gh pr merge' into 'main'")

	r.expectAllowed(t, []guardCase{
		{command: `gh pr merge 12 --squash`, base: "epic/E-1"},
		{command: `gh pr create --base main --fill`},
	})
}

// Antigravity reads a JSON decision on stdout, and the script exits 0 either way.
func TestMergeGuardAnswersAntigravityInJSON(t *testing.T) {
	r := newGuardRepo(t, "main", "feat")

	code, stdout, stderr := r.run(t, `git push origin HEAD:main`, "", "--antigravity")
	if code != 0 {
		t.Fatalf("exit %d, want 0: %s", code, stderr)
	}

	var decision struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(stdout), &decision); err != nil {
		t.Fatalf("stdout is not a JSON decision: %v\n%s", err, stdout)
	}

	if decision.Decision != "deny" || !strings.Contains(decision.Reason, "targeting 'main'") {
		t.Errorf("decision %+v, want a deny targeting 'main'", decision)
	}

	code, stdout, _ = r.run(t, `git push -u origin docs/vision-001-active && gh pr create --base main`, "", "--antigravity")
	if code != 0 || stdout != "" {
		t.Errorf("exit %d with %q, want exit 0 and no decision", code, stdout)
	}
}
