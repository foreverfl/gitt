# doctree

`git worktree` + per-worktree `docker compose` 스택을 한 바이너리로 묶은 CLI.
zshrc에 흩어져 있던 `worktree-add` / `worktree-build` / `worktree-remove` 함수를 대체.

여러 worktree에서 동시에 컨테이너를 띄울 때 발생하는 **호스트 포트 race**를
SQLite 백엔드 데몬에서 직렬화해서 막는 게 핵심.

## Commands

| 명령어 | 동작 |
| --- | --- |
| `doctree on` | 데몬 기동 (`~/.doctree/doctree.sock`, `~/.doctree/doctree.db`). 다른 모든 명령은 데몬이 떠있어야 동작 |
| `doctree off` | 데몬 종료 |
| `doctree add <branch>` | `<repo>/../.worktrees/<repo>/<branch>` 에 worktree 생성, 포트 할당, `.env.worktree` 작성 |
| `doctree build` | 현재 worktree에서 `docker compose up -d` |
| `doctree remove` | 현재 worktree의 `docker compose down` + worktree 폴더 삭제 |

데몬이 안 떠있으면 `doctree add`/`doctree remove`는 즉시 에러로 끊고 `doctree on` 안내. (auto-start 안 함)

## Install

원라인 설치 (macOS / Linux, amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/foreverfl/doctree/main/install.sh | sh
```

- 기본 설치 위치: `$HOME/.local/bin/doctree` — `DOCTREE_INSTALL_DIR` 로 덮어쓸 수 있음
- 특정 버전: `DOCTREE_VERSION=v0.0.1 curl ... | sh`
- `~/.local/bin` 이 PATH에 없으면 셸 rc에 추가:

  ```bash
  export PATH="$HOME/.local/bin:$PATH"
  ```

소스 빌드:

```bash
git clone https://github.com/foreverfl/doctree ~/code/doctree
cd ~/code/doctree
go build -o ~/.local/bin/doctree .
```

## Uninstall

```bash
doctree uninstall
```

데몬 종료 → `~/.doctree/` (sock, pid, log, db) 삭제 → 바이너리(`os.Executable()`) 삭제.
프롬프트 없이 가려면 `-y` / `--yes`.

## zshrc

```bash
# 셸 시작 시 데몬 한 번 띄움 (이미 떠있으면 noop)
doctree on >/dev/null 2>&1

alias dt-add='doctree add'
alias dt-on='doctree on'
alias dt-off='doctree off'
alias dt-build='doctree build'
alias dt-remove='doctree remove'
```

## Repo convention

`doctree add` 가 호출되는 repo는 다음을 가정:

- `infra/docker/compose.local.yml` — compose 파일
- `infra/docker/.env.local` (선택) — 공용 기본값
- `infra/docker/.env.worktree` — `doctree add` 가 worktree별 포트 매핑을 써넣는 파일

`compose.local.yml` 쪽 변경 사항:

- `container_name` 라인 제거 → compose가 `<project>-<service>-<n>` 으로 자동 부여
- 호스트 포트는 `"${POSTGRES_HOST_PORT:-5432}:5432"` 형태로 변수화
- 컨테이너 내부 포트와 서비스 간 통신(`postgres:5432`, `redis:6379`)은 그대로 둠

## Status

`on` / `off` 까지 동작. `add` / `build` / `remove` 등 나머지는 TODO.
