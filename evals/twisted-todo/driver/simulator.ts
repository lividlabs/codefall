// The PM simulator: a model call given the brief and the transcript so far, answering in character.
// It has two jobs. `answerQuestions` answers an AskUserQuestion call; `nextMove` reads the end of a
// turn and decides whether the verb asked something in prose (reply), finished (done), or stopped
// on something the person cannot fix (stuck).
//
// It runs through the Agent SDK with no tools and no project settings, so it uses the same login
// as the sessions under test and needs no separate API key.

import { query } from "@anthropic-ai/claude-agent-sdk";

export type Question = {
  question: string;
  header: string;
  options: { label: string; description?: string }[];
  multiSelect: boolean;
};

export type Move =
  | { kind: "reply"; text: string; reason: string }
  | { kind: "done"; reason: string }
  | { kind: "stuck"; reason: string };

export type SimulatorOptions = {
  brief: string;
  verb: string;
  notes: string;
  model: string;
  scratchDir: string;
  log?: (line: string) => void;
};

const SYSTEM = `You play a person in a conversation with a coding agent. The person's brief follows; stay in character, in her own words, in product vocabulary. You never name files, libraries, databases, frameworks, or components. You never offer to do a step yourself and never ask anyone to merge anything. You answer in two to five sentences unless one word is the right answer. You add no features the brief does not contain. You decide nothing technical: when asked a technical question you say you cannot judge it and ask the agent to pick what it thinks is right and tell you in plain words.

You answer with JSON only, no prose around it, in the shape the task asks for.`;

export class Simulator {
  private readonly opts: SimulatorOptions;

  constructor(opts: SimulatorOptions) {
    this.opts = opts;
  }

  async answerQuestions(questions: Question[], transcriptTail: string): Promise<Record<string, string>> {
    const task = `The agent asked these questions through a form. For each, answer with the exact label of one option when one fits (or several labels joined with ", " when multiSelect is true), or with your own short answer when none fits. Reply as JSON: {"answers": {"<question text exactly as given>": "<answer>"}, "reason": "<one sentence>"}.

Questions:
${JSON.stringify(questions, null, 2)}`;
    const out = await this.ask(task, transcriptTail);
    const answers = out?.answers;
    if (!answers || typeof answers !== "object") throw new Error(`simulator returned no answers: ${JSON.stringify(out)}`);
    this.opts.log?.(`answered form: ${JSON.stringify(answers)} — ${out.reason ?? ""}`);
    // Every question must have an answer, keyed by its exact text.
    for (const q of questions) {
      if (!(q.question in answers)) answers[q.question] = q.options[0]?.label ?? "Yes";
    }
    return answers;
  }

  async nextMove(turnText: string, transcriptTail: string): Promise<Move> {
    const task = `The agent's turn just ended. Decide what you do. Reply as JSON in one of these shapes:
- {"kind": "reply", "text": "<what you say next>", "reason": "<one sentence>"} when the agent asked you something, is waiting for a confirmation or a go-ahead, or showed you a document to approve.
- {"kind": "done", "reason": "<one sentence>"} when the agent gave its final report: it says what happens next, names at most one command for you, or says it is safe to clear. Also "done" when the agent says it has stopped and the remedy is a command for you to run or an install to do.
- {"kind": "stuck", "reason": "<one sentence>"} when the agent is looping, asked the same thing a third time, or is asking for something the brief cannot answer and parking it is not offered.

Never reply "done" while the agent is clearly waiting for your answer. Never reply with a command or a merge.

The agent's last turn:
---
${turnText.trim() || "(no text; the agent ended its turn without saying anything)"}
---`;
    const out = await this.ask(task, transcriptTail);
    if (out?.kind === "reply" && typeof out.text === "string") {
      this.opts.log?.(`reply — ${out.reason ?? ""}`);
      return { kind: "reply", text: out.text, reason: String(out.reason ?? "") };
    }
    if (out?.kind === "done") return { kind: "done", reason: String(out.reason ?? "") };
    if (out?.kind === "stuck") return { kind: "stuck", reason: String(out.reason ?? "") };
    throw new Error(`simulator returned an unreadable move: ${JSON.stringify(out)}`);
  }

  private async ask(task: string, transcriptTail: string): Promise<any> {
    const prompt = `## The person you play\n\n${this.opts.brief}\n\n## This session\n\nThe agent is running \`${this.opts.verb}\`. ${this.opts.notes}\n\n## The conversation so far (most recent last)\n\n${transcriptTail}\n\n## Your task\n\n${task}`;
    let text = "";
    for await (const message of query({
      prompt,
      options: {
        cwd: this.opts.scratchDir,
        model: this.opts.model,
        systemPrompt: { type: "custom", prompt: SYSTEM },
        settingSources: [],
        tools: [],
        maxTurns: 1,
        persistSession: false,
        permissionMode: "dontAsk",
      },
    })) {
      if (message.type === "result" && (message as any).subtype === "success") text = (message as any).result ?? "";
      if (message.type === "result" && (message as any).subtype !== "success") {
        throw new Error(`simulator call failed: ${(message as any).subtype}`);
      }
    }
    return parseJson(text);
  }
}

function parseJson(text: string): any {
  const fenced = text.match(/```(?:json)?\s*([\s\S]*?)```/);
  const candidate = (fenced ? fenced[1] : text).trim();
  const start = candidate.indexOf("{");
  const end = candidate.lastIndexOf("}");
  if (start < 0 || end < 0) return null;
  try {
    return JSON.parse(candidate.slice(start, end + 1));
  } catch {
    return null;
  }
}
