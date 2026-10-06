// Writes one session's transcript twice: every SDK message as a JSONL line, and a readable
// Markdown rendering of the conversation. The judge reads the Markdown and quotes from either.

import { appendFileSync, writeFileSync } from "node:fs";
import type { SDKMessage } from "@anthropic-ai/claude-agent-sdk";

const CLIP = 600;

export class Transcript {
  readonly jsonlPath: string;
  readonly mdPath: string;
  private md: string[] = [];
  private lastAssistantText = "";

  constructor(dir: string, verb: string, firstMessage: string) {
    this.jsonlPath = `${dir}/${verb}.jsonl`;
    this.mdPath = `${dir}/${verb}.md`;
    writeFileSync(this.jsonlPath, "");
    this.md.push(`# ${verb}\n`, `Started ${new Date().toISOString()}.\n`);
    this.note("PM", firstMessage);
  }

  /** The assistant's text since the last user turn; what the simulator reads to decide its move. */
  get turnText(): string {
    return this.lastAssistantText;
  }

  /** The readable transcript so far, clipped from the front, for the simulator's context. */
  tail(chars = 14000): string {
    const all = this.md.join("\n");
    return all.length > chars ? `…\n${all.slice(-chars)}` : all;
  }

  record(message: SDKMessage): void {
    appendFileSync(this.jsonlPath, JSON.stringify(message) + "\n");
    switch (message.type) {
      case "assistant": {
        if (message.parent_tool_use_id) return; // a subagent's frames; kept in the JSONL only
        for (const block of message.message.content as any[]) {
          if (block.type === "text" && block.text?.trim()) {
            this.lastAssistantText += (this.lastAssistantText ? "\n\n" : "") + block.text;
            this.push(`**Verb:** ${block.text.trim()}`);
          } else if (block.type === "tool_use") {
            this.push(`> tool \`${block.name}\` — ${describeInput(block.name, block.input)}`);
          }
        }
        break;
      }
      case "user": {
        if (message.parent_tool_use_id) return;
        const content = message.message.content;
        if (typeof content === "string") {
          this.push(`**PM:** ${content}`);
        } else {
          for (const block of content as any[]) {
            if (block.type === "text") this.push(`**PM:** ${block.text}`);
            else if (block.type === "tool_result") {
              const text = typeof block.content === "string" ? block.content : flatten(block.content);
              this.push(`> result ${block.is_error ? "(error) " : ""}— ${clip(text)}`);
            }
          }
        }
        break;
      }
      case "system": {
        if ((message as any).subtype === "init") {
          const m = message as any;
          this.push(`> session ${m.session_id} · model ${m.model} · ${m.tools?.length ?? "?"} tools · permission mode ${m.permissionMode}`);
        }
        break;
      }
      case "result": {
        const r = message as any;
        this.push(
          `> turn ended · ${r.subtype} · turns ${r.num_turns} · cost so far $${Number(r.total_cost_usd ?? 0).toFixed(2)}`,
        );
        break;
      }
      default:
        break;
    }
  }

  /** Lines from outside the session: the simulator's decisions and the driver's actions. */
  note(who: "PM" | "PM-sim" | "driver", text: string): void {
    this.push(`**${who}:** ${text.trim()}`);
    appendFileSync(this.jsonlPath, JSON.stringify({ type: "eval_note", who, text, at: new Date().toISOString() }) + "\n");
  }

  /** Called when the simulator has answered a turn, so the next turn's text starts clean. */
  newTurn(): void {
    this.lastAssistantText = "";
  }

  finish(summary: Record<string, unknown>): void {
    this.push(`\n---\nFinished ${new Date().toISOString()}.\n\n\`\`\`json\n${JSON.stringify(summary, null, 2)}\n\`\`\``);
  }

  private push(line: string): void {
    this.md.push(line + "\n");
    writeFileSync(this.mdPath, this.md.join("\n"));
  }
}

function describeInput(tool: string, input: any): string {
  if (!input) return "";
  switch (tool) {
    case "Bash":
      return `\`${clip(String(input.command ?? ""), 300)}\``;
    case "Read":
    case "Write":
    case "Edit":
      return String(input.file_path ?? "");
    case "Glob":
    case "Grep":
      return `${input.pattern ?? ""} ${input.path ?? ""}`.trim();
    case "Skill":
      return `${input.skill ?? ""} ${input.args ?? ""}`.trim();
    case "Agent":
      return `${input.description ?? ""}${input.isolation ? ` (${input.isolation})` : ""}`;
    case "AskUserQuestion":
      return (input.questions ?? [])
        .map((q: any) => `${q.header}: ${q.question} [${(q.options ?? []).map((o: any) => o.label).join(" | ")}]`)
        .join("; ");
    default:
      return clip(JSON.stringify(input), 300);
  }
}

function flatten(blocks: any[]): string {
  return (blocks ?? []).map((b) => (b.type === "text" ? b.text : `[${b.type}]`)).join("\n");
}

function clip(text: string, max = CLIP): string {
  const one = text.replace(/\s+/g, " ").trim();
  return one.length > max ? `${one.slice(0, max)}… (${one.length} chars)` : one;
}
