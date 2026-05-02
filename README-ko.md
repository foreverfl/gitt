# gitt

| [English](./README.md) | [한국어](./README-ko.md) | [日本語](./README-ja.md) |
| --- | --- | --- |

**gitt** 는 `git worktree` 기반의 Docker Compose 프로젝트용 에이전트 워크트리 관리 툴.

Git, GitHub, Docker 에 익숙하지 않아도 브랜치별로 격리된 워크트리를 만들고
그에 맞는 Docker Compose 스택을 함께 띄울 수 있음.

**Claude Code**, **Codex** 같은 AI 코딩 도구로 개발에 기여하고 싶은 비개발자에게
특히 유용함. 워크트리, 브랜치, compose 설정을 신경 쓰지 않고 여러 AI 코딩
에이전트를 병렬로 돌릴 수 있음.

## Commands

| 명령어 | 동작 |
| --- | --- |
| `gitt on` | 데몬 기동 (`~/.gitt/gitt.sock`, `~/.gitt/gitt.db`). `~/.gitt/config.toml` 에 `[ui] logo_enabled = true` 일 때만 박스형 로고 배너 출력 (기본값: 비활성). `gitt logo` 로 토글 가능. |
| `gitt off` | 데몬 종료 |
| `gitt add <branch>` | `<repo>/.worktrees/<branch>` 에 worktree 생성. 브랜치가 존재하면 체크아웃, 없으면 신규 생성. 브랜치명의 `/`·`\`는 `-`로 변환됨. 해당 브랜치가 이미 어딘가에 체크아웃되어 있으면 (예: 레포 루트의 `main`) 새로 만들지 않고 기존 경로를 알려주며 데몬에 등록함. `--print-path` 를 붙이면 사람이 읽는 출력은 stderr로 보내고 stdout에는 worktree 절대 경로만 한 줄 출력 (셸 래퍼용). **데몬 필요** |
| `gitt remove <branch>` | 해당 브랜치의 worktree 폴더 삭제 (`git worktree remove`). **데몬 필요** |
| `gitt rename <old> <new>` | 브랜치와 worktree 폴더를 한 번에 rename. `<repo>/.worktrees/<old>` → `<repo>/.worktrees/<new>` 이동 + 브랜치 rename + 데몬 레코드 갱신을 묶어서 처리. **데몬 필요** |
| `gitt status` | 현재 worktree의 저장소, 브랜치, 경로, 상태 (clean/dirty/rebase/merge/conflict 등) 출력 |
| `gitt vscode` | `<repo-root>/<repo-name>.code-workspace` 를 생성(또는 갱신). 등록된 각 worktree를 브랜치명으로 레이블된 폴더 항목으로 추가. 기존 `settings`/`extensions` 는 그대로 유지. 신규 생성 시에만 `terminal.integrated.environmentChangesIndicator: "off"` 와 `terminal.integrated.environmentChangesRelaunch: false` 를 `settings` 에 한 번 주입 (folder 목록 변경 때마다 terminal env에 contribute하는 extension이 띄우는 relaunch 경고 억제). 이후에는 gitt이 `settings` 를 절대 건드리지 않으므로 사용자가 값을 바꾸거나 키를 지워도 그대로 유지됨. 모든 worktree가 `.worktrees/` 아래에 있어 VSCode 타이틀이 구분이 안 될 때 유용. bare 레이아웃 안 어디서든(메인 루트 또는 임의의 worktree) 실행 가능 — 결과 파일은 항상 메인 저장소 루트에 생성됨. 이후 `code <repo-root>/<repo-name>.code-workspace` 로 열기 (예: `code gitt/gitt.code-workspace`). `.worktrees/` 를 직접 여는 것은 이 명령이 해결하려는 바로 그 상황이므로 권장하지 않음. **데몬 필요** |
| `gitt sqlite` | 데몬의 DB 연결이 정상인지 확인하는 SQLite self-test 실행. **데몬 필요** |
| `gitt config` | `~/.gitt/config.toml` 을 에디터로 열기. 최초 실행 시 기본값으로 파일 생성, 이후에는 기존 파일을 그대로 열기. 에디터 탐색 순서: `$VISUAL` → `$EDITOR` → `vi`. `code --wait` 같은 값도 동작 — 명령어를 공백으로 분리해서 실행함. 데몬 불필요. |
| `gitt update` | 최신 릴리스를 받아 설치. 데몬 종료 → 등록된 워크트리 폴더 강제 삭제(미커밋·untracked 변경은 복구 불가) → 각 레포에서 `git worktree prune` → `~/.gitt/` 삭제 → 바이너리 교체. `-y`/`--yes` 로 확인 프롬프트 생략 |
| `gitt version` | 설치된 gitt 버전 출력 |
| `gitt logo` | gitt 로고 아트를 출력하고, `gitt on` 시작 시 로고를 보여줄지 인터랙티브하게 토글. 선택값을 `~/.gitt/config.toml` 의 `[ui] logo_enabled` 에 저장. 인터랙티브 터미널(stdin이 TTY) 필요. |
| `gitt uninstall` | 데몬 종료 → `~/.gitt/` 삭제 → 바이너리 삭제. `-y`/`--yes` 로 확인 프롬프트 생략 |

데몬이 필요한 명령은 데몬이 떠있지 않으면 즉시 에러로 끊고 `gitt on` 안내.
(auto-start 안 함)

## Configuration

`gitt config` 는 `~/.gitt/config.toml` 을 에디터로 열어줌. 최초 실행 시 기본값으로 파일 생성. 스키마:

```toml
[ui]
# Whether `gitt on` prints the boxed logo banner on startup.
# Toggle this with `gitt logo`.
logo_enabled = false

