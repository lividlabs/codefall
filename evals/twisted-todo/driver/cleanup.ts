// What the driver leaves behind on the laptop when a run ends, passed, failed, or interrupted:
// nothing it or a verb started. It stops the fixture servers and removes implement's worker
// worktrees, and returns what it did so the run summary can say so.
//
// A server is found by where it runs, not by its port or a PID file alone. Verbs start the fixture
// with `scripts/local.sh start` from the project directory and from worker worktrees, sometimes on
// another port (`PORT=3100`), and a worker may start one by hand; a PID file is only written by
// `local.sh` and is overwritten by the next start. So a candidate is any process listening on a TCP
// port, plus any process named in a `data/server.pid` file, and it is stopped only when its working
// directory is the project directory, one of its worktrees, or a directory under either. Nothing
// else on the machine has its working directory inside a throwaway checkout, so an unrelated
// process is never a match. Beads' own Dolt server is left alone: bd owns it.

import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, readlinkSync, realpathSync, rmSync } from "node:fs";
import { join, sep } from "node:path";

export type StoppedServer = { pid: number; command: string; args: string; cwd: string; listening: string[]; signal: "SIGTERM" | "SIGKILL" };

export type CleanupReport = {
  /** Why the cleanup ran: the chain ended, the driver threw, or a signal arrived. */
  reason: string;
  at: string;
  serversStopped: StoppedServer[];
  serversLeft: { pid: number; cwd: string; error: string }[];
  /** Linked worktrees removed with `git worktree remove`. */
  worktreesRemoved: string[];
  /** Worktree records whose directory was already gone; `git worktree prune` cleared them. */
  worktreesPruned: string[];
  worktreesLeft: { path: string; error: string }[];
};

/** Commands that listen from inside the project but are not the fixture's to stop. */
const NOT_OURS = new Set(["dolt"]);

export function cleanUpProject(projectDir: string, reason: string, log: (line: string) => void): CleanupReport {
  const report: CleanupReport = { reason, at: new Date().toISOString(), serversStopped: [], serversLeft: [], worktreesRemoved: [], worktreesPruned: [], worktreesLeft: [] };
  if (!existsSync(projectDir)) {
    log(`cleanup: ${projectDir} does not exist; nothing to stop or remove`);
    return report;
  }

  const worktrees = listWorktrees(projectDir);
  const roots = [projectDir, ...worktrees.map((w) => w.path)].filter(existsSync).map((p) => realpathSync(p));

  // Servers first: a server whose worktree is removed under it keeps running from a deleted directory.
  stopServers(roots, report, log);
  removeWorktrees(projectDir, worktrees, report, log);

  log(describeCleanup(report));
  return report;
}

/** One line for the log: what the cleanup stopped and removed, and anything it could not. */
export function describeCleanup(report: CleanupReport): string {
  return (
    `cleanup (${report.reason}): stopped ${report.serversStopped.length} server(s)` +
    (report.serversStopped.length ? ` [${report.serversStopped.map((s) => `pid ${s.pid} ${s.listening.join(" ") || s.command}`).join("; ")}]` : "") +
    `, removed ${report.worktreesRemoved.length} worktree(s)` +
    (report.worktreesPruned.length ? `, pruned ${report.worktreesPruned.length} stale worktree record(s)` : "") +
    (report.serversLeft.length || report.worktreesLeft.length ? `; left ${report.serversLeft.length} server(s) and ${report.worktreesLeft.length} worktree(s), see summary.json` : "")
  );
}

function stopServers(roots: string[], report: CleanupReport, log: (line: string) => void): void {
  const listeners = listeningProcesses();
  const candidates = new Map<number, { command: string; listening: string[] }>(listeners);
  const pidFiles = new Map<number, string>();
  for (const root of roots) {
    const file = join(root, "data", "server.pid");
    const pid = existsSync(file) ? Number(readFileSync(file, "utf8").trim()) : NaN;
    if (Number.isInteger(pid) && pid > 0 && alive(pid)) {
      pidFiles.set(pid, file);
      if (!candidates.has(pid)) candidates.set(pid, { command: "", listening: [] });
    }
  }

  const own = new Set([process.pid, process.ppid]);
  for (const [pid, info] of candidates) {
    if (own.has(pid)) continue;
    const cwd = processCwd(pid);
    if (!cwd || !roots.some((root) => cwd === root || cwd.startsWith(root + sep))) continue;
    const command = info.command || processField(pid, "comm").split("/").at(-1) || "";
    if (NOT_OURS.has(command)) continue;
    const args = processField(pid, "args");
    try {
      const signal = terminate(pid);
      report.serversStopped.push({ pid, command, args, cwd, listening: info.listening, signal });
      log(`cleanup: stopped pid ${pid} (${args || command}) in ${cwd}${info.listening.length ? `, listening on ${info.listening.join(", ")}` : ""}${signal === "SIGKILL" ? ", after SIGTERM did not stop it" : ""}`);
      const file = pidFiles.get(pid);
      if (file) rmSync(file, { force: true });
    } catch (error: any) {
      report.serversLeft.push({ pid, cwd, error: error.message });
      log(`cleanup: could not stop pid ${pid} in ${cwd}: ${error.message}`);
    }
  }
}

