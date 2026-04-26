# gitt

| [English](./README.md) | [한국어](./README-ko.md) | [日本語](./README-ja.md) |
| --- | --- | --- |

**gitt** is an agent worktree management tool for Docker Compose projects, built on top
of `git worktree`.

Spin up an isolated worktree and bring up its own Docker Compose stack per branch —
even if you're not familiar with Git, GitHub, or Docker.

Especially useful for non-developers who want to contribute with AI coding tools like
**Claude Code** or **Codex**. gitt lets anyone run multiple AI coding agents in parallel
without thinking about worktrees, branches, or compose configuration.

## Commands

| Command | Description |
| --- | --- |
| `gitt on` | Start the daemon (`~/.gitt/gitt.sock`, `~/.gitt/gitt.db`) |
| `gitt off` | Stop the daemon |
| `gitt add <branch>` | Create a worktree at `<repo-parent>/.worktrees/<repo-name>/<branch>`. Checks out the branch if it exists, creates a new one otherwise. `/` and `\` in branch names are converted to `-`. **Requires daemon** |
| `gitt remove <branch>` | Remove the worktree folder for the given branch (`git worktree remove`). **Requires daemon** |
| `gitt status` | Print the current worktree's repository, branch, path, and state (clean/dirty/rebase/merge/conflict, etc.) |
| `gitt sqlite` | Run a SQLite self-test against the daemon's database to confirm the DB connection is healthy. **Requires daemon** |
| `gitt update` | Fetch and install the latest gitt release |
| `gitt version` | Print the installed gitt version |
| `gitt logo` | Print the gitt logo art in a sky-blue box |
| `gitt uninstall` | Stop the daemon → remove `~/.gitt/` → remove the binary. Use `-y`/`--yes` to skip the confirmation prompt |

Daemon-dependent commands fail fast with a `gitt on` hint when the daemon is not
running. (No auto-start.)

## Install

One-line install (macOS / Linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/foreverfl/gitt/main/install.sh | sh
```

- Default install location: `$HOME/.local/bin/gitt` — override with `GITT_INSTALL_DIR`
- Specific version: `GITT_VERSION=v0.0.1 curl ... | sh`
- If `~/.local/bin` is not on your `PATH`, add the line below to your shell rc
  (`~/.zshrc` for zsh, `~/.bashrc` for bash) and reload the shell:

  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```

## Uninstall

```bash
gitt uninstall
```

Stops the daemon → removes `~/.gitt/` (sock, pid, log, db) → removes the binary
(`os.Executable()`). Use `-y` / `--yes` to skip the prompt.
