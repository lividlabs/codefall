// One verb, one fresh session. The first message is the slash command. AskUserQuestion calls are
// answered through `canUseTool`; every other tool is allowed, so the run never waits on a person.
// When a turn ends, the simulator reads it and either replies, declares the session done, or says
// it is stuck. Streaming input keeps the session alive between turns.

import { query, type SDKUserMessage, type SDKMessage } from "@anthropic-ai/claude-agent-sdk";
import { Simulator, type Question } from "./simulator.ts";
import { Transcript } from "./transcript.ts";

export type SessionOptions = {
  projectDir: string;
  runDir: string;
  verb: string;
  firstMessage: string;
  simulator: Simulator;
  model: string;
  effort?: "low" | "medium" | "high" | "xhigh" | "max";
  maxPmTurns: number;
  maxMinutes: number;
  log: (line: string) => void;
};

export type SessionOutcome = {
  verb: string;
  status: "done" | "stuck" | "error" | "turn-limit" | "time-limit";
  pmTurns: number;
  costUsd: number;
  sessionId?: string;
  lastText: string;
  transcript: Transcript;
};

class MessageQueue implements AsyncIterable<SDKUserMessage> {
  private waiting: ((value: IteratorResult<SDKUserMessage>) => void) | null = null;
  private buffered: SDKUserMessage[] = [];
  private closed = false;

  push(text: string): void {
    const message: SDKUserMessage = {
      type: "user",
      message: { role: "user", content: text },
      parent_tool_use_id: null,
    };
    if (this.waiting) {
      const resolve = this.waiting;
      this.waiting = null;
      resolve({ value: message, done: false });
    } else {
      this.buffered.push(message);
    }
  }

  close(): void {
    this.closed = true;
    if (this.waiting) {
      const resolve = this.waiting;
      this.waiting = null;
      resolve({ value: undefined as any, done: true });
    }
  }

  [Symbol.asyncIterator](): AsyncIterator<SDKUserMessage> {
    return {
      next: () => {
        if (this.buffered.length) return Promise.resolve({ value: this.buffered.shift()!, done: false });
        if (this.closed) return Promise.resolve({ value: undefined as any, done: true });
        return new Promise((resolve) => (this.waiting = resolve));
      },
    };
  }
}

export async function runSession(opts: SessionOptions): Promise<SessionOutcome> {
  const transcript = new Transcript(opts.runDir, opts.verb, opts.firstMessage);
  const queue = new MessageQueue();
  const abort = new AbortController();
  const deadline = Date.now() + opts.maxMinutes * 60_000;
  let pmTurns = 0;
  let costUsd = 0;
  let sessionId: string | undefined;
  let status: SessionOutcome["status"] = "done";
  let settling = false;

  queue.push(opts.firstMessage);

  const stream = query({
    prompt: queue,
    options: {
      cwd: opts.projectDir,
      model: opts.model,
      ...(opts.effort ? { effort: opts.effort } : {}),
      systemPrompt: { type: "preset", preset: "claude_code" },
      settingSources: ["project", "local"],
      permissionMode: "acceptEdits",
      abortController: abort,
      canUseTool: async (toolName, input) => {
        if (toolName === "AskUserQuestion") {
          const questions = (input as any).questions as Question[];
          transcript.note("PM-sim", `asked through a form: ${questions.map((q) => q.question).join(" / ")}`);
          const answers = await opts.simulator.answerQuestions(questions, transcript.tail());
          transcript.note("PM", Object.entries(answers).map(([q, a]) => `${q} → ${a}`).join("\n"));
          pmTurns += 1;
          return { behavior: "allow", updatedInput: { ...input, answers } };
        }
        // Everything else is allowed. The guard hook still runs first and denies a merge to main.
        return { behavior: "allow", updatedInput: input };
      },
      stderr: (data) => opts.log(`[stderr ${opts.verb}] ${data.trimEnd()}`),
    },
  });

  try {
    for await (const message of stream as AsyncIterable<SDKMessage>) {
      transcript.record(message);
      if (message.type === "system" && (message as any).subtype === "init") sessionId = (message as any).session_id;
      if (message.type !== "result") continue;

      const result = message as any;
      costUsd = Number(result.total_cost_usd ?? costUsd);
      if (result.subtype !== "success") {
        transcript.note("driver", `the session ended with ${result.subtype}; stopping this verb`);
        status = "error";
        queue.close();
        break;
      }
      if (Date.now() > deadline) {
        transcript.note("driver", `time limit of ${opts.maxMinutes} minutes reached; stopping this verb`);
        status = "time-limit";
        queue.close();
        break;
      }
      if (settling) continue; // a result that arrived while we were already deciding

      settling = true;
      const move = await opts.simulator.nextMove(transcript.turnText, transcript.tail());
      settling = false;
      if (move.kind === "done") {
        transcript.note("PM-sim", `the verb is finished — ${move.reason}`);
        queue.close();
        break;
      }
      if (move.kind === "stuck") {
        transcript.note("PM-sim", `stuck — ${move.reason}`);
        status = "stuck";
        queue.close();
        break;
      }
      pmTurns += 1;
      if (pmTurns > opts.maxPmTurns) {
        transcript.note("driver", `turn limit of ${opts.maxPmTurns} reached; stopping this verb`);
        status = "turn-limit";
        queue.close();
        break;
      }
      transcript.note("PM", move.text);
      transcript.newTurn();
      queue.push(move.text);
    }
  } finally {
    queue.close();
    abort.abort();
  }

  const outcome: SessionOutcome = {
    verb: opts.verb,
    status,
    pmTurns,
    costUsd,
    sessionId,
    lastText: transcript.turnText,
    transcript,
  };
  transcript.finish({ status, pmTurns, costUsd, sessionId });
  return outcome;
}