function removeWorktrees(projectDir: string, worktrees: { path: string }[], report: CleanupReport, log: (line: string) => void): void {
  // Every linked worktree of the throwaway repository is the run's own; implement's workers put
  // theirs under <project>/.claude/worktrees/. `--force` twice removes one with uncommitted work or
  // a lock, which a worker that was stopped mid-task leaves. Branches stay; only the checkouts go.
  for (const { path } of worktrees) {
    if (!existsSync(path)) {
      report.worktreesPruned.push(path);
      continue;
    }
    try {
      git(projectDir, "worktree", "remove", "--force", "--force", path);
      report.worktreesRemoved.push(path);
      log(`cleanup: removed worktree ${path}`);
    } catch (error: any) {
      report.worktreesLeft.push({ path, error: error.message });
      log(`cleanup: could not remove worktree ${path}: ${error.message}`);
    }
  }
  try {
    git(projectDir, "worktree", "prune");
  } catch (error: any) {
    log(`cleanup: git worktree prune failed: ${error.message}`);
  }
}

/** The linked worktrees of the repository at `projectDir`, without the main one. */
function listWorktrees(projectDir: string): { path: string }[] {
  let out: string;
  try {
    out = git(projectDir, "worktree", "list", "--porcelain");
  } catch {
    return [];
  }
  const paths = out
    .split("\n")
    .filter((line) => line.startsWith("worktree "))
    .map((line) => line.slice("worktree ".length));
  return paths.slice(1).map((path) => ({ path }));
}

function git(cwd: string, ...args: string[]): string {
  try {
    return execFileSync("git", args, { cwd, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  } catch (error: any) {
    throw new Error(`git ${args.join(" ")}: ${(error.stderr ?? error.message).toString().trim()}`);
  }
}

/** Every process listening on a TCP port that this user can see, with its command name and addresses. */
function listeningProcesses(): Map<number, { command: string; listening: string[] }> {
  const found = new Map<number, { command: string; listening: string[] }>();
  let out = "";
  try {
    out = execFileSync("lsof", ["-nP", "-iTCP", "-sTCP:LISTEN", "-Fpcn"], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] });
  } catch (error: any) {
    out = error.stdout?.toString() ?? ""; // lsof exits 1 when nothing listens
  }
  let current: { command: string; listening: string[] } | undefined;
  for (const line of out.split("\n")) {
    const tag = line[0];
    const value = line.slice(1);
    if (tag === "p") {
      current = { command: "", listening: [] };
      found.set(Number(value), current);
    } else if (tag === "c" && current) {
      current.command = value;
    } else if (tag === "n" && current && !current.listening.includes(value)) {
      current.listening.push(value);
    }
  }
  return found;
}

/** A process's working directory, resolved, or undefined when it cannot be read. */
function processCwd(pid: number): string | undefined {
  try {
    return realpathSync(readlinkSync(`/proc/${pid}/cwd`));
  } catch {
    // not Linux, or the process is gone
  }
  try {
    const out = execFileSync("lsof", ["-a", "-p", String(pid), "-d", "cwd", "-Fn"], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] });
    const path = out.split("\n").find((line) => line.startsWith("n"))?.slice(1);
    return path && existsSync(path) ? realpathSync(path) : path;
  } catch {
    return undefined;
  }
}

function processField(pid: number, field: "comm" | "args"): string {
  try {
    return execFileSync("ps", ["-o", `${field}=`, "-p", String(pid)], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    return "";
  }
}

function alive(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch (error: any) {
    return error.code === "EPERM";
  }
}

/** SIGTERM, five seconds to exit, then SIGKILL. Synchronous, so it also runs from a signal handler. */
function terminate(pid: number): "SIGTERM" | "SIGKILL" {
  process.kill(pid, "SIGTERM");
  if (waitForExit(pid, 5_000)) return "SIGTERM";
  process.kill(pid, "SIGKILL");
  if (waitForExit(pid, 2_000)) return "SIGKILL";
  throw new Error("still running after SIGKILL");
}

function waitForExit(pid: number, ms: number): boolean {
  const until = Date.now() + ms;
  const tick = new Int32Array(new SharedArrayBuffer(4));
  while (Date.now() < until) {
    if (!alive(pid)) return true;
    Atomics.wait(tick, 0, 0, 100);
  }
  return !alive(pid);
}
