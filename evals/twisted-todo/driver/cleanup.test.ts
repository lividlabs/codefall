// The end-of-run cleanup against a scratch repository: worker worktrees (clean, dirty, locked, and
// one whose directory is already gone) and dummy servers in the project and in a worktree, beside
// processes it must not touch: a server outside the project and a process in the project that
// listens on nothing.
//
//   node --test cleanup.test.ts

import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, realpathSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { cleanUpProject } from "./cleanup.ts";

const git = (cwd: string, ...args: string[]) => execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });

const SERVER = `require("node:http").createServer((q, s) => s.end("ok")).listen(0, "127.0.0.1", () => require("node:fs").writeFileSync(process.argv[1], "up"))`;

/** Starts a detached process in `cwd` through a shell, so it is not this test's child, and returns its pid. */
function startDetached(cwd: string, command: string): number {
  const out = execFileSync("sh", ["-c", `nohup ${command} >/dev/null 2>&1 & echo $!`], { cwd, encoding: "utf8" });
  return Number(out.trim());
}

function startServer(cwd: string, readyFile: string): number {
  const pid = startDetached(cwd, `node -e '${SERVER}' ${readyFile}`);
  const until = Date.now() + 10_000;
  while (!existsSync(readyFile)) {
    if (Date.now() > until) throw new Error(`server in ${cwd} did not start`);
    execFileSync("sleep", ["0.1"]);
  }
  return pid;
}

function alive(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

test("stops the project's servers and removes its worktrees, and touches nothing else", () => {
  const scratch = realpathSync(mkdtempSync(join(tmpdir(), "cleanup-test-")));
  const project = join(scratch, "project");
  const outside = join(scratch, "outside");
  mkdirSync(project);
  mkdirSync(outside);
  const pids: number[] = [];
  try {
    git(project, "init", "-q", "-b", "main");
    writeFileSync(join(project, "README.md"), "scratch\n");
    git(project, "add", "-A");
    git(project, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "init");

    const wt = (name: string) => join(project, ".claude", "worktrees", name);
    for (const name of ["agent-clean", "agent-dirty", "agent-locked", "agent-gone"]) git(project, "worktree", "add", "-q", "-b", name, wt(name));
    writeFileSync(join(wt("agent-dirty"), "README.md"), "uncommitted\n");
    writeFileSync(join(wt("agent-dirty"), "untracked.txt"), "untracked\n");
    git(project, "worktree", "lock", wt("agent-locked"));
    rmSync(wt("agent-gone"), { recursive: true, force: true });

    // A server started the way local.sh starts it, with its PID file, in the project directory.
    mkdirSync(join(project, "data"));
    const projectServer = startServer(project, join(scratch, "ready-project"));
    writeFileSync(join(project, "data", "server.pid"), `${projectServer}\n`);
    // A server a worker started by hand in its worktree, from a subdirectory, with no PID file.
    mkdirSync(join(wt("agent-dirty"), "src"));
    const worktreeServer = startServer(join(wt("agent-dirty"), "src"), join(scratch, "ready-worktree"));
    // Not the run's: a server outside the project, and a process in the project that does not listen.
    const outsideServer = startServer(outside, join(scratch, "ready-outside"));
    const sleeper = startDetached(project, "sleep 60");
    pids.push(projectServer, worktreeServer, outsideServer, sleeper);

    const lines: string[] = [];
    const report = cleanUpProject(project, "test", (line) => lines.push(line));

    assert.deepEqual(report.serversStopped.map((s) => s.pid).sort(), [projectServer, worktreeServer].sort());
    assert.ok(report.serversStopped.every((s) => s.listening.length === 1 && s.signal === "SIGTERM"), JSON.stringify(report.serversStopped));
    assert.ok(!alive(projectServer) && !alive(worktreeServer), "the project's servers are stopped");
    assert.ok(alive(outsideServer), "the server outside the project is untouched");
    assert.ok(alive(sleeper), "a process in the project that listens on nothing is untouched");
    assert.ok(!existsSync(join(project, "data", "server.pid")), "the stopped server's PID file is removed");

    assert.deepEqual(report.worktreesRemoved.sort(), [wt("agent-clean"), wt("agent-dirty"), wt("agent-locked")].sort());
    assert.deepEqual(report.worktreesPruned, [wt("agent-gone")]);
    assert.deepEqual(report.serversLeft, []);
    assert.deepEqual(report.worktreesLeft, []);
    const remaining = git(project, "worktree", "list", "--porcelain").split("\n").filter((l) => l.startsWith("worktree "));
    assert.deepEqual(remaining, [`worktree ${project}`]);
    assert.match(git(project, "branch", "--list", "agent-dirty"), /agent-dirty/, "branches stay");

    assert.match(lines.at(-1)!, /stopped 2 server\(s\).*removed 3 worktree\(s\), pruned 1 stale worktree record/);

    // A second run finds nothing left to do.
    const again = cleanUpProject(project, "again", () => {});
    assert.deepEqual([again.serversStopped.length, again.worktreesRemoved.length, again.worktreesPruned.length], [0, 0, 0]);
  } finally {
    for (const pid of pids) if (alive(pid)) process.kill(pid, "SIGKILL");
    rmSync(scratch, { recursive: true, force: true });
  }
});

test("a project directory that does not exist is a no-op", () => {
  const report = cleanUpProject(join(tmpdir(), "cleanup-test-missing-" + process.pid), "test", () => {});
  assert.deepEqual([report.serversStopped.length, report.worktreesRemoved.length], [0, 0]);
});
