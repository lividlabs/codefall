# The case file

The format of a test case. Read it before running a case, and before writing one —
`codefall-implement` writes cases from a bead's approved criteria and follows this file.

## Contents

- Where a case lives
- The identifier
- Frontmatter
- Fixed messages
- The body
- Where a criterion comes from
- Variant-conditional criteria
- Dropped criteria
- A case precedes its spec
- The sibling pair
- An example

## Where a case lives

```
<root>/test-cases/<area>/<slug>.md
```

`<root>` is the testing root `test.dir` declares in `.codefall/settings.json`, and `<area>` is a
product area — `checkout`, `search`, `billing` — not a layer and not a runner. `<slug>` is words and
hyphens: no numeric prefix, no `CASE-01` registry.

`.codefall/shared/check-cases.sh` checks everything in this file that a script can check.

## The identifier

**A case's id is its path minus the extension, relative to `test-cases/`** — `checkout/place-order`.
There is no registry beside the directory; the directory of case files is the list.

The frontmatter `id` repeats the path deliberately. A file moved without its frontmatter is then
caught by the check script rather than at review. **Every test title begins with the case id**, so a
failing test names its case without anyone looking it up; a variant-scoped title
(`checkout/place-order — [saved-card] …`) satisfies that.

## Frontmatter

YAML, four keys, three of them required.

| Key | Required | Rule |
| --- | --- | --- |
| `id` | yes | The path minus `.md`, relative to `test-cases/`. |
| `modalities` | yes | Non-empty. `spec` and `agentic` only, no duplicates. |
| `variants` | yes | Non-empty list of mappings, each with a unique `name`. |
| `messages` | no | Map of key to fixed string. Only meaningful on a conversational surface. |

**`modalities` takes two values and no others.** There is no `manual` modality: a case nobody runs
is a coverage claim with nothing behind it.

**Variants are named input sets**, and one case runs once per variant. Variant data lives in the
frontmatter and nowhere else — never in an environment variable, never in the spec. A difference in
inputs with the same expected outcomes is a variant; a difference in expected outcomes is a separate
case.

Every other field on a variant is the case's own: the route, the dates, the seeded state, the
carrier. Name them for what they are and let the body cite them.

## Fixed messages

`messages` holds the exact strings a run sends to a conversational surface — a chat product, an
agent under test. A surface with no conversation has no use for the key and omits it.

- **A message is used as written.** It is never reworded, never composed from variant fields, and
  never assembled at run time.
- **A variant's own `messages` overlays the case-level map**, key by key, so a route-specific or
  date-specific wording sits with the variant it belongs to and the shared ones stay shared.
- **A missing key is an error**, not a reason to write a message.
- **`{named}` tokens** mark the one thing a case file cannot pre-write: a value the run itself
  minted, such as the confirmation code of an order it just placed. The template stays fixed; only
  the token's value comes from the run. Every mismatch is an error — a token with no value, a value
  with no token, and values supplied for a template that has no tokens at all. Tokens are
  `{name}`-shaped, letters and digits; a brace pair holding anything else is prose and is left
  alone.

## The body

Five sections, in this order. An agentic case carries all five; a spec-only case may omit `Steps`.

- **Preconditions** — what has to be true before the case runs, and the commands that make it true.
  Setup is named here as scripts to run, never as something to click through.
- **Variants** — a table: the variant, what is different about it, and why that difference is worth
  a run. This is where probe output and pinned data are recorded, with the date they were checked.
- **Criteria** — numbered, each carrying its citation. This is the case.
- **Steps** — for an agentic run: what to do, in order, and what to verify at each point. Required
  when `modalities` includes `agentic`.
- **Reporting** — what a run of this case reports beyond the standard shape, if anything.

## Where a criterion comes from

**The spec's criteria are the floor.** A criterion that restates one cites it in full —
`SPEC-003-REQ-01-AC-01` — which is the form `codefall-specify` writes and the form `grep` finds. A
case covers every criterion of the requirement it tests, or records why one was dropped. A bug
report's criteria are a floor the same way, cited as `BUG-012-AC-01`.

