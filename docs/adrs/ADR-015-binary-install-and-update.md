# ADR-015: Binary Install and Update

## Status

Accepted — 2026-10-07

## Context

The `codefall` binary reaches a machine four ways. The install script, `curl -fsSL
https://install.codefall.dev/sh | sh`, downloads a release archive, checks it against the release's
`checksums.txt`, and puts the binary in `~/.local/bin` or `CODEFALL_INSTALL_DIR`. mise installs it as
`packslip:github.com/lividlabs/codefall`, one directory per version under its `installs/` directory,
and chooses the version from a `mise.toml`, often one a project commits. `go install` builds it from
source into `GOBIN` or `GOPATH/bin`. A person can also download an archive from the GitHub releases
page and put the binary wherever they like.

Nothing updates the binary except repeating whichever of those put it there. A `codefall update`
command is wanted, and it has to answer a question first: how was the binary it would replace
installed? Only one of the four leaves a binary that codefall may replace. mise records which version
lives in which directory, so overwriting the binary in its `0.26.0` directory with another version
makes that record false, and it overrides the version a project pinned for everyone working in it. A
`go install` binary is built from source and belongs to the Go toolchain. A binary a person copied by
hand is wherever they put it, for reasons codefall does not know.

**Which binary runs depends on the directory.** mise puts a project's tools on `PATH` when the shell
enters the project, so the same machine can run a mise-installed `codefall` in one directory and a
script-installed one everywhere else. The binary `update` has to reason about is the one that is
running, found by `os.Executable`, not one recorded somewhere else.

**A release binary does not know how it was installed.** The script, mise, and a manual download all
deliver the same GoReleaser build, so nothing stamped into it can tell them apart. Only where it sits
on disk can, and a script install has no distinctive place: `~/.local/bin`, or any directory
`CODEFALL_INSTALL_DIR` names, looks the same as a directory someone copied a binary into.

Four alternatives were weighed.

**Replace whatever binary is running.** The simplest command, and wrong for every install method but
the script, for the reasons above.

**Treat anything that is not mise and not `go install` as a script install.** No receipt is needed,
and a binary someone copied by hand, or one a tool codefall does not know about manages, is replaced
on a guess.

**Start from the receipt and replace the binary it names.** `update` would then replace the
script-installed binary while a mise-installed one runs in the current directory, and the person
would see no change in the place they ran the command.

**Have `update` run mise.** `mise upgrade --bump` rewrites the version in the mise config, which may
be a file a whole team shares. Changing it is the person's decision, and `update` names the command
instead of running it.

## Decision

### The install script writes a receipt

After the binary is in place, `install.sh` writes `${XDG_STATE_HOME:-~/.local/state}/codefall/install.json`,
on macOS as on Linux:

```json
{
  "path": "/Users/someone/.local/bin/codefall",
  "version": "0.29.0"
}
```

`path` is absolute, with symbolic links in its directory resolved, so it compares equal to the
resolved `os.Executable`. `version` is the release the script installed. The file is written to a
temporary name and renamed into place. A receipt that cannot be written leaves the installed binary
where it is and says so; it does not fail the install. A reader ignores fields it does not know, so a
field can be added without changing this record.

There is one receipt per user. A second script install to another directory replaces it, and the
binary the first one installed is then recognised as unknown.

### The install script puts the directory on `PATH`

`~/.local/bin` is not on `PATH` on a default macOS account, so a script install that stops at
copying the binary leaves `codefall` unrunnable by name. The script checks `PATH` after the copy,
and when the install directory is missing from it, appends an export line to the startup file of
the shell `$SHELL` names: `.zshrc` under `ZDOTDIR` or the home directory for zsh, `.bashrc` for bash
on Linux and `.bash_profile` for bash on macOS, where the terminal starts a login shell, and
`config.fish` for fish. The directory is written as `$HOME/...` when it lies under the home
directory. It then says what it changed and prints the `source` line that applies it to the current
shell. This is what rustup, uv, and bun do.

It does not edit when the directory is already on `PATH`, when the startup file already names the
directory (it says to open a new terminal instead), when `CODEFALL_NO_MODIFY_PATH=1` is set, or when
`$SHELL` is a shell it does not know; in the last two cases it prints the line to add. A startup
file it cannot write is reported the same way. Nothing in this step fails the install.

