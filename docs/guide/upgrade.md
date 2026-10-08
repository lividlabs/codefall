# /codefall-upgrade

`/codefall-upgrade` brings a project's Codefall install and its architecture documents up to date
after a new release. It reports every difference between your documents and the templates Codefall
ships now, and it changes only the items you choose. Run it after you install a newer `codefall`, or
when you want to adopt Codefall's architecture in a project that never used it.

```
/codefall-upgrade
/codefall-upgrade adopt
```

Codefall's skills are instructions your coding agent follows when you type a slash command; the
[README](../../README.md) introduces them.

## How is it different from `codefall upgrade`?

The two cover different files.

- **`codefall upgrade`**, the command-line tool, reinstalls the skills, shared files, and hooks.
  [Setting up a project](setup.md) describes it.
- **`/codefall-upgrade`**, the skill, offers to run that command first and then works on the
  project's documents.

The skill runs the command only when you say yes. It first shows you the breaking changes between
your installed version and the new one. It never runs the command with `--yes`. When `codefall` is
not on your `PATH`, the skill says so and goes on to the documents, which need no binary. The files
the command changes are left uncommitted for you to commit.

You start `/codefall-upgrade` yourself. An agent never runs it on its own, and no other skill starts
it, because it changes documents a project depends on.

## Which documents does it update?

When `/codefall-scaffold` starts a project, it copies Codefall's architecture decisions into the
project as ADRs (architecture decision records). These are the *inherited ADRs*, numbered
`ADR-BASE-01`, `ADR-TS-01`, `ADR-GO-01`, and so on. The [architecture guide](architecture.md)
explains them. Codefall's templates change over time, and the skill compares your copies with the
current ones.

| In scope | Out of scope |
| --- | --- |
| Inherited ADRs: new ones, revised ones, and renamed ones | Moving code to a different architecture |
| `docs/adrs/_TEMPLATE.md` | Extracting a component into a separate service |
| Changes to the `AGENTS.md` skeleton, reported for you to merge | Changing component boundaries |
| `.codefall/scaffold.json` | Source code, build configuration, and lint rules |
| One line in `docs/decision-log.md` recording the run | The project's own `ADR-NNN` decisions |
| A `CUSTOMIZE.md` section for plans under a renamed skill's old directory | Moving or renaming those plans |

The skill moves documents and never changes code. If you ask it to move code between architectures
or pull a component out into a service, it says that work belongs to a future `migrate` skill and
stops.

## How does it know what you changed?

`.codefall/scaffold.json` records a hash of each inherited ADR as it was written, and whether you
amended it during scaffolding. The skill recomputes each hash and puts every ADR in one of four
states:

| State | Meaning | What the skill may do |
| --- | --- | --- |
| untouched | The hash matches, and you did not amend it | Supersede or rename it, if you take the item |
| amended | You changed it during the scaffold interview | Show you the diff and leave it alone |
| edited | The hash no longer matches | Show you the diff and leave it alone |
| unverifiable | No record and no old template to compare with | Treat it as edited |

Only an untouched ADR is ever updated automatically. An amended or edited ADR holds a decision the
project changed on purpose, and a new ADR built from the template would drop that change. For those,
the diff is what the skill gives you. Writing the ADR that carries your change forward is yours to
do. If you have seen the diff and want the template's version anyway, you can name the file and say
so, and the skill tells you that your amendment will stop applying.

Projects scaffolded before Codefall 0.4.0 have no `scaffold.json`. For those, the skill looks for the
old templates to compare against, and offers to write `scaffold.json` so later runs do not need to.

## What does it report?

For each difference, the report names the item, its state, what changed, and whether the skill can
apply it or you have to merge it by hand. The differences fall into these kinds:

- **Missing:** a template the project should have and does not.
- **Revised:** the project's copy differs from the current template.
- **Renamed:** the project has the template under an older identifier. A project from 0.2.x has
  `ADR-003-dependency-injection.md`, which is now `ADR-TS-01`. The report lists every document that
  still cites the old identifier.
- **Retired:** the project has a document whose template no longer ships. The skill leaves it alone.
- **Skeleton drift:** the `AGENTS.md` skeleton changed. Every project fills its `AGENTS.md` in by
  hand, so these are reported as advice and never applied wholesale.
- **Provenance:** `scaffold.json` is missing or out of date.
- **Legacy documents:** the project has plans under `docs/designs/`, from before `codefall-design`
  was renamed `codefall-plan`.

When Codefall's changelog is reachable, the report also says why each template changed, grouped by
release. The comparison, not the changelog, decides what changed.

Then the skill stops. Nothing is applied until you name the items you want. "Everything safe" is a
valid answer.

## What happens when you take an item?

The skill works on a branch named `upgrade/<date>` and applies one item at a time, so each change
can be reviewed as its own diff. It needs a clean working tree before it writes anything, or your
explicit go-ahead.

It never rewrites an ADR the project has ratified. A revised template lands as a new ADR that
supersedes the old one. The old ADR stays in place, and the only change to it is its Status line,
which becomes `Superseded by <id> — <date>`. When the name did not change, the new ADR gets an
edition number, such as `ADR-BASE-02.2-package-by-component.md`.

After each item, the skill updates `scaffold.json` with the new file's hash. At the end, it adds one
line to `docs/decision-log.md`, checks that every link and hash holds, commits what you took, pushes
the branch, and opens a pull request. A person merges it.

A partial run is fine. The hashes record what is current, so the next run picks up where this one
left off.

## What if you don't want a template?

You can decline a missing template. The skill records the decline in `scaffold.json` and does not
offer that template again. To see the declined templates again, run `/codefall-upgrade adopt`.
Answering "not now" records nothing.

## Can it add Codefall's architecture to an existing project?

Yes. In a project that was never scaffolded, every template is missing, and the same report-and-take
process installs Codefall's set of architecture decisions, which Codefall calls *the stance*. The skill
detects the project's languages from files such as `package.json` and `go.mod`, and asks you to
confirm them. If a language has no supported profile, it says so and stops, as
`/codefall-scaffold` does.

Adoption installs documents that describe an architecture your code may not follow yet. The skill
does not check the code against them, and its report says so. Bringing the code in line is a
separate job, and so is the boundary enforcement the documents call for. The
[architecture guide](architecture.md) describes both.
