# gitt

| [English](./README.md) | [한국어](./README-ko.md) | [日本語](./README-ja.md) |
| --- | --- | --- |

**gitt** は `git worktree` をベースにした、Docker Compose プロジェクト向けの
エージェント worktree 管理ツール。

Git, GitHub, Docker に詳しくなくても、ブランチごとに隔離された worktree を作成し、
それぞれ専用の Docker Compose スタックを立ち上げられる。

**Claude Code** や **Codex** のような AI コーディングツールで開発に参加したい
非開発者にとって特に便利。worktree、ブランチ、compose の設定を意識せずに、
複数の AI コーディングエージェントを並列で動かせる。

## Commands

| コマンド | 動作 |
| --- | --- |
| `gitt on` | デーモン起動 (`~/.gitt/gitt.sock`, `~/.gitt/gitt.db`) |
| `gitt off` | デーモン停止 |
| `gitt add <branch>` | `<repo>/.worktrees/<branch>` に worktree を作成。ブランチが存在すればチェックアウト、なければ新規作成。ブランチ名の `/`・`\` は `-` に変換される。対象ブランチがすでにどこかにチェックアウト済み（例: リポジトリルートの `main`）の場合は、新規作成せず既存パスを通知してデーモンに登録する。**デーモン必須** |
| `gitt remove <branch>` | 指定ブランチの worktree フォルダを削除 (`git worktree remove`)。**デーモン必須** |
| `gitt rename <old> <new>` | ブランチと worktree フォルダを同時にリネーム。`<repo>/.worktrees/<old>` → `<repo>/.worktrees/<new>` への移動、ブランチ名の変更、デーモンレコードの更新を一括で実行。**デーモン必須** |
| `gitt status` | 現在の worktree のリポジトリ、ブランチ、パス、状態 (clean/dirty/rebase/merge/conflict など) を出力 |
| `gitt vscode` | `<repo-root>/<repo-name>.code-workspace` を作成（または更新）。登録済み worktree ごとにブランチ名でラベル付けされたフォルダーエントリを追加。既存の `settings`/`extensions` は保持される。すべての worktree が `.worktrees/` 配下にあり VSCode のタイトルで区別しにくい場合に便利。bare レイアウト内のどこからでも実行可能（メインルートまたは任意の worktree） — 出力ファイルは常にメインリポジトリのルートに生成される。その後 `code <repo-root>/<repo-name>.code-workspace` で開く（例: `code gitt/gitt.code-workspace`）。`.worktrees/` を直接開くのはこのコマンドが解決しようとしている状況そのものなので避けること。**デーモン必須** |
| `gitt sqlite` | デーモンの DB 接続が正常かを確認する SQLite セルフテストを実行。**デーモン必須** |
| `gitt config` | `~/.gitt/config.toml` をエディタで開く。初回実行時はデフォルト値でファイルを作成し、以降は既存ファイルをそのまま開く。エディタの解決順: `$VISUAL` → `$EDITOR` → `vi`。`code --wait` のような値も動作 — コマンドを空白で分割して実行する。デーモン不要。 |
| `gitt update` | 最新リリースを取得してインストール。デーモン停止 → 登録済み worktree フォルダを強制削除（未コミット・untracked の変更は復元不可）→ 各リポジトリで `git worktree prune` → `~/.gitt/` 削除 → バイナリ差し替え。`-y`/`--yes` で確認プロンプトをスキップ |
| `gitt version` | インストール済みの gitt バージョンを表示 |
| `gitt logo` | gitt のロゴアートを水色のボックスで表示 |
| `gitt uninstall` | デーモン停止 → `~/.gitt/` 削除 → バイナリ削除。`-y`/`--yes` で確認プロンプトをスキップ |

デーモンが必須のコマンドはデーモンが起動していない場合、即座にエラーで停止し
`gitt on` を案内する。(auto-start はしない)

## Configuration

`gitt config` は `~/.gitt/config.toml` をエディタで開く。初回実行時はデフォルト値でファイルを作成する。スキーマ:

```toml
[worktree]
copy    = [".env", ".env.local", ".env.development", ".envrc", ".npmrc", ".nvmrc"]
symlink = ["node_modules", ".venv"]
ignore  = ["dist", "build", ".next", ".cache", "target"]
```

- `copy` — 新しい worktree にコピーされるファイル（env・シークレット等、共有しないもの）。
- `symlink` — 新しい worktree にシンボリックリンクされるパス（重複を避けたい重いディレクトリ）。
- `ignore` — 新しい worktree のシード時にスキップするパス（ビルド成果物、キャッシュ等）。

> **注意:** このリリースでは config インフラと編集コマンドのみを含む。`copy`、`symlink`、`ignore` の
> リストはまだ `gitt add` や `gitt clone` では使われていない。スキーマが先行着地し、
> コマンドへの組み込みは次のステップで行われる予定。

**エディタのヒント:** シェル rc に `export EDITOR="code --wait"` を追加すると VS Code が使える。
`--wait` フラグは必須 — ファイルを閉じるまでエディタがブロックする必要がある。

## Install

ワンラインインストール (macOS / Linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/foreverfl/gitt/main/install.sh | sh
```

- デフォルトのインストール先: `$HOME/.local/bin/gitt` — `GITT_INSTALL_DIR` で上書き可能
- 特定バージョン: `GITT_VERSION=v0.0.1 curl ... | sh`
- `~/.local/bin` が `PATH` に含まれていない場合は、シェル rc (zsh は `~/.zshrc`、
  bash は `~/.bashrc`) に以下の行を追加してシェルを再読み込み:

  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```

## Uninstall

```bash
gitt uninstall
```

デーモン停止 → `~/.gitt/` (sock, pid, log, db) 削除 → バイナリ (`os.Executable()`) 削除。
プロンプトなしで実行するには `-y` / `--yes`。
