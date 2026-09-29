# codefall-report — where it came from

What this took from elsewhere, and what it deliberately did not, so nobody re-adds it. None of this
is instruction — the skill is the instruction.

## Taken

**From `codefall-specify`**, the shape: an interview that pushes back on vague answers for at most two
rounds, a recap where every line is a sentence, a canonical document in the repository with a
generated tracker mirror, append-only identifiers, and a status that describes the document and never
the work. A bug report holds the place a spec holds for a feature, which settles the open question
`docs/PLAN.md` carried.

**From issue templates in common use**, the fields: steps to reproduce, expected result, actual
result, environment, frequency, and the last version that worked.

**From `codefall-test`'s agentic run**, the reproduction rules: the declared `start` and nothing
else, a driver proved by one live call, a failure reported rather than routed around, and nothing
mocked or changed by hand.

## Dropped

**Finding the cause.** It is `codefall-design`'s, in `reference/bugs.md`, whether design runs on its
own or inside `codefall-fix`. A report that guessed at a cause would hand design a conclusion to
unlearn.

**Reproduction as a gate.** An attempt that fails does not stop the report. Bugs that only happen in
production, on particular data, or some of the time are real, and refusing them would lose them.

**Priority.** Severity is the reporter's judgment of impact; priority is the team's decision about
order, and a report that set it would be guessing.

**Embedding images in the issue.** `gh` cannot upload them, so evidence is committed beside the
report and the issue lists it by path, the same answer `codefall-specify` reached for mockups.
