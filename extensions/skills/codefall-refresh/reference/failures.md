# Failures, and the sentence to say

What refresh's commands in the primary checkout fail with most often — `git`, `start`, and
`update` — what each means, and what to tell a teammate who does not read stack traces. Read at
step 4 or step 8 when a command exits non-zero. Match on the stderr; say the row's sentence in your
own words, with the one line of stderr that says it.

A row marked **environment** ends with "run `/codefall-refresh` again once that is done". A row
marked **script** names `/codefall-equip`, because the script is wrong and this skill does not edit
it. A row marked **checkout** is the user's to resolve. A row marked **harness** ends with opening a
session in the primary checkout and running `/codefall-refresh` there.

| stderr says | Kind | What it means | What to do |
| --- | --- | --- | --- |
| `Operation not permitted`, `sandbox`, or a permission request rejected, on a path in the primary checkout, from a run started in a worktree | harness | The harness confines the session to the directory it started in — Codex's sandbox, OpenCode's external-directory permission in a non-interactive run | Open a session in the primary checkout and run `/codefall-refresh` there |
| `Cannot connect to the Docker daemon`, `docker: command not found` | environment | Docker is not running, or not installed | Start Docker Desktop, or install it; then refresh again |
| `port is already allocated`, `address already in use` | environment | Something else holds the port a service needs | Find what is on the port (`lsof -i :<port>`) and stop it, or stop a stale container |
| `no configuration file provided`, `compose.yaml: no such file` | script | `start` names a compose file that is not there | Equip: the compose file moved, or the command is wrong |
| `container ... is unhealthy`, `dependency failed to start` | environment | A service came up and failed its healthcheck | `docker compose logs <service>` says why; usually a wrong env var or a full disk |
| `ECONNREFUSED`, `connection refused`, `could not connect to server` | environment | `update` reached for a service that is down | `start` did not bring it up, or it stopped; check `docker compose ps` |
| `password authentication failed`, `FATAL: role ... does not exist` | environment | The database credentials in `.env` do not match the running database | Compare `.env` with `.env.example`; a fresh database needs the user the compose file creates |
| `Migration ... failed`, `migrate: error`, `relation ... already exists` | environment | A migration is half-applied, or the local database has drifted from the migrations | Say which migration; the fix is the migration tool's resolve or repair command, named by the tool |
| `Unable to acquire lock`, `migration lock` | environment | A previous migration run was interrupted and left the lock | The migration tool's unlock command; name it |
| `lockfile ... out of date`, `npm ci can only install ... in sync` | checkout | The lockfile and the manifest disagree in the checkout itself | The branch is broken, not the machine: whoever changed the manifest owes a lockfile update |
| `EACCES`, `permission denied` on a package directory | environment | A dependency directory is owned by another user or root | `sudo chown -R $(whoami)` on the directory named, then refresh again |
| `command not found`, `No such file or directory` on the declared command | script | The declared command's program is not there | Equip: doctor's Local environment section says which; the path moved or the tool is not installed |
| `command not found` on a tool inside the script (`prisma`, `goose`) | environment | The tool the script calls is not installed | Install it; a `mise.toml` or `.tool-versions` says which version, and `mise install` does it |
| `The engine "node" is incompatible`, `requires go 1.xx` | environment | The runtime version is wrong for the checkout | `mise install` when the project pins versions; otherwise install the version the message names |
| `ENOSPC`, `no space left on device` | environment | The disk is full | Free space; Docker images and old `node_modules` are the usual weight |
| `Could not resolve host`, `network is unreachable`, `ETIMEDOUT` | environment | No network, or a registry is unreachable | Check the connection or the VPN; refresh again when it is back |
| `401`, `403`, `authentication required` from a registry | environment | The package registry needs a login | `npm login`, `docker login`, or the registry's own; name the one the URL points at |
| `usage:` or the script's own help text | script | The declared command's arguments are wrong | Equip: the declaration and the script disagree about the subcommand |

A failure with no row is still three things: what failed, quoted in one line; what it means, in
plain words, or "I cannot tell from this" when that is true; and what to do next, even when that
is "send this line to whoever maintains the script".
