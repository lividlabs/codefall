// The judge: a read-only agent in another harness, started through the project's own
// .codefall/shared/run-agent.sh, the way codefall's verbs start a reviewer or a consult. The
// driver writes the prompt (judge/judge-prompt.md plus the schema and the run directory), tries
// `muse` first and `opencode` as the fallback, reads the JSON answer, writes verdict.md from it,
// and records which harness answered. The judge never goes through the Claude SDK.

import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, copyFileSync, existsSync } from "node:fs";
import { join } from "node:path";

export type JudgeRecord = {
  verdict: any | null;
  judgedBy: string | null;
  tried: { agent: string; exit: number; note: string }[];
  rawTail: string;
};

// Exit codes of run-agent.sh, as running-agents.md reads them.
const ANSWERED = 0;
const SKIP = new Set([69, 64]); // not runnable here: not on PATH, or unknown
const ADVANCE = new Set([73, 75, 76]); // ran and failed: wrote nothing, timed out, exited non-zero

export async function judgeRun(opts: {
  runDir: string;
  judgeDir: string;
  briefPath: string;
  projectDir: string;
  agents: string[];
  log: (line: string) => void;
}): Promise<JudgeRecord> {
  copyFileSync(join(opts.judgeDir, "rubric.md"), join(opts.runDir, "rubric.md"));
  copyFileSync(opts.briefPath, join(opts.runDir, "brief.md"));

  const script = join(opts.projectDir, ".codefall", "shared", "run-agent.sh");
  if (!existsSync(script)) throw new Error(`no run-agent.sh at ${script}; was codefall init run on the project?`);

  const schemaPath = join(opts.judgeDir, "verdict.schema.json");
  const schema = readFileSync(schemaPath, "utf8");
  const instructions = readFileSync(join(opts.judgeDir, "judge-prompt.md"), "utf8");
  const promptPath = join(opts.runDir, "judge-prompt.rendered.md");
  writeFileSync(
    promptPath,
    `${instructions}\n\n## The run directory\n\nYou are running inside it: \`${opts.runDir}\`. Read \`rubric.md\` first, then every transcript in chain order, then \`evidence/\` and \`deviations.txt\`.\n\n## The shape of your answer\n\nReply with one JSON object and nothing else: an instance of this schema, with \`overall\`, \`overall_reason\`, and \`blocks\` at the top level, not a copy of the schema itself:\n\n\`\`\`json\n${schema}\n\`\`\`\n`,
  );

  const tried: JudgeRecord["tried"] = [];
  let answer: string | null = null;
  let judgedBy: string | null = null;

  for (const agent of opts.agents) {
    const outPath = join(opts.runDir, `judge.${agent.replace(/[^a-z0-9]+/gi, "-")}.out`);
    opts.log(`judge: trying ${agent} through run-agent.sh`);
    const exit = runAgent(script, agent, promptPath, schemaPath, outPath, opts.runDir, opts.log);
    if (exit === ANSWERED) {
      answer = readFileSync(outPath, "utf8");
      judgedBy = agent;
      tried.push({ agent, exit, note: "answered" });
      break;
    }
    const note = SKIP.has(exit) ? "not runnable here; skipped" : ADVANCE.has(exit) ? "ran and failed; advancing" : exit === 70 ? "is `current`; the driver has no subagent to run, skipped" : `unexpected exit ${exit}`;
    tried.push({ agent, exit, note });
    opts.log(`judge: ${agent} exited ${exit}: ${note}`);
  }

  const verdict = answer ? parseJson(answer) : null;
  if (answer && !verdict) opts.log("judge: the answer did not parse as JSON; verdict.json carries the raw tail");

  writeFileSync(join(opts.runDir, "verdict.md"), renderVerdict(verdict, judgedBy, tried));
  const record: JudgeRecord = { verdict, judgedBy, tried, rawTail: (answer ?? "").slice(-4000) };
  writeFileSync(join(opts.runDir, "verdict.json"), JSON.stringify(record, null, 2) + "\n");
  opts.log(`judge: ${verdict?.overall ?? "no verdict"} (${judgedBy ?? "no agent answered"})`);
  return record;
}

function runAgent(script: string, agent: string, prompt: string, schema: string, out: string, cwd: string, log: (line: string) => void): number {
  try {
    execFileSync("bash", [script, agent, prompt, schema, out], {
      cwd,
      env: { ...process.env, CODEFALL_REVIEW_TIMEOUT: process.env.CODEFALL_REVIEW_TIMEOUT ?? "1800" },
      stdio: ["ignore", "ignore", "pipe"],
      encoding: "utf8",
    });
    return 0;
  } catch (error: any) {
    if (error.stderr) log(`[run-agent ${agent}] ${String(error.stderr).trim().slice(-600)}`);
    return typeof error.status === "number" ? error.status : 76;
  }
}

export function parseJson(text: string): any {
  // Muse has been seen to print its final object twice; take the last complete object.
  const fenced = [...text.matchAll(/```(?:json)?\s*([\s\S]*?)```/g)].map((m) => m[1]);
  const candidates = fenced.length ? fenced.reverse() : [text];
  for (const candidate of candidates) {
    const start = candidate.indexOf("{");
    const end = candidate.lastIndexOf("}");
    if (start < 0 || end < 0) continue;
    try {
      return unwrap(JSON.parse(candidate.slice(start, end + 1)));
    } catch {
      // try the next candidate
    }
  }
  return null;
}

// A judge has been seen to answer with a copy of the schema whose `properties` hold the values
// (`properties.overall`, `properties.blocks`) instead of a plain object. Read that shape too.
function unwrap(parsed: any): any {
  if (parsed && typeof parsed === "object" && parsed.overall === undefined && typeof parsed.properties?.overall === "string") {
    return parsed.properties;
  }
  return parsed;
}

export function renderVerdict(v: any, judgedBy: string | null, tried: JudgeRecord["tried"]): string {
  const lines: string[] = ["# Verdict", ""];
  lines.push(`Judged by: ${judgedBy ?? "nobody answered"}. Tried: ${tried.map((t) => `${t.agent} (${t.note})`).join("; ")}.`, "");
  if (!v) {
    lines.push("No JSON verdict was parsed. See verdict.json for the raw answer.", "");
    return lines.join("\n");
  }
  lines.push(`**Overall: ${v.overall}**`, "", v.overall_reason ?? "", "");
  lines.push("| Block | Verdict | Judged on |", "| --- | --- | --- |");
  for (const b of v.blocks ?? []) lines.push(`| ${b.block}. ${b.title ?? ""} | ${b.verdict} | ${(b.judged_on ?? []).join(", ")} |`);
  lines.push("");
  for (const b of v.blocks ?? []) {
    lines.push(`## ${b.block}. ${b.title ?? ""}`, "", `Verdict: **${b.verdict}**`, "");
    for (const q of b.quotes ?? []) lines.push(`> ${String(q).replace(/\n/g, "\n> ")}`, "");
    lines.push(b.reason ?? "", "");
  }
  return lines.join("\n");
}
