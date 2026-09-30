# The agentic run

How a case in the `agentic` modality is driven and judged. The session running it is the runner —
this one, or a subagent of it running one variant: setup runs as scripts, the case's `## Steps` are
worked through a driver, and each criterion gets a verdict from what the run made observable. Read
it before anything is driven.

## Contents

- The driver
- Bringing the application up
- Setup runs as scripts
- Working the steps
- Judging a criterion
- Observing transient state
- A negative observation needs a second method
- Real side effects
- The anomaly sweep
- What the run writes
- What a run never does

## The driver

**The case's surface decides the kind of driver.** A browser tool for a web or desktop-shell
surface; the shell for a command-line or HTTP surface, and the shell is always there.

**Read this session's own tool list and name the driver in the confirmation.** There is no registry
of drivers to consult — a list written down elsewhere goes stale, and what matters is what the
session that will drive can actually call. A subagent running one variant reads its own list and
proves the driver again before it drives anything.

**One live call proves it before the run depends on it.** A single request through the driver to the
application's health endpoint or its front page, and the result is what proves the driver, not the
tool's presence in the list. **A failure is reported as a failure** — never routed around by
switching to a second tool mid-run, and never replaced by reading the application's own logs instead
of driving it.

**With no working driver, the agentic modality is refused for that case.** Say so plainly, say which
surface the case needs, and say that a project supplies a driver by checking an MCP server entry
into its harness configuration, where the whole team gets the same one. Refusing the modality is not
refusing the case: its `spec` modality, when it declares one, still runs.

## Bringing the application up

The application runs through the project's declared `start` command — the one `local.start` names in
`.codefall/settings.json`. It is not started any other way, and a run never starts a service the
project did not declare.

When `local` is undeclared, stop and name `/codefall-equip`. When the application is down after
`start`, stop and name the remedy for the branch the run is on: `/codefall-refresh` on the default
branch, and the declared `local.update` command on any other, since refresh would leave the branch
under test. The run has nothing to drive, and repairing the environment is not this verb's work.

## Setup runs as scripts

**The Preconditions section names commands. Run them as written.** Setup is deterministic; clicking
through what a script already does costs time and fills the report with noise that is not about the
product.

State a case needs — a seeded account, a saved preference, a forced inventory condition — is forced
through the commands `<root>/AGENTS.md` and the case's Preconditions name. A state with no such
command is unforceable, and its criterion was dropped when the case was written.

## Working the steps

Follow `## Steps` in order, through the driver, once per variant.

- **Every message is a fixed string from the case's `messages` map**, merged with the variant's own.
  Never reworded, never composed from variant fields, never improvised when a key is missing.
- **Setup-only retry, bounded at three or four attempts.** When the product does not reach the state
  under test — the search never ran, the list never rendered — repeat the same fixed input up to
  three or four times. **Record every attempt count, including on eventual success**: a pass that
  took three attempts is data about the product.
- **Assertions never retry.** Once the state under test is on screen, a criterion that is not met is
  a result, not a setup hiccup.
- **Prefer a script to the driver for anything a script can see.** An HTTP read, a seeded-state
  check, a returned document. Reserve the driver for what only the driver can observe.

## Judging a criterion

**Judge what is observable against the criterion's words.** Nothing else decides a verdict.

**Never read application code to judge.** Not the component, not the route handler, not the spec
mid-run. The session running the test does not consult the implementation it is testing. Reading the
project's own testing scripts in order to run them is fine; reading application code to decide what
"correct" means is not.

| Verdict | When |
| --- | --- |
| `held` | What the criterion asks for was observed |
| `failed` | The state arose and the criterion was not met |
| `skipped` | Not applicable to this variant — the case names which variants a criterion covers |
| `unreachable` | The state to observe it never arose |

**Record a pass when you see one.** If the criterion's condition was on screen, the verdict is
`held`, even when it was observed in passing rather than by a deliberate check. Withholding a pass
because the observation felt informal produces a verdict nobody can act on: there is no defect to
fix and no evidence to follow. `unreachable` means the state never arose; it does not mean the
observation was made and not written down.

## Observing transient state

Some criteria describe a state that lasts under a second — a control disabled while a request is in
flight, a spinner, a button that re-enables. **Every driver call is a round trip**, so acting in one
call and checking in the next reliably misses these. Polling harder does not help.

Put the watching inside the page instead. In one evaluate-style call, install an observer and
perform the action; in a later call, read back what the observer recorded.

```js
// one call: start recording, then act
() => {
  const target = document.querySelector('[aria-label="Maximum stops"]');
  window.__probe = { saw: false, samples: [] };
  const disabled = () => target.matches('[disabled], [aria-disabled="true"]');
  const record = () => { const d = disabled(); window.__probe.samples.push(d); if (d) window.__probe.saw = true; };
  const mo = new MutationObserver(record);
  mo.observe(target, { subtree: true, attributes: true });
  window.__probe.stop = () => mo.disconnect();
  record();                               // baseline, before the action
  target.querySelector('button').click();
};
// a later call: read what the page saw
() => { const p = window.__probe; p.stop?.(); return { saw: p.saw, samples: p.samples }; };
```

A `samples` of `[false, true, false]` is the transition, captured; latency stops mattering because
the page did the watching. Only mark such a criterion `unreachable` after this comes up empty too.
The same shape covers other timing-bound observations: record request starts, count renders, capture
a notification's appearance, then read the log back. On a non-browser surface the equivalent is a
log or a stream the run is already reading, watched from the moment before the action.

## A negative observation needs a second method

When a probe reports something **absent**, check it a different way before it becomes a finding.
Over-narrow queries produce false absences: a prefix match misses a line written mid-paragraph, a
leaf-node text scan misses text that sits one level up the tree. Both read as rendering defects and
neither is one.

Cheap second methods: dump the whole region's text and read it; walk up two or three ancestors and
print their text; search the whole document for a distinctive substring; on a shell surface, re-read
the command's full output rather than the filtered line. **Absence found by one narrow selector is
not absence.**

## Real side effects

A run against real services creates real things: orders, bookings, accounts, messages, rows in
someone else's system.

- **Record each one the moment it exists**, in the run's working notes, with the identifier the
  product gave it. A run that dies before its cleanup leaves the report as the only trace.
- **Clean up the way `<root>/AGENTS.md` says.** That file carries the project's rule, because only
  the project knows what cancelling one of its orders means.
- **Never leave one unrecorded.** A side effect the run could not clean up goes in the report with
  its identifier and what remains to be done about it.

## The anomaly sweep

**Every run ends with one.** User-visible wrongness the run noticed that no criterion asked about: a
broken layout, a stale spinner, copy that contradicts itself, an error surfaced on screen, a
duplicated row. List it beside the verdict and never fold it into the verdict — an anomaly is a
finding, not a failed criterion.

## What the run writes

A report per run, written whether the run passed, failed, or ended part way: verdicts per variant,
the per-criterion table, attempt counts, side effects and their disposition, the driver that ran,
and the anomaly sweep. The shape is in `run-report.md`; the classification of anything the run found is
in `triage.md`.

## What a run never does

- **Never mocks, fakes, or intercepts** any layer. A state that cannot be forced for real is
  `unreachable`, not simulated.
- **Never edits a case file, a spec, or application code.** A run produces a report and triage
  notes; changes are separate work the user starts.
- **Never reads application code to decide a verdict.**
- **Never works around a driver failure**, and never substitutes a different surface for the one the
  case names.
- **Never reports an all-green first run as a success without saying so.** The first run of a new
  case finding nothing is worth naming in the report: it is the shape a case written from the code
  produces.
