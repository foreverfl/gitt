-- repos holds per-repo metadata shared across every worktree of the same
-- repository: filesystem layout (root/bare/worktrees paths), the default
-- branch, and the project shape (language/framework/compose_monorepo) that
-- drives port allocation when a worktree is created. One row per repository,
-- keyed by root_path so callers can look it up from the user's cwd without
-- consulting git.
--
-- Port allocation forks on compose_monorepo: when 1, gitt parses the
-- repo's compose file and assigns one host_port per declared service; when
-- 0, gitt assigns a single host_port using a (language, framework) default
-- (e.g. node+next → 3000, python+django → 8000). framework is optional —
-- Go CLIs and similar plain-language projects leave it as the empty string.
CREATE TABLE IF NOT EXISTS repos (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  root_path         TEXT    NOT NULL UNIQUE,
  bare_path         TEXT    NOT NULL,
  worktrees_path    TEXT    NOT NULL,
  default_branch    TEXT    NOT NULL,
  language          TEXT    NOT NULL,
  framework         TEXT    NOT NULL DEFAULT '',
  compose_monorepo  INTEGER NOT NULL DEFAULT 0,
  created_at        TEXT    NOT NULL,
  updated_at        TEXT    NOT NULL
);

-- worktrees is one row per checked-out branch under a repo, linked to its
-- parent via repo_id. ON DELETE CASCADE makes "drop a repo" remove every
-- worktree underneath it, and the same chain through ports cascades down
-- to the per-worktree port rows. Callers that only have a filesystem path
-- (e.g. `gitt add` running from cwd) join through repos.root_path to find
-- the right repo_id rather than denormalising the path back into this row.
CREATE TABLE IF NOT EXISTS worktrees (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,

  repo_id          INTEGER NOT NULL,
  branch_name      TEXT NOT NULL,
  safe_branch_name TEXT NOT NULL,
  worktree_path    TEXT NOT NULL,

  status           TEXT NOT NULL DEFAULT 'created',
  is_protected     INTEGER NOT NULL DEFAULT 0,

  created_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE,
  UNIQUE (repo_id, branch_name),
  UNIQUE (worktree_path)
);

-- ports records each host:container port mapping a worktree currently owns.
-- A 1:N child of worktrees so the chain reads repos → worktrees → ports:
-- a monorepo worktree gets multiple rows (e.g. web, api, db), a single-
-- service worktree gets one. host_port is globally UNIQUE so two worktrees
-- never collide on the same host port; UNIQUE(worktree_id, name) keeps a
-- single worktree from declaring the same named service twice. ON DELETE
-- CASCADE through worktree_id means dropping a worktree releases its ports.
CREATE TABLE IF NOT EXISTS ports (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  worktree_id     INTEGER NOT NULL,
  name            TEXT NOT NULL,
  host_port       INTEGER NOT NULL UNIQUE,
  container_port  INTEGER NOT NULL,
  protocol        TEXT NOT NULL DEFAULT 'tcp',
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,

  FOREIGN KEY (worktree_id) REFERENCES worktrees(id) ON DELETE CASCADE,
  UNIQUE (worktree_id, name)
);
