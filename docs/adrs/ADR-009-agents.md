# ADR-009: Agents

## Status

Accepted — 2026-09-26

## Context

Two verbs already, and more to come, need another reader than the session running them. `review`
runs its reviewer in a subagent of the current harness by default, or in another harness when the
user types `via=codex` or one of four other names, because the context that finds a problem must not
be the one that fixes it. The next need is a consult: a run that cannot settle a technical question
on its own, a worker whose task contradicts the design, a review finding the reviewer is unsure of,
a decision under a persona whose owner cannot answer it. Both are the same call with a different
prompt: run another agent, read-only, and let the session decide what to do with what it says.

Before this record, which agent to run was decided per invocation and by whoever remembered to type
`via=`. Nothing in the project said which reviewer the team had settled on, nothing said what to do
when that reviewer's binary was missing on one machine, and nothing said what to reach for when the
first reader failed. The review script found a harness by `command -v <name>`, which worked only
because the harness names happened to be binary names, and which the previous record made a rule.

Several shapes were weighed.

**A map of agents keyed by name.** The first draft: `agents: {architect: {...}, security: {...}}`
with a default named beside it. A map cannot say which agent to try next, and the two real needs a
project has, "this machine does not have codex" and "the first reader got stuck", are both answered
by an order.

**A per-user substitution layer.** A person who cannot run the configured harness maps its name to
one they can, in a file that is not checked in. This solves availability by asking every person to
declare it, when the run can find it out for itself by looking on PATH. It survives here only for
what genuinely describes a person: the persona, which is a later decision.

**Detecting the harness by environment variable or by walking the process tree.** Claude Code sets
several variables; the other four harnesses were not verified to set any. A process walk from the
skill's shell reaches the harness binary, but four of the five harnesses read the same skills
directory, a wrapper script adds a hop, and a renamed binary gives a wrong answer that a per-harness
order would then act on. Every harness states its own name in the prompt it gives its agent, and a
skill can simply ask.

**Detection in preflight.** Putting the process walk in the shared preflight script would have made
every skill agree on a wrong answer at once. Preflight stays out of this: the answer comes from the
agent, checked against the directory the skill was loaded from.

**Two orders per use, presented as alternatives.** A flat list and a per-use order are not
alternatives but layers: the list defines the agents and their default order, and a use carries its
own order only when it wants a different one.

## Decision

### The list

`.codefall/settings.json` carries an optional top-level `agents` array. Each entry has a `name`, a
lower-case slug a person types after `via=` and reads in a report; a `harness`, one of the five
harness names or `current`; and optionally a `model`, a string passed to that harness untouched. The
order of the array is the default order every use walks. Names are unique.

Absent, or empty, the list is one entry: `subagent` on `current`. `codefall init` writes that entry
explicitly into a new project's settings, and adds nothing to a project's existing file, because
absent already means the same thing.

### `current`

`current` is the harness running the session, whatever it is, and the agent it names is that
harness's own subagent. One checked-in entry means Claude Code's subagent in one person's session and
Muse's in another's. It is always runnable, which is what makes it the floor of an order: a list that
ends on `current` always has a reader a run can start. `doctor` warns about an order that names no
agent on `current`, and a project that wants a hard stop when its external reviewer is down may
leave it off on purpose.

### Per-use orders

A use that should walk a different order from the default carries its own `agents` key holding an
ordered subset of the names: `review.agents` now, `consult.agents` when consult lands. The same word
means the same thing wherever it appears. An absent key means the top-level order.

### The per-harness override

An optional top-level `agentsByHarness` object, keyed by harness name, holds an ordered subset of
the names for a session running in that harness. It exists because a second opinion from the model
you are already using is worth less: in Claude Code the first external reader should probably be
Codex, and in Codex it should be Claude. A harness with no entry, or a session that cannot say which
harness it is in, walks the flat order. The key is `agentsByHarness` rather than a second nesting
under `harnesses`, which already means something else in the file.

### Resolution

A use resolves its order in three steps and takes the first that answers: the per-harness override
for the harness the session is running in; the use's own `agents` key; the top-level list.

### Walking the order

