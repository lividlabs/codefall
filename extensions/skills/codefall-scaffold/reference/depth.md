# Depth, and the boundary-enforcement obligation

What each depth answer emits. Read at step 5.

## The three depths

**Docs only** (default). The decision layer, nothing else. Correct when dropping the stance into an
existing project, or when the project's shape is not settled enough for code to be anything but
guesswork.

**Docs + project files.** The root scaffolding a project needs before any real code: manifest, build
config, linter **including the boundary rules**, formatter, test runner, and CI that runs all of it.

**Runnable skeleton.** The component folders from the interview, each with its facade and nested
`domain/` `application/` `infrastructure/` `presentation/`, plus one composition root per app that
starts the app end to end with no features in it.

The concrete file list for both code tiers is the profile's business — see its **Depth notes**.
Follow the profile rather than reaching for what you would reflexively pick for the language.

Both code tiers also write the project's local-environment scripts, `start` and `update`, and
declare them under `local` in `.codefall/project.json` when that file exists. The profile's depth
notes say what `update` does for the language. `start` exits `0` with nothing to bring up, unless
the profile's depth notes give it a service to start, and until a task adds one. The procedure is
`codefall-equip`'s.

## The boundary-enforcement obligation

Every profile's boundary-enforcement ADR says the same thing: it is wired **at the project scaffold,
day one**. Docs-only output cannot satisfy that. When emitting docs only, say so plainly in the
report and name it as the first task the user owes the project.

The boundary config is the encoded architecture. Whenever you write it, it enforces all five rules:

1. facade-only imports;
2. inward-only layers;
3. cross-component via facades only;
4. shared modules cannot import components;
5. inner rings cannot touch the UI framework or gateways.

How much tooling that takes is the profile's call, because it depends on the language's visibility
model. Read the profile rather than assuming a language with real visibility needs nothing.
