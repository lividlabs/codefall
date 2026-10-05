// Runs the chain: envision, specify, design, implement, review, test, each a fresh session in the
// project directory, with the PM simulator answering; lands the document stack and merges the code
// stack as the person would; gathers evidence; runs the judge.
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

const model = args.model ?? "opus";
const simModel = args["sim-model"] ?? "sonnet";
const judgeModel = args["judge-model"] ?? "opus";
const effort = (args.effort as any) ?? undefined;
const settleDesign = args["settle-design"] !== "false";
const maxPmTurns = Number(args["max-pm-turns"] ?? 60);
const maxMinutes = Number(args["max-minutes"] ?? 90);
const brief = readFileSync(join(evalRoot, "brief.md"), "utf8");
const gh = makeGh(repo, runDir, log);
const summary: Record<string, unknown> = { repo, projectDir, model, simModel, startedAt: new Date().toISOString(), sessions: [] as unknown[] };

async function runVerb(verb: Verb, personaOverride?: "engineer"): Promise<SessionOutcome> {
  const s = step(verb);
  const firstMessage = `${s.skill} ${s.argument(projectDir)}`.trim();
  const persona = personaOverride ?? "product-manager";
  setPersona(persona);
  log(`--- ${verb}: ${firstMessage.slice(0, 80)}${firstMessage.length > 80 ? "…" : ""} (persona ${persona})`);
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
  if (personaOverride) setPersona("product-manager");
  return outcome;
}

async function main(): Promise<void> {
  if (!args["judge-only"]) {
    const from = (args.from as Verb | undefined) ?? "envision";
    const verbs = CHAIN.slice(CHAIN.indexOf(from));
    if (!verbs.length) fail(`--from must be one of ${CHAIN.join(", ")}`);

    for (const verb of verbs) {
      const outcome = await runVerb(verb);
      if (outcome.status !== "done") {
        log(`stopping the chain: ${verb} ended ${outcome.status}`);
        break;
      }
      if (verb === "design") await afterDesign(outcome);
      if (verb === "test") afterTest();
    }
  }

  gh.gatherEvidence(projectDir, runDir, epicId());
  if (!existsSync(join(runDir, "deviations.txt"))) writeFileSync(join(runDir, "deviations.txt"), "(none)\n");
  summary.finishedAt = new Date().toISOString();
  writeFileSync(join(runDir, "summary.json"), JSON.stringify(summary, null, 2) + "\n");

  const record = await judgeRun({ runDir, judgeDir: join(evalRoot, "judge"), briefPath: join(evalRoot, "brief.md"), model: judgeModel, log });
  log(`run directory: ${runDir}`);
  log(`overall: ${record.verdict?.overall ?? "see verdict.json"}`);
}

// Between design and implement: settle a Draft design if allowed, then wait for the Action.
async function afterDesign(outcome: SessionOutcome): Promise<void> {
  const design = latestDoc("designs");
  const isDraft = design ? /\*\*Status:\*\*\s*Draft/.test(readFileSync(design, "utf8")) : false;
  const parked = design ? /## Decisions needed/.test(readFileSync(design, "utf8")) : false;
  if (isDraft && parked) {
    if (!settleDesign) {
      log("the design is Draft with decisions parked; --settle-design=false, so the chain stops here");
      throw new StopChain();
    }
    deviation("the design ended Draft with decisions parked under the product-manager persona; the driver ran one extra design session with the engineer persona to settle them (design-settle.md)");
    const settled = await runVerb("design-settle", "engineer");
    if (settled.status !== "done") throw new StopChain();
  }

  const pr = gh.designPr() ?? gh.listPrs("all").find((p) => p.headRefName.startsWith("design/"));
  if (!pr) {
    deviation("no design pull request was found after the design session; implement will run against whatever is on main");
    return;
  }
  if (pr.state === "MERGED") {
    log(`design PR #${pr.number} already merged`);
    return;
  }
  log(`waiting for the Action to land the document stack at design PR #${pr.number}`);
  const landed = await gh.waitForStackLanding(pr.number, Number(args["land-minutes"] ?? 15));
  if (!landed) {
    deviation(`the Action did not merge design PR #${pr.number} within the wait; the driver merged the stack as the person would, from outside the project directory`);
    try {
      gh.mergeStack(pr.number);
    } catch (error: any) {
      deviation(`the driver's own stack merge failed too: ${error.message}`);
    }
  }
}

// After test: merge the code stack as the person would, so the delivery reaches main.
function afterTest(): void {
  const top = gh.codeStackTop();
  if (!top) {
    log("no open code stack to merge");
    return;
  }
  const open = gh.listPrs("open").filter((p) => p.headRefName.startsWith("feat/"));
  log(`merging the code stack: ${open.length} layer(s), top #${top.number} ${top.headRefName}`);
  try {
    gh.mergeStack(top.number);
    appendFileSync(join(runDir, "deviations.txt"), `(not a deviation) the driver merged the code stack at #${top.number} after test, as the person would, from outside the project directory\n`);
  } catch (error: any) {
    deviation(`merging the code stack at #${top.number} failed: ${error.message}`);
  }
}

class StopChain extends Error {}

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
  if (error instanceof StopChain) {
    log("chain stopped; gathering evidence and judging what exists");
    gh.gatherEvidence(projectDir, runDir, epicId());
    return judgeRun({ runDir, judgeDir: join(evalRoot, "judge"), briefPath: join(evalRoot, "brief.md"), model: judgeModel, log });
  }
  log(`driver error: ${error.stack ?? error.message}`);
  process.exit(1);
});
