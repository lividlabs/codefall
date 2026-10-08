# The lenses

What each lens asks. Read at the confirmation, which names the lenses that will run, and at step 3.

## Contents

- Code lenses
- Document lenses
- Notes on particular lenses

## Code lenses

| Lens | The question |
| --- | --- |
| `correctness` | Logic errors, off-by-one, wrong conditionals, missing guards, unreachable paths, null and empty and boundary inputs |
| `failures` | Errors swallowed, caught and ignored, logged and continued past, or returned and never checked |
| `behaviour` | Anything a consumer would notice that changed, especially where it looks unintentional |
| `tests` | Does the work carry the tests it needs, and do they test behaviour rather than implementation |
| `types` | Where new types appear: do they make invalid states unrepresentable, or push that job onto every caller |
| `conventions` | `AGENTS.md` and the ADRs — boundary rules first, since those are the encoded architecture |
| `comments` | Do the comments say what the code does, and does anything the work touched now lie |
| `docs` | Does the code contradict a document that describes it — `README.md`, `AGENTS.md`, a plan, an ADR — and is that document one the project may amend or one that is immutable |
| `simplify` | Reuse it should have used, dead code, nesting that early returns would flatten |
| `local` | Does the change add infrastructure, a dependency, a migration, or generated code, and leave the project's declared `start` and `update` scripts as they were — so the next teammate's refresh brings an environment current that the change has made stale |
| `security` | Injection, authn/authz bypass, data exposure, secrets in the diff |

## Document lenses

Every document target gets `structure` and `status`, plus the lenses for its kind. Each verb owns
the rules its documents are held to — required sections, status vocabularies, identifier forms,
EARS, the Tasks callout. Read them in `codefall-envision`, `codefall-specify`,
`codefall-plan` and `codefall-scaffold` when a lens needs the detail; they are not restated here.

| Target | Lens | The question |
| --- | --- | --- |
| all | `structure` | Required sections present; no empty headings; no template guidance left in; identifiers written in full |
| all | `status` | Status line is one of the document's own values with a real date; `Archived` is under `archive/`; a superseded ADR names one that exists |
| vision | `floor` | A problem stated, who feels it, and why now — not a solution wearing a requirement's clothes |
| vision | `scope` | What this is not, stated. No `## Non-goals` means it has not said where it stops |
| vision | `testable` | Could what it asks for be tested at all, in principle? Not how — whether |
| spec | `trace` | Does not run without a vision. Reports a spec requirement with no vision parent as **unframed**, and a vision requirement no spec requirement reaches as **unaddressed** |
| spec | `criteria` | Do the acceptance criteria hold to EARS, as `codefall-specify` defines it — its patterns, its rule about failure behaviour, and its ban on implementation vocabulary |
| spec | `precision` | Ambiguity a reader could resolve two ways; undefined terms; missing non-functional requirements |
| spec | `stories` | Every requirement has a user story with the `so that` clause `codefall-specify` makes mandatory |
| plan | `coverage` | Every spec requirement has a home in the plan, or the plan says why not |
| plan | `decisions` | Hard-to-reverse choices with no ADR; assumptions the spec does not guarantee; a conditional section present with nothing behind it |
| plan | `tasks` | The Tasks callout is staged or created, as `codefall-plan` defines them, over a table of the four columns and no status column; no ID on the `Retired:` line reappears as a row |
| adr | `alternatives` | Alternatives genuinely weighed, not asserted and dismissed in a clause |
| adr | `consequences` | Consequences stated, including the ones that cost something |
| adr | `coherence` | No contradiction with another accepted ADR; project decisions numbered bare `ADR-NNN` rather than continuing an inherited sequence |

## Notes on particular lenses

**A `docs` finding is not always a documentation fix.** An ADR contradicted by the code is never a
docs edit — ADRs are immutable, and the only in-place change allowed is flipping Status to
`Superseded by <id> — <date>`. Such a finding says which it is: the code is wrong, or the decision
changed and nobody wrote the superseding ADR. A `README.md` or an `AGENTS.md` that has fallen behind
is an ordinary fix.

**`local` looks at the diff's shape, not its code.** A compose file, a migrations directory, a
lockfile, a codegen config, or an `.env.example` changed with no change under the commands
`.codefall/settings.json` declares under `local` is the finding, and the fix is `codefall-equip`'s
procedure applied to the change. A project with no `local` block gets one finding saying so, not
one per file. A change under those scripts that drops, resets, or deletes is a finding too, against
the contract in `codefall-equip`.

**`tests` also reads the test cases the work carries.** A criterion in a case file citing no source
— neither a spec criterion in full nor a requirement it is `derived` from — is a finding, and so is
a generated spec sitting beside no case, since a case is written before its spec.

**`simplify` is a lens, not a separate verb.** `codefall-implement` already ran the host's own
simplify pass over each bead's diff, so a simplification finding on that code is either something
that pass missed or code implement never saw. Say which.

**Neither half of `trace` is a defect.** A spec requirement with no vision parent is often
legitimate — a precondition the vision never anticipated. An unaddressed vision requirement may
mean the vision should catch up. The finding asks the question; it does not assert the spec is
wrong. It carries `minor`, rising to `important` when the unaddressed requirement is one the vision
called out as the reason for the work.

**An ADR edited after ratification is itself a serious finding.** Detect it rather than assume it:
`git log --format='%h %ad' --date=short -- <adr>` against the date on its Status line. Commits after
that date that are not the single permitted edit — flipping Status to `Superseded by` — are the
finding, and the log gives you which ones to cite.