A run walks the resolved order from the first entry. An entry whose harness is not on PATH, not
authenticated, or refuses the model at startup is skipped before any prompt is sent. An entry that
ran and got stuck, meaning a timeout, a non-zero exit, output that will not parse after the one retry
the skill already allows, or a consult that answers that it cannot settle the question, advances the
walk to the next entry with the failure folded into the prompt. Each entry is tried once; the walk
ends at the end of the order. The report names every agent tried and why each was skipped or
failed, and the findings file's `reviewer` records the one that answered. A configured fallback that
is reported is not the silent fallback the review skill forbids.

What is not stuck: a reviewer returning no findings has completed a review, and a consult giving a
confident answer the session disagrees with has completed a consult. Only mechanical failure and an
explicit "cannot settle" advance the walk, otherwise the fallback becomes a way to shop for the
answer wanted.

### `via=` is a one-run override

`review`'s `via=` argument keeps working and takes either a name from the list or the raw
`harness[:model]` form it takes today. It overrides the resolved order for that run and writes
nothing.

### Whoami

The agent states which harness it is running in, from the five names, because every harness says so
in the prompt it gives its agent. The skill checks the answer against the directory it was loaded
from: a skill under `.claude/skills/` is Claude Code and one under `.agents/skills/` is one of the
other four. A harness the agent cannot name is `unknown`, which falls through to the flat order and
loses only the per-harness override.

### A consult proposes

An agent run this way is read-only. It proposes, and the session decides, exactly as the reviewer
proposes and the session triages. A consult never writes a file, never decides a hard-to-reverse
choice, and never stands in for a human the rule already names: a disagreement with the user is a
preference, not a question, and a missing precondition is an exit, not a question. One round per
stuck point, which may ask several agents at once; never a second round because the first was
inconclusive.

### Project config

The list, the per-use orders, and the override are facts about the project: checked in, validated by
the schema and the settings module, reported by `doctor`, and changed through a pull request. The
ignored per-user file, when it exists, holds the persona and nothing else for now, and the persona
itself is a later decision.

## Consequences

- **Which agent reviews is a team decision, recorded once.** A project that has settled on a
  reviewer no longer depends on someone typing `via=`, and a run on any machine walks the same
  order.
- **A missing binary on one machine is not a configuration problem.** It is skipped and reported,
  and the order ends on something the machine can run. The cost is that a person can believe their
  project reviews through Codex when their own runs never reach it; `doctor` says so per agent, and
  every report names who actually ran.
- **Fallback is bounded.** Each agent once, one round per stuck point, and the end of the order is
  the end. A run cannot loop through readers, and it cannot retry until it hears what it wants.
- **The harness name comes from the agent, not from detection.** This works in every harness on day
  one and needs no script; the risk is a wrong self-report, checked against the skills directory,
  and the cost of a miss is a weaker second opinion, never a wrong write.
- **Settings gain their first cross-field rules.** An order names agents the list defines, and the
  override is keyed by harnesses the format knows. The settings module checks both across the
  document, which no field's own check could.
- **Every external agent costs latency, tokens, and an account.** A panel asked at once multiplies
  that. The `agentsByHarness` override and per-use orders exist so a project can keep the default
  walk short and reach for the panel only where a hard-to-reverse choice justifies it.
- **`current` is a value the schema knows and the harness module does not.** It names no directory
  and installs nothing, so it belongs to the settings format, and the harness module stays the
  roster of what `init` can set up.
- **Nothing uses the list yet.** This record and the settings land first; `review` adopts the
  order and makes `via=` an override next, and consult follows on the same list. Until then the
  block is validated and reported and read by nothing, which is the same footing the `review` block
  stood on when it landed.

## Related

- ADR-006, *Install Layout* — the two skills directories the whoami check reads.
- ADR-008, *Upstream Amendments* — the amendments a consult may inform and never make.
- `docs/decision-log.md`, *A harness is named for its binary, 2026-09-26* — why an agent's harness
  needs no mapping to the command that runs it.
- `docs/decision-log.md`, *Agents are an ordered list in project settings, 2026-09-26* — the
  alternatives seen and not taken, in brief.
- `extensions/skills/codefall-review/reference/reviewers.md` — the three reviewers and the `via=`
  forms this record turns into an override.
- `cli/schemas/settings.schema.json` — where the list, the per-use order, and the override are
  published.