[worktree]
copy    = [".env", ".env.local", ".env.development", ".envrc", ".npmrc", ".nvmrc"]
symlink = ["node_modules", ".venv"]
ignore  = ["dist", "build", ".next", ".cache", "target"]
```

- `logo_enabled` — `gitt on` 시작 시 박스형 로고 배너 표시 여부. 파일을 직접 편집하는 것보다 `gitt logo` 로 토글하는 게 편함.
- `copy` — 새 worktree에 복사되는 파일 (env/시크릿 파일 등 공유하지 않을 것들).
- `symlink` — 새 worktree에 심링크되는 경로 (중복 보관을 원치 않는 무거운 디렉토리).
- `ignore` — 새 worktree 세팅 시 건너뛸 경로 (빌드 결과물, 캐시 등).

> **참고:** 이번 릴리스에서는 config 인프라와 편집 명령만 포함됨. `copy`, `symlink`, `ignore` 목록은
> 아직 `gitt add` 또는 `gitt clone` 에서 실제로 사용되지 않음. 스키마가 먼저 착지하고,
> 명령어 연동은 다음 단계에서 진행됨.

**에디터 팁:** 셸 rc에 `export EDITOR="code --wait"` 를 추가하면 VS Code 사용 가능.
`--wait` 플래그가 필수 — 파일을 닫을 때까지 에디터가 블로킹되어야 함.

## 셸 래퍼 (선택)

바이너리는 부모 셸의 디렉토리를 바꿀 수 없음. 래퍼 없이 `gitt add` 를 실행하면
"새 터미널을 열고 cd ... 를 실행하세요" 안내가 출력되고 직접 복사-붙여넣기해야 함.
아래 래퍼를 사용하면 `gitt add` 를 가로채 `--print-path` 로 worktree 경로를 받아
자동으로 이동해 줌.

`~/.zshrc` 또는 `~/.bashrc` 에 붙여넣은 뒤 `source` 하거나 새 터미널을 열기:

```sh
# gitt — auto-cd into the new worktree on `gitt add`.
gitt() {
  if [ "$1" = "add" ]; then
    shift
    target="$(command gitt add --print-path "$@")" || return $?
    [ -n "$target" ] && cd "$target"
  else
    command gitt "$@"
  fi
}
```

`--print-path` 가 설정되면 진행 상황 메시지는 stderr로, stdout에는 worktree
절대 경로 한 줄만 출력 — 래퍼가 이 값을 캡처해 이동함. 래퍼 없이 사용하면
`gitt add` 는 기존과 동일하게 동작함.

## Install

원라인 설치 (macOS / Linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/foreverfl/gitt/main/install.sh | sh
```

- 기본 설치 위치: `$HOME/.local/bin/gitt` — `GITT_INSTALL_DIR` 로 덮어쓸 수 있음
- 특정 버전: `GITT_VERSION=v0.0.1 curl ... | sh`
- `~/.local/bin` 이 PATH에 없으면 셸 rc (zsh는 `~/.zshrc`, bash는 `~/.bashrc`)
  에 아래 줄을 추가하고 셸을 다시 로드:

  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```

## Uninstall

```bash
gitt uninstall
```

데몬 종료 → `~/.gitt/` (sock, pid, log, db) 삭제 → 바이너리(`os.Executable()`) 삭제.
프롬프트 없이 가려면 `-y` / `--yes`.
