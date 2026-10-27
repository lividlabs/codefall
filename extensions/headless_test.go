package extensions

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// CODEFALL_HEADLESS is one variable with one meaning in every harness: a run that was given a prompt
// to answer, which the two runner scripts start, does not get a session notice. The runners set it,
// and the notice reads it, whichever harness runs the notice script or the OpenCode plugin. The
// merge guard reads nothing of the kind and stays on in every run.

const (
	noticeScript = "hooks/shared/codefall-session-notice.sh"
	pluginFile   = "hooks/opencode/codefall.js"
)

// The runners are two copies of one piece of logic, and each header says a change to one is made to
// the other. This is the one line of it a missing copy would leave a reviewer or a worker without.
func TestBothRunnersMarkEveryHarnessRunHeadless(t *testing.T) {
	for _, runner := range []string{"shared/run-agent.sh", "shared/run-worker.sh"} {
		body, err := os.ReadFile(runner)
		if err != nil {
			t.Fatalf("read %s: %v", runner, err)
		}

		if !bytes.Contains(body, []byte("\n  export CODEFALL_HEADLESS=1\n")) {
			t.Errorf("%s does not export CODEFALL_HEADLESS=1 in run_agent, ahead of every harness's run", runner)
		}
	}
}

// The OpenCode plugin has no script to read the variable for it, so it reads it itself. Nothing here
// runs the plugin, so the pin is textual.
func TestOpenCodePluginReadsTheHeadlessVariable(t *testing.T) {
	body, err := os.ReadFile(pluginFile)
	if err != nil {
		t.Fatalf("read %s: %v", pluginFile, err)
	}

	if !bytes.Contains(body, []byte("process.env.CODEFALL_HEADLESS")) {
		t.Errorf("%s does not read CODEFALL_HEADLESS, so a headless OpenCode run would be sent the notice", pluginFile)
	}
}

// The notice script is what Claude Code's and Codex's SessionStart entries run, and what the plugin
// runs beside the Beads prime. An empty repository has things to say (no local environment, no
// testing root, no Beads database), so a notice that stays silent under the variable is the
// variable's doing and not a project with nothing to fix.
func TestSessionNoticeSaysNothingWhenHeadless(t *testing.T) {
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("the notice needs %s: %v", tool, err)
		}
	}

	script, err := filepath.Abs(noticeScript)
	if err != nil {
		t.Fatal(err)
	}

	project := t.TempDir()

	gitInit := exec.Command("git", "init", "-q")
	gitInit.Dir = project

	if out, err := gitInit.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// The test's own environment may carry the variable, if the suite itself runs under a runner.
	env := make([]string, 0, len(os.Environ()))

	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "CODEFALL_HEADLESS=") {
			env = append(env, entry)
		}
	}

	notice := func(extra ...string) string {
		t.Helper()

		var stdout bytes.Buffer

		cmd := exec.Command("bash", script, project)
		cmd.Env = append(env, extra...)
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			t.Fatalf("running the notice: %v", err)
		}

		return stdout.String()
	}

	if got := notice(); !strings.HasPrefix(got, "codefall:") {
		t.Fatalf("notice = %q in an empty repository, want it to name what needs attention", got)
	}

	if got := notice("CODEFALL_HEADLESS=1"); got != "" {
		t.Errorf("notice = %q under CODEFALL_HEADLESS=1, want nothing", got)
	}
}
