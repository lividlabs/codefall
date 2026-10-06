// The run protocol: one entry per session, in chain order. Each says what the first message is,
// what the simulator must and must not do in that session beyond the brief, and how the driver
// tells that the session is done. PLAN.md restates this table in prose.

import { readdirSync, existsSync } from "node:fs";
import { join } from "node:path";

export type Verb = "equip-local" | "equip-test" | "envision" | "specify" | "design" | "implement" | "review" | "test";

export type Step = {
  verb: Verb;
  /** The skill the session starts with. */
  skill: string;
  /** Builds the argument that follows the slash command, from the project tree as it stands. */
  argument: (projectDir: string) => string;
  /** Guidance for the simulator in this session, added to the brief. */
  simulatorNotes: string;
  /**
   * What the driver does once the session is done, in the person's place:
   * - `merge-own-pr`: the verb opened a pull request a person merges (equip); the driver merges it.
   * - `label-and-wait`: the verb opened a document pull request; the driver adds the `auto-merge` label
   *   as the person would, waits for the Action to merge it, and merges it itself only as a
   *   recorded fallback. `prefix` is the branch prefix.
   * - `none`: nothing to land (implement, review; the code stack stays open).
   * - `merge-code-stack`: the delivery is over; the driver merges the code stack top.
   */
  after: { kind: "merge-own-pr"; prefix: string } | { kind: "label-and-wait"; prefix: string } | { kind: "none" } | { kind: "merge-code-stack" };
};

// The idea, in the person's words, as the first message of the chain. The brief has the rest.
export const IDEA = `I want to build a todo list with a twist. It's called Forfeit. No task can be added alone: you add tasks two at a time, and the two are rivals. When you finish one, the other is forfeited right then — it leaves the list, it is never marked done, and it goes to a second page that keeps the record of everything I gave up, what beat it, and when. The main page shows only the open pairs. The number of tasks I've finished is always exactly the number I've forfeited. It's a small web page for me alone on my laptop: no accounts, no sign-in, nothing that talks to another service.`;

function findId(projectDir: string, dir: string, prefix: string, fallback: string): string {
  const path = join(projectDir, "docs", dir);
  if (!existsSync(path)) return fallback;
  const ids = readdirSync(path)
    .map((f) => f.match(new RegExp(`^(${prefix}-\\d{3})`))?.[1])
    .filter((x): x is string => Boolean(x))
    .sort();
  return ids.at(-1) ?? fallback;
}

export const STEPS: Step[] = [
  {
    verb: "equip-local",
    skill: "/codefall-equip",
    argument: () => "local",
    simulatorNotes: `This is engineering work: when asked whether to continue, say yes. The project already has scripts/local.sh with start and update. When the agent shows what it found and asks whether to declare it or draft new scripts, say declare what exists. When it offers to prove the scripts by running them, say yes. Never ask to merge anything; the session is over when the report ends.`,
    after: { kind: "merge-own-pr", prefix: "equip/" },
  },
  {
    verb: "equip-test",
    skill: "/codefall-equip",
    argument: () => "test",
    simulatorNotes: `This is engineering work: when asked whether to continue, say yes. The project already has a Playwright configuration pointed at testing/test-cases with no specs yet. When the agent shows what it found and asks whether to declare it or set up the default runner, say declare what exists. Never ask to merge anything; the session is over when the report ends.`,
    after: { kind: "merge-own-pr", prefix: "equip/" },
  },
  {
    verb: "envision",
    skill: "/codefall-envision",
    argument: () => IDEA,
    simulatorNotes: `This is the vision interview. Give the idea as you would say it out loud; answer why now, who feels the problem, what this does not cover. Keep the rules at the level of rules, not exact behaviour: when the agent asks for precise behaviour, say that is for the spec. Confirm the document when it carries the five rules and the out-of-scope list. It is Ready, not Draft. Never ask to merge anything; the session is over when the report ends.`,
    after: { kind: "label-and-wait", prefix: "vision/" },
  },
  {
    verb: "specify",
    skill: "/codefall-specify",
    argument: (dir) => findId(dir, "visions", "VISION", "VISION-001"),
    simulatorNotes: `This is the spec interview. Yes, work from the vision. Aim for three requirements: adding a pair, finishing one task and forfeiting its rival, and the Forfeits page. You have no mockups; ask the agent to make them, and you want the main page full and empty and the Forfeits page full and empty, no loading or error drawings. Answer failure questions from the brief. Confirm the recap and the document when they carry the five rules. Ready, not Draft. Never ask to merge anything.`,
    after: { kind: "label-and-wait", prefix: "spec/" },
  },
  {
    verb: "design",
    skill: "/codefall-design",
    argument: (dir) => findId(dir, "specs", "SPEC", "SPEC-001"),
    simulatorNotes: `This is the design. You can judge the tier (accept the agent's call; tier 1 is expected), the task cut (five tasks or fewer; ask for a smaller cut if more), and the test criteria in product terms (they must cover the five rules). You cannot judge anything technical: say so plainly and let the agent set it aside. When the agent asks whether to settle the technical decisions it set aside with sensible defaults now or leave them for an engineer, say settle them now so building can start. Confirm the document when the plan covers the three requirements.`,
    after: { kind: "label-and-wait", prefix: "design/" },
  },
  {
    verb: "implement",
    skill: "/codefall-implement",
    argument: (dir) => findId(dir, "designs", "DESIGN", "DESIGN-001"),
    simulatorNotes: `When told this is engineering work and asked whether to continue, say yes. At the go gate say go; accept the serial stack and the models proposed. The driver answers permission prompts, so the permissions condition is met. If a worker fails and the agent asks what to do, say retry once, then skip the task and tell you. Never ask to merge; the report's merge order is for later.`,
    after: { kind: "none" },
  },
  {
    verb: "review",
    skill: "/codefall-review",
    argument: (dir) => findId(dir, "designs", "DESIGN", "DESIGN-001"),
    simulatorNotes: `Say yes to engineering work. Review now with every lens. When the agent asks "I found N problems. Fix them all?", say yes, fix them all. It then runs implement itself in this same session: at implement's go gate say go, and answer as you would in the implement session. Give a one-sentence reason for anything you dismiss. Never ask to merge.`,
    after: { kind: "none" },
  },
  {
    verb: "test",
    skill: "/codefall-test",
    argument: (dir) => findId(dir, "designs", "DESIGN", "DESIGN-001"),
    simulatorNotes: `Run now. Accept the driver the agent proposes. When the agent asks "I found N problems. Fix them all?", take the ones that break one of the five rules or something you said you would see, and say no to the rest with a reason; if none break a rule, say no to all of them. If you took any, the agent runs implement itself in this session: at its go gate say go. Never ask to merge.`,
    after: { kind: "merge-code-stack" },
  },
];

export const CHAIN: Verb[] = ["equip-local", "equip-test", "envision", "specify", "design", "implement", "review", "test"];

export function step(verb: Verb): Step {
  const found = STEPS.find((s) => s.verb === verb);
  if (!found) throw new Error(`no step named ${verb}`);
  return found;
}
