# Installing codefall

`codefall` is a single binary. You can install it with a script, with mise, or by downloading a
release yourself. The [README](../../README.md#prerequisites) lists the other tools Codefall needs.

## Install with the script

This command installs the latest release:

```sh
curl -fsSL https://install.codefall.dev/sh | sh
```

The script does four things:

1. It downloads the release archive for your platform, along with the release's `checksums.txt`.
2. It verifies the archive against `checksums.txt` and stops if the two do not match.
3. It installs the binary to `~/.local/bin`.
4. It writes a record of the binary's path and version to `~/.local/state/codefall/install.json`, or
   to `$XDG_STATE_HOME/codefall/install.json` when `XDG_STATE_HOME` is set. `codefall update` reads
   this record to know that the script installed the binary.

The script's source is [`scripts/install.sh`](../../scripts/install.sh).

### Install a specific version

Pass the version after `sh -s --`:

```sh
curl -fsSL https://install.codefall.dev/sh | sh -s -- 0.28.0
```

### Install somewhere other than `~/.local/bin`

Set `CODEFALL_INSTALL_DIR` to the directory you want:

```sh
curl -fsSL https://install.codefall.dev/sh | CODEFALL_INSTALL_DIR="$HOME/bin" sh
```

### How the script changes your `PATH`

When the install directory is not on your `PATH`, the script adds a line to your shell's startup file
and tells you to open a new terminal. It picks the file from your login shell:

| Shell | Startup file |
| --- | --- |
| zsh | `.zshrc` |
| bash on Linux | `.bashrc` |
| bash on macOS | `.bash_profile`, because Terminal and iTerm start bash as a login shell |
| fish | `config.fish` |

To edit the file yourself, set `CODEFALL_NO_MODIFY_PATH=1`. The script then prints the line to add
and leaves the file alone.

## Update codefall

`codefall update` replaces a binary the script installed with the latest release:

```sh
codefall update           # the latest release
codefall update 0.30.0    # a specific version
codefall update --check   # report only
```

`--check` reports how the running binary was installed and what the latest release is, and changes
nothing.

When mise or `go install` manages the binary, `update` changes nothing either. It prints the command
that updates the binary through that tool instead.

## Install with mise

[mise](https://mise.jdx.dev/) installs and manages development tools. It can install `codefall` for
one project or for your whole machine:

```sh
mise use packslip:github.com/lividlabs/codefall      # the current project
mise use -g packslip:github.com/lividlabs/codefall   # every project (global)
```

The repository used to be named `lividlabs/codefall-cli`. A mise configuration that still names
`packslip:github.com/lividlabs/codefall-cli` keeps working on mise 2026.9.16 or later, because those
versions follow a renamed repository by its ID. Earlier versions of mise need the new name.

## Download a release

Every [release](https://github.com/lividlabs/codefall/releases) has prebuilt binaries for macOS and
Linux, on Intel and ARM. Download the archive for your platform, extract `codefall`, and put it in a
directory on your `PATH`. `codefall update` does not manage a binary installed this way.