Two alternatives were not taken. Printing the exact line for the person to run, as mise and
Homebrew do, is one step fewer for the script and one more for every new user, and the step is the
one that decides whether the first `codefall` command works. Asking at the prompt, as deno does,
needs `/dev/tty` because the script's stdin is the pipe from `curl`, and that is the part most
likely to misbehave in containers and CI.

### `codefall update` finds out how the running binary was installed before it does anything

Decided here, landed in the next pull request. It starts from `os.Executable`, with symbolic links
resolved, and checks in this order:

1. **Native:** a receipt exists and its `path` is this binary.
2. **mise:** the path is under `$MISE_DATA_DIR/installs/` when that is set, and otherwise has a
   `/mise/installs/` segment.
3. **`go install` or checkout build:** the binary carries no release version, which `buildinfo`
   already reports as a development build.
4. **Unknown:** none of the above.

A missing receipt does not stop the checks: the binary is running from somewhere, and the other
checks still say what they can.

### What `update` does for each

- **Native:** downloads the release `latest` names, or the version given as an argument, checks it
  against the release's `checksums.txt` as the script does, writes the new binary beside the old one
  under a temporary name, renames it over the old one, and writes the new version into the receipt. A
  directory it cannot write to fails the command, printing the install script's line; it never asks
  for elevated privileges.
- **mise:** changes nothing. It says this `codefall` was installed by mise, with its path and version,
  and prints `mise upgrade packslip:github.com/lividlabs/codefall`, which stays inside the version the
  mise config allows, and `mise upgrade --bump packslip:github.com/lividlabs/codefall`, which moves to
  the latest release and rewrites the version in that config.
- **`go install` or checkout build:** changes nothing, and prints
  `go install github.com/lividlabs/codefall/cli/cmd/codefall@latest`.
- **Unknown:** changes nothing, and prints the path it found and the install script's line, which
  installs natively and writes a receipt.

A native binary already at the target version is reported "already up to date". That and a completed
native update exit zero; every other case exits non-zero. `--check` runs the same checks, reports the
install method, the path, the installed version, and the latest release, changes nothing, and exits
zero.

### The command is `update`, beside `upgrade`

`codefall upgrade` brings a project's install level with the binary (ADR-010). `codefall update`
brings the binary level with the latest release. A person runs them in that order, `update` and then
`upgrade`, and `update`'s help names `upgrade` for a project. The command lives in its own
component, `internal/update/`: it shares no step with init's use case, and its download and file
replacement are not pure.

## Consequences

- **Only a script install updates itself.** A mise or `go install` binary is updated by the tool
  that installed it, and `update` names that tool's command rather than reaching around it. A
  project's mise pin stays the project's decision.
- **A script install from before the receipt existed is unknown.** It is told to run the install
  script once, which writes the receipt, and `update` works on it after that.
- **The receipt's location and fields are a contract between the script and the binary.** Moving the
  file or renaming a field means both sides read the old shape for as long as receipts written by an
  older script exist, which is why this record fixes them.
- **The mise check reads a path layout codefall does not own.** A mise release that moves its
  `installs/` directory turns a mise install into an unknown one, which changes nothing and prints the
  install script's line — the safe failure, and a wrong suggestion until the check is updated.
- **The script writes to a file it does not own.** A startup file gets one commented export line,
  appended, never rewritten, and only when it does not already name the directory. A person who
  wants no such edit sets `CODEFALL_NO_MODIFY_PATH=1` and is given the line to add.
- **`update` and `upgrade` differ by one word.** The help text of each names the other, and the
  order a person runs them in matches what each one brings current.

## Related

- ADR-010, *Upgrade* — the command that brings a project's install level with the binary, which a
  person runs after `update`.
- `scripts/install.sh` — the install script, which writes the receipt.
- `cli/internal/shared/buildinfo/` — how a binary knows whether it is a release, which the third
  check reads.
- `docs/decision-log.md`, *Binary install receipt and `codefall update`, 2026-10-07* — the
  alternatives seen and not taken, in brief.
