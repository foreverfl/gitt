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
| `gitt add <branch>` | `<repo-parent>/.worktrees/<repo-name>/<branch>` に worktree を作成。ブランチが存在すればチェックアウト、なければ新規作成。ブランチ名の `/`・`\` は `-` に変換される。**デーモン必須** |
| `gitt remove <branch>` | 指定ブランチの worktree フォルダを削除 (`git worktree remove`)。**デーモン必須** |
| `gitt status` | 現在の worktree のリポジトリ、ブランチ、パス、状態 (clean/dirty/rebase/merge/conflict など) を出力 |
| `gitt sqlite` | デーモンの DB 接続が正常かを確認する SQLite セルフテストを実行。**デーモン必須** |
| `gitt update` | 最新リリースを取得してインストール |
| `gitt version` | インストール済みの gitt バージョンを表示 |
| `gitt logo` | gitt のロゴアートを水色のボックスで表示 |
| `gitt uninstall` | デーモン停止 → `~/.gitt/` 削除 → バイナリ削除。`-y`/`--yes` で確認プロンプトをスキップ |

デーモンが必須のコマンドはデーモンが起動していない場合、即座にエラーで停止し
`gitt on` を案内する。(auto-start はしない)

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
