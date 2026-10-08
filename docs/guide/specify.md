# /codefall-specify

`/codefall-specify` turns a feature idea into a *spec*: a document that states what the finished
feature must do, precisely enough that another session can build it without asking you again. Use it
before planning any feature larger than a small fix.

```
/codefall-specify Travelers can export a trip itinerary as a file.
```

The skill interviews you, checks whether the feature already exists, writes the spec, and, in a
project that uses GitHub Issues, copies it there. It decides *what* will be true when the work is done and never *how* it gets
built; that is [`/codefall-plan`](plan.md)'s job.

## What it writes

The skill writes the spec to `docs/specs/SPEC-003-slug.md`, where the number is the next one free.
A spec holds one cohesive feature, divided into *requirements*. Each requirement is a capability
someone can use, written as a user story with its own acceptance criteria:

```
As a traveler, I want to export my itinerary, so that I can share it with people who don't use the app.
```

A requirement is the unit somebody picks up and builds, so each one becomes its own GitHub issue
when the project uses them.
Around the requirements, the spec has a Context section that explains why the feature matters, and
optional sections for key terms, edge cases deliberately left alone, existing behavior that must
keep working, assumptions, what is out of scope, design notes, and open questions.

## How acceptance criteria are written

An *acceptance criterion* is a testable statement of what the feature must do. Codefall writes them
in [EARS](https://alistairmavin.com/ears/), the Easy Approach to Requirements Syntax, published at
Rolls-Royce in 2009 and used here unchanged. EARS limits each criterion to six sentence patterns,
with the clauses always in the same order:

```
The system SHALL order exported segments by departure time
WHEN a traveler selects export, the system SHALL produce a file containing the itinerary
IF the trip is missing a departure date, THEN the system SHALL name the missing field
WHILE an export is in progress, the system SHALL show progress and allow cancellation
```

Each criterion reads as an obligation, not a description, so the criteria are the requirements, and
there is no second list to keep in step with them. Failures get their own keyword, `IF … THEN`,
which makes the error cases visible as a group instead of scattered among the normal ones.

## How identifiers work

Every identifier starts with its spec's number, so `grep SPEC-003` finds the document, its
requirements, and every test and ticket that cites them:

```
SPEC-003                        the spec
SPEC-003-REQ-01                 a requirement
SPEC-003-REQ-01-AC-01           a criterion
```

Numbers are only ever added, at every level. A retired number is never reused, so a test that cites
`SPEC-003-REQ-01-AC-04` never comes to mean something else.

## How the spec reaches GitHub

Whether the spec is copied anywhere is the `tracker` setting in `.codefall/settings.json`, which
`init` asks about. `github` copies specs and bug reports to GitHub Issues; `beads` copies them
nowhere, and the document is the whole record. Beads holds the tasks under either setting, from the
moment [`/codefall-plan`](plan.md) creates them.

Under `github`, the spec document is the source of truth, and the GitHub issues are a copy of it,
which Codefall calls the *mirror*. GitHub gets one parent issue for the spec and a sub-issue for each requirement.
Each requirement issue carries its story and criteria in full, so nobody has to click through to the
spec to work the ticket. Running `/codefall-specify` on the spec again regenerates those issue
bodies. When the spec is a `Draft`, every issue in the set gets a `draft` label.

GitHub Issues is the only tracker that gets a copy today; Jira and Linear are planned. If the mirror fails,
the skill still writes the spec, tells you it is pending, and gives you the command that fixes the
problem.

## How the spec lands

The skill shows you the whole spec before it writes anything. It then writes the spec on its own
branch, `spec/SPEC-003-slug`, waits for the mirror to write the issue number back into the document,
commits, pushes, and opens a pull request without asking you. Its report gives you the pull
request's address.

To merge it, add the `auto-merge` label or have someone approve it. Either one triggers a GitHub
Action, installed with `/codefall-equip landing`, that merges a labelled or approved pull request
when it touches only document paths. No skill ever adds the label or approves. In a project without
the Action, you get the same pull request and a person merges it. This holds for every skill that
writes a document, and [`docs/landing-documents.md`](../landing-documents.md) explains it. Document
pull requests do not stack on each other; each one branches from `main` and lands on its own.

## Mockups, visions, and large features

**Mockups.** When a requirement has a screen and no mockup yet, the skill runs
[`/codefall-mock-up`](mock-up.md) for it in the same run, on the same branch, so the spec and its
mockups arrive together. If you have a mockup already, it is imported instead. If you choose to skip
it, the requirement's Design notes say `Mockup: pending`, its issue gets a `requires-mockup` label
when the project has one, and `/codefall-plan` makes the mockup before it starts. The spec links to each mockup by path under **Design notes**.

**Visions.** If a [vision](envision.md) covers the feature, the skill offers it as context and adds
the spec's identifier to the vision's `Related` line. A vision is never required. When the interview
shows a `Draft` or `Ready` vision is wrong, the skill amends it on the same branch; an `Active`
vision is frozen.

**Large features.** A feature too large for one cohesive spec becomes sibling specs, not a parent
and its children. The vision above them is what groups them, which is why a vision's `Related` line
holds a list.

**Questions nobody can answer.** When you cannot answer a question of fact, such as how the existing
system behaves in some case, the skill puts it once to the project's *consult agents*: the agents
[configured](configuration.md) to give a second opinion, or a subagent of your current coding agent
when none are. The answer comes back as a proposal you confirm. Questions of preference are never
sent to them.

## Statuses

A spec's `Status` describes the document only. Whether the work is queued, underway, or done is
Beads' to say once the plan's tasks exist, and the issue tracker's before that in a project that
has one.

| Status | Meaning |
| --- | --- |
| `Draft` | You said you are stopping and will come back to it |
| `Ready` | Written and agreed; the normal end of a session |
| `Archived` | Superseded or dropped; the file moves to `docs/specs/archive/` |

To change a spec later, run the skill on it. It can promote or reopen a spec, edit it (new
requirements and criteria take the next free number), split it into siblings, or archive it. An
archived spec keeps its identifier and gains a `Replaced by` line when something took its place.
