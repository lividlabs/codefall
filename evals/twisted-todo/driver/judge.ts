// The judge: one session with read-only tools over the run directory, prompted from
// judge/judge-prompt.md, writing verdict.md and ending with a JSON block the driver saves as
// verdict.json.

import { query } from "@anthropic-ai/claude-agent-sdk";
import { readFileSync, writeFileSync, copyFileSync } from "node:fs";
import { join } from "node:path";

export async function judgeRun(opts: {
  runDir: string;
  judgeDir: string;
  briefPath: string;
  model: string;
  log: (line: string) => void;
}): Promise<any> {
  copyFileSync(join(opts.judgeDir, "rubric.md"), join(opts.runDir, "rubric.md"));
  copyFileSync(opts.briefPath, join(opts.runDir, "brief.md"));
  const system = readFileSync(join(opts.judgeDir, "judge-prompt.md"), "utf8");

  let text = "";
  let cost = 0;
  for await (const message of query({
    prompt: `Judge the run in ${opts.runDir}. Read rubric.md first, then every transcript in chain order, then evidence/ and deviations.txt. Write verdict.md there, then end with the JSON block.`,
    options: {
      cwd: opts.runDir,
      model: opts.model,
      systemPrompt: { type: "custom", prompt: system },
      settingSources: [],
      tools: ["Read", "Glob", "Grep", "Write"],
      permissionMode: "acceptEdits",
      canUseTool: async (toolName, input) => {
        // Writes are confined to verdict.md in the run directory; everything else is read-only.
        if (toolName === "Write" && String((input as any).file_path ?? "").endsWith("/verdict.md")) {
          return { behavior: "allow", updatedInput: input };
        }
        if (toolName === "Write") return { behavior: "deny", message: "the judge writes only verdict.md" };
        return { behavior: "allow", updatedInput: input };
      },
      maxTurns: 200,
      persistSession: false,
    },
  })) {
    if (message.type === "result") {
      const r = message as any;
      cost = Number(r.total_cost_usd ?? 0);
      if (r.subtype === "success") text = r.result ?? "";
      else opts.log(`judge ended with ${r.subtype}`);
    }
  }

  const fenced = text.match(/```json\s*([\s\S]*?)```\s*$/);
  let verdict: any = null;
  if (fenced) {
    try {
      verdict = JSON.parse(fenced[1]);
    } catch {
      verdict = null;
    }
  }
  const record = { verdict, judgeCostUsd: cost, rawTail: text.slice(-4000) };
  writeFileSync(join(opts.runDir, "verdict.json"), JSON.stringify(record, null, 2) + "\n");
  opts.log(`judge: ${verdict?.overall ?? "no JSON verdict parsed"} (cost $${cost.toFixed(2)})`);
  return record;
}
