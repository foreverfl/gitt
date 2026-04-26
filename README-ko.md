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
| `gitt on` | 데몬 기동 (`~/.gitt/gitt.sock`, `~/.gitt/gitt.db`) |
| `gitt off` | 데몬 종료 |
| `gitt add <branch>` | `<repo-parent>/.worktrees/<repo-name>/<branch>` 에 worktree 생성. 브랜치가 존재하면 체크아웃, 없으면 신규 생성. 브랜치명의 `/`·`\`는 `-`로 변환됨. **데몬 필요** |
| `gitt remove <branch>` | 해당 브랜치의 worktree 폴더 삭제 (`git worktree remove`). **데몬 필요** |
| `gitt status` | 현재 worktree의 저장소, 브랜치, 경로, 상태 (clean/dirty/rebase/merge/conflict 등) 출력 |
| `gitt sqlite` | 데몬의 DB 연결이 정상인지 확인하는 SQLite self-test 실행. **데몬 필요** |
| `gitt update` | 최신 릴리스를 받아 설치 |
| `gitt version` | 설치된 gitt 버전 출력 |
| `gitt logo` | gitt 로고 아트를 하늘색 박스로 출력 |
| `gitt uninstall` | 데몬 종료 → `~/.gitt/` 삭제 → 바이너리 삭제. `-y`/`--yes` 로 확인 프롬프트 생략 |

데몬이 필요한 명령은 데몬이 떠있지 않으면 즉시 에러로 끊고 `gitt on` 안내.
(auto-start 안 함)

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
