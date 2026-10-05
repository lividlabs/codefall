// Runs the chain: equip (local, then test), envision, specify, design, implement, review, test,
// each a fresh session in the project directory, with the PM simulator answering. Between
// sessions it does what the person would: merges the equip pull requests, adds the `auto-merge` label to
// each document pull request and waits for the Action to merge it, returns the checkout to main,
// and merges the code stack after test. Then it gathers evidence and runs the judge through
// .codefall/shared/run-agent.sh.
//
//   node run.ts                      the whole chain, then the judge
//   node run.ts --from implement     resume a run from a verb (pass --run <dir> to reuse a run dir)
//   node run.ts --judge-only --run runs/<timestamp>
//
// Reads evals/twisted-todo/.throwaway.env, written by fixture/setup.sh.

import { readFileSync, mkdirSync, appendFileSync, existsSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { CHAIN, step, type Verb } from "./protocol.ts";
import { Simulator } from "./simulator.ts";
import { runSession, type SessionOutcome } from "./session.ts";
import { makeGh } from "./github.ts";
import { judgeRun } from "./judge.ts";

const here = dirname(fileURLToPath(import.meta.url));
const evalRoot = resolve(here, "..");

const args = parseArgs(process.argv.slice(2));
const env = readEnvFile(join(evalRoot, ".throwaway.env"));
const projectDir = args.project ?? env.PROJECT_DIR;
const repo = args.repo ?? env.REPO;
if (!projectDir || !repo) fail("no project: run fixture/setup.sh first, or pass --project <dir> --repo <owner/name>");

const runDir = resolve(args.run ?? join(evalRoot, "runs", new Date().toISOString().replace(/[:.]/g, "").slice(0, 15)));
mkdirSync(runDir, { recursive: true });
const logPath = join(runDir, "driver.log");
const log = (line: string) => {
  const stamped = `${new Date().toISOString()} ${line}`;
  console.log(stamped);
  appendFileSync(logPath, stamped + "\n");
};
const deviation = (text: string) => {
  log(`DEVIATION: ${text}`);
  appendFileSync(join(runDir, "deviations.txt"), text + "\n");
};
const asThePerson = (text: string) => {
  log(`as the person: ${text}`);
  appendFileSync(join(runDir, "deviations.txt"), `(not a deviation) ${text}\n`);
};

const model = args.model ?? "opus";
const simModel = args["sim-model"] ?? "sonnet";
const judgeAgents = (args["judge-agents"] ?? "muse,opencode").split(",").map((s) => s.trim()).filter(Boolean);
const effort = (args.effort as any) ?? undefined;
const maxPmTurns = Number(args["max-pm-turns"] ?? 60);
const maxMinutes = Number(args["max-minutes"] ?? 90);
const landMinutes = Number(args["land-minutes"] ?? 10);
const brief = readFileSync(join(evalRoot, "brief.md"), "utf8");
const gh = makeGh(repo, runDir, log);
const summary: Record<string, unknown> = { repo, projectDir, model, simModel, judgeAgents, startedAt: new Date().toISOString(), sessions: [] as unknown[] };

async function runVerb(verb: Verb): Promise<SessionOutcome> {
  const s = step(verb);
  const firstMessage = `${s.skill} ${s.argument(projectDir)}`.trim();
  log(`--- ${verb}: ${firstMessage.slice(0, 80)}${firstMessage.length > 80 ? "…" : ""}`);
  const simulator = new Simulator({
    brief,
    verb,
    notes: s.simulatorNotes,
    model: simModel,
    scratchDir: runDir,
    log: (line) => log(`[PM-sim ${verb}] ${line}`),
  });
  const outcome = await runSession({ projectDir, runDir, verb, firstMessage, simulator, model, effort, maxPmTurns, maxMinutes, log });
  log(`${verb} ended: ${outcome.status}, ${outcome.pmTurns} PM turns, $${outcome.costUsd.toFixed(2)}`);
  (summary.sessions as unknown[]).push({ verb, status: outcome.status, pmTurns: outcome.pmTurns, costUsd: outcome.costUsd, sessionId: outcome.sessionId });
  writeFileSync(join(runDir, "summary.json"), JSON.stringify(summary, null, 2) + "\n");
  return outcome;
}

async function main(): Promise<void> {
  if (!args["judge-only"]) {
    const from = (args.from as Verb | undefined) ?? CHAIN[0];
    const verbs = CHAIN.slice(CHAIN.indexOf(from));
    if (!verbs.length) fail(`--from must be one of ${CHAIN.join(", ")}`);

    setPersona("product-manager");
    for (const verb of verbs) {
      const outcome = await runVerb(verb);
      if (outcome.status !== "done") {
        log(`stopping the chain: ${verb} ended ${outcome.status}`);
        break;
      }
      const goOn = await afterVerb(verb);
      if (!goOn) break;
    }
  }

  gh.gatherEvidence(projectDir, runDir, epicId());
  if (!existsSync(join(runDir, "deviations.txt"))) writeFileSync(join(runDir, "deviations.txt"), "(none)\n");
  summary.finishedAt = new Date().toISOString();
  writeFileSync(join(runDir, "summary.json"), JSON.stringify(summary, null, 2) + "\n");

  const record = await judgeRun({ runDir, judgeDir: join(evalRoot, "judge"), briefPath: join(evalRoot, "brief.md"), projectDir, agents: judgeAgents, log });
  summary.judgedBy = record.judgedBy;
  writeFileSync(join(runDir, "summary.json"), JSON.stringify(summary, null, 2) + "\n");
  log(`run directory: ${runDir}`);
  log(`overall: ${record.verdict?.overall ?? "see verdict.json"} (judged by ${record.judgedBy ?? "nobody"})`);
}

// What the person would do once a verb's session is over. Returns false when the chain should stop.
async function afterVerb(verb: Verb): Promise<boolean> {
  const after = step(verb).after;
  switch (after.kind) {
    case "merge-own-pr": {
      const pr = gh.latestPr(after.prefix);
      if (!pr) {
        deviation(`${verb} opened no pull request on ${after.prefix}*; the next verb runs against main as it is`);
        return true;
      }
      if (pr.state !== "MERGED") {
        gh.mergePr(pr.number);
        asThePerson(`the driver merged ${verb}'s pull request #${pr.number} (${pr.headRefName}), which is a person's to merge`);
      }
      gh.returnToMain(projectDir);
      return true;
    }
    case "label-and-wait": {
      if (verb === "design") {
        const design = latestDoc("designs");
        const text = design ? readFileSync(design, "utf8") : "";
        if (/\*\*Status:\*\*\s*Draft/.test(text) && /## Decisions needed/.test(text)) {
          log("the design is Draft with decisions left for an engineer; the chain stops here, because no engineer is in this run");
          return false;
        }
      }
      const pr = gh.latestPr(after.prefix);
      if (!pr) {
        deviation(`${verb} opened no pull request on ${after.prefix}*; the next verb runs against main as it is`);
        return true;
      }
      if (pr.state === "MERGED") {
        deviation(`${verb}'s pull request #${pr.number} was already merged before the person added the auto-merge label; something other than the person merged it`);
      } else {
        if (pr.labels.some((l) => l.name === "auto-merge")) {
          deviation(`${verb}'s pull request #${pr.number} already carried the auto-merge label when the session ended; the verb or something in it added it, which no verb may do`);
        } else {
          gh.addAutoMergeLabel(pr.number);
          asThePerson(`the driver added the auto-merge label to ${verb}'s pull request #${pr.number}, as the person would after reading the report, from outside the project directory`);
        }
        log(`waiting up to ${landMinutes} minutes for the Action to merge ${verb}'s pull request #${pr.number}`);
        const merged = await gh.waitForMerge(pr.number, landMinutes);
        if (!merged) {
          deviation(`the Action did not merge ${verb}'s pull request #${pr.number} within ${landMinutes} minutes; the driver merged it as the person would, from outside the project directory`);
          try {
            gh.mergePr(pr.number);
          } catch (error: any) {
            deviation(`the driver's own merge of #${pr.number} failed too: ${error.message}`);
          }
        }
      }
      gh.returnToMain(projectDir);
      asThePerson(`the driver returned the checkout to main after ${verb}`);
      return true;
    }
    case "merge-code-stack": {
      const top = gh.codeStackTop();
      if (!top) {
        log("no open code stack to merge");
        return true;
      }
      const open = gh.listPrs("open").filter((p) => p.headRefName.startsWith("feat/"));
      log(`merging the code stack: ${open.length} layer(s), top #${top.number} ${top.headRefName}`);
      try {
        gh.mergeStack(top.number);
        asThePerson(`the driver merged the code stack at #${top.number} after test, which is a person's to merge, from outside the project directory`);
      } catch (error: any) {
        deviation(`merging the code stack at #${top.number} failed: ${error.message}`);
      }
      gh.returnToMain(projectDir);
      return true;
    }
    case "none":
      return true;
  }
}

function setPersona(persona: "engineer" | "product-manager"): void {
  const bin = env.CODEFALL_BIN ?? "codefall";
  execFileSync(bin, ["config", "persona", persona], { cwd: projectDir, stdio: "ignore" });
}

function latestDoc(kind: "designs" | "specs" | "visions"): string | undefined {
  const dir = join(projectDir, "docs", kind);
  if (!existsSync(dir)) return undefined;
  const files = execFileSync("ls", [dir], { encoding: "utf8" }).split("\n").filter((f) => /^(DESIGN|SPEC|VISION)-\d{3}.*\.md$/.test(f)).sort();
  return files.length ? join(dir, files.at(-1)!) : undefined;
}

function epicId(): string {
  try {
    const prefix = execFileSync("bd", ["config", "get", "issue_prefix"], { cwd: projectDir, encoding: "utf8" }).trim();
    const design = latestDoc("designs")?.match(/(DESIGN-\d{3})/)?.[1] ?? "DESIGN-001";
    return `${prefix}-${design}`;
  } catch {
    return "forfeit-DESIGN-001";
  }
}

function parseArgs(argv: string[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (!a.startsWith("--")) continue;
    const [key, inline] = a.slice(2).split("=");
    if (inline !== undefined) out[key] = inline;
    else if (argv[i + 1] && !argv[i + 1].startsWith("--")) out[key] = argv[++i];
    else out[key] = "true";
  }
  return out;
}

function readEnvFile(path: string): Record<string, string> {
  if (!existsSync(path)) return {};
  return Object.fromEntries(
    readFileSync(path, "utf8")
      .split("\n")
      .filter((l) => l && !l.startsWith("#"))
      .map((l) => l.split("=") as [string, string])
      .map(([k, ...v]) => [k, v.join("=")]),
  );
}

function fail(message: string): never {
  console.error(message);
  process.exit(2);
}

main().catch((error) => {
  log(`driver error: ${error.stack ?? error.message}`);
  process.exit(1);
});
