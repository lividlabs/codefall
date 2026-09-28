# The contract

What the declared `start` and `update` promise. A candidate that does not keep it is reported and
never declared; a draft that would not keep it is not written.

- Both are **shell commands run from the project root**, declared as strings. A Makefile target, a
  package script, or a script of the project's own are all fine.
- **`start`** brings up what the project needs running locally to develop against. Exit `0` when
  everything it manages is up, whether it started it or found it running. Safe to call twice; the
  second call is cheap.
- **`update`** makes the local environment match the checkout: dependencies to the lockfile,
  pending migrations applied, generated code regenerated, whatever a changed definition
  invalidates rebuilt. Exit `0` when the environment matches. Safe to call when nothing changed,
  and cheap then — it asks the package manager and the migration tool, which already answer
  "nothing to do" quickly, rather than doing the work unconditionally.
- **`update` may assume `start` has run.** It does not start anything; `codefall-refresh` runs
  `start` first.
- **Plain scripts.** A person runs either from a terminal. CI can run either. Neither needs the
  extension or a harness. A non-zero exit and a sentence on stderr are how one reports a problem,
  written for the person who reads it next.
- **Never destructive.** No dropping a database, no deleting data directories, no `--force`
  recreation. A command that resets state to get to a known state is not idempotent; it is
  starting over every time.
