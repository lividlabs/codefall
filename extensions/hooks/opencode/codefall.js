// codefall's OpenCode plugin: the same hooks the JSON harnesses get, in OpenCode 2's plugin
// form. Shell commands pass through the shared merge guard, and a new session gets primed with
// what Beads knows and told what needs attention in the project.
//
// The guard and the notice are delegated to the scripts in hooks/shared/ — one logic source
// for every harness; this file is only the adapter from OpenCode's shapes to them.
//
// OpenCode 2 loads a plugin only as a default export with an id and a setup function, and
// codefall supports no other OpenCode. The export is a plain object, not a Plugin.define call,
// because a project need not have @opencode/plugin installed: the loader checks the shape, not
// where it came from. OpenCode 2 has no `$` helper, so Node's modules run the processes and read
// the files, which the runtime it embeds (Bun, in 2.1.4) provides.
//
// Both scripts are found from this file's own directory, not from the worktree root: init
// installs the plugin at <install>/.opencode/plugins/ and the scripts at <install>/.codefall/,
// and the install is the repository root only when init was run there.
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { resolve } from "node:path";

const guard = `${import.meta.dir}/../../.codefall/hooks/shared/codefall-block-merge-to-main.sh`;
const notice = `${import.meta.dir}/../../.codefall/hooks/shared/codefall-session-notice.sh`;

// Run a program with no shell in between and collect what it printed. Whatever the program is
// given in `input` goes in through piped stdin, because the command it carries is
// model-controlled. A program that cannot start settles with a null code, not a rejection.
const run = (program, args, { cwd, input } = {}) =>
  new Promise((settle) => {
    let stdout = "";
    let stderr = "";
    let child;
    try {
      child = spawn(program, args, {
        cwd,
        stdio: [input === undefined ? "ignore" : "pipe", "pipe", "pipe"],
      });
    } catch (error) {
      settle({ code: null, stdout, stderr: String(error) });
      return;
    }
    child.stdout.on("data", (chunk) => (stdout += chunk));
    child.stderr.on("data", (chunk) => (stderr += chunk));
    child.on("error", (error) => settle({ code: null, stdout, stderr: String(error) }));
    child.on("close", (code) => settle({ code, stdout, stderr }));
    if (child.stdin) {
      // A script that answers without reading its input closes the pipe under the write.
      child.stdin.on("error", () => {});
      child.stdin.end(input);
    }
  });

export default {
  id: "codefall",

  async setup(ctx) {
    const directory = ctx.location.directory;

    // Deny merges and pushes to the default branch. The script answers exit 2 for an
    // objection, with the reason on stderr already worded as codefall's; OpenCode blocks the
    // tool call on a thrown error, and the model reads the error's message as the tool's
    // result. OpenCode 2 names its shell tool "shell".
    await ctx.tool.hook("execute.before", async (event) => {
      if (event.tool !== "shell") return;

      const command = event.input?.command;
      if (typeof command !== "string") return;

      // Missing script means the extension is not installed here (or the project moved); the
      // guard stands down rather than throwing on every shell call. Fail-open is the deliberate
      // trade: this is one layer, and a repository ruleset protecting the default branch is the
      // backstop — the shared script states the same limit about itself. A script that cannot
      // start is the same case, and the other harnesses treat a hook that fails to run the same
      // way.
      if (!existsSync(guard)) return;

      // The script speaks the Claude-shaped payload, so the shape is rebuilt here rather than
      // read from the event directly. It reads the branch it is standing on, so it stands where
      // the command will: the shell tool's own working directory when it names one.
      const workdir = typeof event.input.workdir === "string" ? event.input.workdir : ".";
      const result = await run("bash", [guard], {
        cwd: resolve(directory, workdir),
        input: JSON.stringify({ tool_input: { command } }),
      });
      if (result.code === null) return;

      if (result.code !== 0) {
        throw new Error(result.stderr.trim() || "codefall: the merge guard denied this command.");
      }
    });

    // Prime a fresh session with what Beads knows and with what the project needs doing: bd's
    // report and codefall's notice go in as one synthetic message without triggering a
    // response. Anything missing (bd, the script, the session) is skipped — a session hook
    // must never break the session that starts it.
    const primed = async (event) => {
      // A headless run — a reviewer or worker the runner scripts start — answers the prompt it
      // was given; a notice that sends it off to refresh the checkout is a run that never
      // answers. The runners set this variable for every harness they start.
      if (process.env.CODEFALL_HEADLESS) return;

      // Events reach every plugin instance on the server, whichever directory the session is
      // in; this instance primes the sessions of its own directory only. A session with a
      // parent is a subagent's, and the parent was primed.
      if (event.location?.directory !== directory || event.data?.parentID) return;

      const sessionID = event.data?.sessionID;
      if (!sessionID) return;

      const parts = [];

      try {
        const prime = await run("bd", ["prime"], { cwd: directory });
        if (prime.code === 0) {
          const text = prime.stdout.trim();
          if (text) parts.push(text);
        }
      } catch {
        // A failed prime is a missed convenience, not an error worth a session over.
      }

      // The notice says what needs attention — the checkout behind the default branch, an
      // environment that has not been refreshed, a testing root nobody declared, a Beads
      // precondition — and prints nothing when the project is current. Plain text, because
      // that is what goes into a message; --hook-json is for the harnesses whose SessionStart
      // entry reads JSON. It reports only: nothing here pulls or updates.
      try {
        if (existsSync(notice)) {
          const reported = await run("bash", [notice], { cwd: directory });
          const text = reported.stdout.trim();
          if (reported.code === 0 && text) parts.push(text);
        }
      } catch {
        // The same trade the prime makes: the session opens either way.
      }

      if (parts.length === 0) return;

      try {
        // resume: false adds the message without starting a turn; the model reads it with the
        // first prompt.
        await ctx.session.synthetic({ sessionID, text: parts.join("\n\n"), resume: false });
      } catch {
        // A session that will not take the context still starts, which is the point.
      }
    };

    // OpenCode 2 has no session-start hook, so the session-created event stands in for one.
    // The subscription is not awaited — setup returns first — and each event is handled on
    // its own, so a slow prime holds up no other session.
    const stop = new AbortController();
    (async () => {
      try {
        for await (const event of ctx.event.subscribe({ signal: stop.signal })) {
          if (event.type === "session.created") primed(event).catch(() => {});
        }
      } catch {
        // A subscription that ends costs the notice and nothing else.
      }
    })();

    // Unloading the plugin disposes the hook; the subscription is the one thing to end by hand.
    return () => stop.abort();
  },
};