**A criterion may go beyond the spec, marked `derived`.** It cites the requirement it elaborates —
`SPEC-003-REQ-01` — and says in one line what it adds: a boundary, a negative path, error handling.
Boundaries and negative paths are the work; a case that could only restate the spec would be a
transcription exercise.

**The one banned source is the implementation.** Not the code, not clicking through the application
to see what it does, and not the implementing agent's own pull request text or notes — in a codebase
built by agents, those were written by the same author as the code. Reading code for mechanics is
allowed and often necessary: a control's role, its accessible name, the shape of a response, so that
a spec can drive it. Reading code to decide what should happen is not.

**A gap the expansion exposes is appended to the spec.** Where the spec is silent or ambiguous, the
verb that found the gap offers the criterion to the spec in its own run — `codefall-plan` when
the case's criteria are drafted — and the case cites the appended identifier. Spec numbering is
append-only, so the addition takes the next number and nothing already cited moves. `derived` is
what a criterion stays when the spec was offered it and declined.

Derived criteria are shown to the user before the case file is written, at the point where the task
plan is approved.

## Variant-conditional criteria

A criterion observable only under some variants names those variants. A spec skips that assertion
elsewhere; an agentic run records the criterion as `skipped` for the variants it does not name —
which is not the same as unreachable, and reads differently in a report.

## Dropped criteria

A state that cannot be reached through the product's own interfaces is unforceable. **The criterion
is dropped, and the drop is recorded in the case file** with its number, its citation, and the
reason:

```markdown
11. **DROPPED — unforceable.** SPEC-003-REQ-02-AC-04, the supplier-unreachable case: nothing is
    mocked, and the supplier cannot be made to fail on demand.
```

Recording it is what makes the omission a decision rather than an oversight. Nothing is mocked,
faked, or intercepted to reach a state, at any layer.

## A case precedes its spec

**A case file is written before the spec generated from it**, never afterwards to describe what a
spec already does. Writing it afterwards is the same mistake as writing criteria from the code, made
one file at a time.

## The sibling pair

A case declaring `spec` is a pair: `<slug>.md` beside a file whose name begins with the same slug,
in whatever suffix the project's runner collects — `place-order.spec.ts`, `place-order_test.go`.
Nothing here knows a runner's suffix, and nothing needs to.

The pairing is checked both ways. A case declaring `spec` with no sibling fails; a sibling whose
slug has no `.md` fails too, because a case is written before its spec.

## An example

````markdown
---
id: checkout/place-order
modalities: [spec, agentic]
messages:
  order: "Place the order with the card ending 4242."
  askStatus: "What is the status of order {orderId}?"
variants:
  - name: card
    savedCard: false
  - name: saved-card
    savedCard: true
    messages:
      order: "Place the order with my saved card."
---

# checkout/place-order

One order placed through the real checkout, against the real payment sandbox.

## Preconditions

`npm run e2e:seed` before each variant; `npm run e2e:save-card` as well on `saved-card`.

## Variants

| Variant | Difference | Why |
|---|---|---|
| `card` | card entered during checkout | The first-time path, where the card form is on screen. |
| `saved-card` | a card already on file | The saved-instrument path, where no form appears. |

## Criteria

1. An order placed with a valid card reaches a confirmation naming the order id.
   *(SPEC-003-REQ-01-AC-01)*
2. The confirmation states the total charged, matching the cart's total.
   *(SPEC-003-REQ-01-AC-02)*
3. A declined card leaves the cart intact and states why, without charging.
   *(derived from SPEC-003-REQ-01 — the negative path; the spec names only the accepted card.)*
4. With a card on file, no card form is shown. *(SPEC-003-REQ-03-AC-01)* — `saved-card` only.
5. **DROPPED — unforceable.** SPEC-003-REQ-01-AC-05, the gateway-timeout case: nothing is mocked,
   and the sandbox cannot be made to time out on demand.

## Steps

1. Run the Preconditions for this variant, then sign in and fill the cart.
2. Send `order`. Verify criteria 1 and 2 on the confirmation, and criterion 4 on `saved-card`.
3. Send `askStatus` with the order id the run just received. Verify the status names that order.
4. Cancel the order per the cleanup rule in `<root>/AGENTS.md`, and record the order id the moment
   it exists.

## Reporting

The order id and whether it was cancelled, beside the verdict.
````
