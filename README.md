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
| `gitt add <branch>` | Create a worktree at `<repo>/.worktrees/<branch>`. Checks out the branch if it exists, creates a new one otherwise. `/` and `\` in branch names are converted to `-`. If the branch is already checked out somewhere (e.g. `main` in the repo root), the existing path is reported and registered with the daemon — no new worktree is created. **Requires daemon** |
| `gitt remove <branch>` | Remove the worktree folder for the given branch (`git worktree remove`). **Requires daemon** |
| `gitt rename <old> <new>` | Rename a branch and its worktree folder together. Updates `<repo>/.worktrees/<old>` → `<repo>/.worktrees/<new>`, renames the branch, and updates the daemon record in one step. **Requires daemon** |
| `gitt status` | Print the current worktree's repository, branch, path, and state (clean/dirty/rebase/merge/conflict, etc.) |
| `gitt vscode` | Write (or update) `<repo-root>/<repo-name>.code-workspace` with one folder entry per registered worktree, labeled by branch name. Preserves existing `settings`/`extensions`. Useful because every worktree lives under `.worktrees/` and otherwise shows an identical title in VSCode. Can be run from anywhere inside the bare layout (main root or any worktree) — the output always lands at the main repo root. Then open it with `code <repo-root>/<repo-name>.code-workspace` (e.g. `code gitt/gitt.code-workspace`); opening `.worktrees/` directly is exactly the situation this command exists to avoid. **Requires daemon** |
| `gitt sqlite` | Run a SQLite self-test against the daemon's database to confirm the DB connection is healthy. **Requires daemon** |
| `gitt config` | Open `~/.gitt/config.toml` in your editor. Creates the file from built-in defaults on first run; otherwise opens the existing file. Editor is resolved in order: `$VISUAL` → `$EDITOR` → `vi`. Values like `code --wait` work — the command is split on whitespace. Does not require the daemon. |
| `gitt update` | Fetch the latest release and install it. Shuts down the daemon, force-deletes all registered worktree folders (uncommitted and untracked changes are unrecoverable), runs `git worktree prune` on each repo, then removes `~/.gitt/` and replaces the binary. Use `-y`/`--yes` to skip the prompt |
| `gitt version` | Print the installed gitt version |
| `gitt logo` | Print the gitt logo art in a sky-blue box |
| `gitt uninstall` | Stop the daemon → remove `~/.gitt/` → remove the binary. Use `-y`/`--yes` to skip the confirmation prompt |

Daemon-dependent commands fail fast with a `gitt on` hint when the daemon is not
running. (No auto-start.)

## Configuration

`gitt config` opens `~/.gitt/config.toml` in your editor. The file is created from
built-in defaults on first run. Schema:

```toml
[worktree]
copy    = [".env", ".env.local", ".env.development", ".envrc", ".npmrc", ".nvmrc"]
symlink = ["node_modules", ".venv"]
ignore  = ["dist", "build", ".next", ".cache", "target"]
```

- `copy` — files copied into each new worktree (env/secret files that should not be shared).
- `symlink` — paths symlinked into each new worktree (heavy directories you do not want duplicated).
- `ignore` — paths skipped when seeding a new worktree (build outputs, caches).

> **Note:** The config infrastructure and the edit command ship in this release. The
> `copy`, `symlink`, and `ignore` lists are not yet consumed by `gitt add` or `gitt clone`.
> Think of this as the schema landing first — wiring it into commands comes next.

**Editor tip:** Add `export EDITOR="code --wait"` to your shell rc to use VS Code.
The `--wait` flag is required so the editor blocks until you close the file.

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
