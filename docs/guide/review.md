# /codefall-review

`/codefall-review` has a second agent read your work and list what is wrong with it. You then
decide which problems to fix, and the session you are in fixes them. Use it before you merge a
branch or a pull request, or whenever you want a second reading of code or a document.

```
/codefall-review
/codefall-review 51
/codefall-review the codepaths on the backend that handle flight fulfillment
```

The first form reviews your uncommitted changes. The second reviews pull request #51. The third
searches for the code you described, shows you the files it found, and asks before it reads any of
them.

Codefall's skills are instructions your coding agent follows when you type a slash command; the
[README](../../README.md) introduces them.

## What can I point it at?

| You give it | It reviews |
| --- | --- |
| nothing | your uncommitted work |
| a branch name | that branch, compared with the default branch |
| a pull request number or URL | that pull request |
| two commits, `<from>..<to>` | the work between them |
| a file or directory path | that path as it stands |
| a document identifier, such as `SPEC-004`, or a path under `docs/` | that document, checked against the document it refines |
| an epic ID or `PLAN-NNN` | all the work built for that plan |
| a description in words | the code a search finds, once you confirm the list |

An *epic* is the bead that groups every task from one plan, and a *bead* is one task in
[Beads](https://github.com/gastownhall/beads), the task tracker Codefall keeps in your git
repository.

Before anything runs, the skill tells you what it resolved your request to and which questions it
will ask. You can drop any of the questions at that point.

## What does it refuse to review?

It reviews only work that can still change, because a fix needs somewhere to land. It refuses these
five things and says which one it hit:

- a merged or closed pull request
- a merged branch
- a superseded ADR (architecture decision record)
- an archived vision or spec
- a single commit

## Who does the reviewing?

The review runs in a different agent from the one that fixes the problems. An agent that both finds
and fixes problems grades its own work, and a second reading by the same agent has the same blind
spots as the first. The session you started triages the findings with you and applies the fixes.

The reviewer comes from the `review` list in `.codefall/settings.json`. Codefall reads the list for
the coding agent you are using, or the `default` list when yours has none, and takes the first agent
on it that this machine can run. With nothing configured, the reviewer is a subagent of your current
coding agent. Every reviewer runs in a read-only mode.

To use a different reviewer for one run, add `via=` with a coding agent and an optional model:

```
/codefall-review 51 via=codex:gpt-5-codex
```

If an agent on the list is not installed, the skill skips it. If one fails, the skill passes the
same request to the next. The report names every agent it tried. [Configuration](configuration.md)
explains how to set the lists.

## What does it look for?

For code, the reviewer asks eleven separate questions, which Codefall calls *lenses*:

| Lens | The question |
| --- | --- |
| `correctness` | Are there logic errors, wrong conditions, or missing guards? |
| `failures` | Are errors swallowed, ignored, or never checked? |
| `behaviour` | Did anything a user would notice change, especially by accident? |
| `tests` | Does the work have the tests it needs? |
| `types` | Do new types prevent invalid states? |
| `conventions` | Does the work follow `AGENTS.md` and the project's ADRs? |
| `comments` | Do the comments still describe what the code does? |
| `docs` | Has a document that describes this code fallen behind it? |
| `simplify` | Is there dead code, duplicated logic, or nesting to flatten? |
| `local` | Did the change leave the local environment scripts out of date? |
| `security` | Is there injection, an authorization bypass, exposed data, or a secret? |

The lenses run in four groups. When the reviewer is a subagent of your coding agent, each group runs
in its own subagent, in parallel. Another coding agent gets all the lenses in one request.

Documents have their own lenses. Every document is checked for structure and status. A spec is also
checked against its vision, a plan against its spec, and an ADR against every other accepted ADR.

The reviewer reports only what it is sure of, and states the conditions under which each problem
appears. Each finding gets a severity: *blocker*, *important*, or *minor*. If your project has a
`REVIEW.md` at its root, the reviewer follows what it says the project cares about.

## What happens to the findings?

You see them as one numbered list, most severe first, and you mark each one:

| Status | Meaning |
| --- | --- |
| `fixed` | You took it, and the session applies the fix in this run. |
| `dismissed` | You rejected it, with a reason. |
| `deferred` | It is real, but not now. |

Fixes land where the reviewed work lives, wherever you started the review from. If you review pull
request #51 from another branch, the fixes go on #51's branch. A fix to your uncommitted work stays
in your working tree.

When you review an epic's work, the skill asks one question: "I found N problems. Fix them all?"
Each code problem you take becomes a new task in the epic, and the same session then runs
`/codefall-implement` to build them. You type one command.

The skill can also file a bead for a finding you deferred, or for a problem in the plan that a fix
here cannot settle. It offers each one and never files a bead without asking.

When the reviewer could not settle a question, the skill puts it to the project's *consult* agents
once. A consult is a second opinion from the agents your settings list for that purpose. A consult
can turn the question into a finding only after the session has checked what the consult cited.

## Where are the findings recorded?

Each review writes two files under `.codefall/reviews/`: a JSON file for the record and a Markdown
file to read. They record what was reviewed, at which commit, which lenses ran, what could not be
checked, and what you decided about every finding. The files are committed with the fixes. For a
review of uncommitted work, they are left unstaged for you to commit with the work or not at all.

The findings stay in the repository so you can see patterns across reviews. A line in `.ignore`
keeps them out of searches that use ripgrep, which most coding agents do. `codefall init` writes
that line, `codefall doctor` warns when it is missing, and the review skill offers to add it back
before it writes findings.

## Can it post the findings to a pull request?

Yes, once a project turns posting on. It is off by default. To turn it on, run:

```
codefall config review posting on
```

That sets `postToPullRequest` in the `review` block of `.codefall/settings.json`. With posting on,
the findings you marked `fixed` or `deferred` appear on the pull request as inline comments after
triage. Dismissed findings are never posted.
