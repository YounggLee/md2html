# md2html

[![License](https://img.shields.io/badge/license-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-macOS-lightgrey)](#)

로컬 Markdown 파일을 인터랙티브 HTML 페이지로 변환해 브라우저로 열어주는 Go CLI.
TOC, 헤딩 앵커, 코드 복사 버튼, 본문 검색(`/`), Mermaid 다이어그램이 한 파일에 자동으로 들어간다.

[Claude Code](https://docs.claude.com/en/docs/claude-code) 스킬로도 동작해서
"이 md 파일 html로 보여줘" 같은 자연어 요청만으로도 변환·오픈이 된다.

## Quickstart

```bash
git clone <this-repo-url> md2html
cd md2html
make install            # build + ~/.claude/skills/md2html에 symlink 설치
md2html ./README.md     # 절대 경로/상대 경로 OK (CLI는 그대로 처리)
```

전제: macOS, Go 1.21+ (`brew install go`).

설치 후 두 경로 다 동작한다:

```bash
# 직접
./bin/md2html /abs/path/foo.md

# 스킬을 통한 호출 (인스톨된 symlink)
~/.claude/skills/md2html/bin/md2html /abs/path/foo.md
```

## 기능

- **자동 TOC** — 사이드바에 `h2`/`h3` 트리. 스크롤 위치에 따라 현재 섹션 강조.
- **헤딩 앵커** — 한글 슬러그(`## API 설계` → `id="api-설계"`). 호버 시 `¶` 버튼.
- **코드 복사 버튼** — 모든 코드 블록에 자동 부착.
- **본문 검색** — `/` 키로 검색창. 매치 하이라이트 + 카운트.
- **Mermaid 다이어그램** — ``` ```mermaid ``` 펜스 → 자동 렌더.
- **체크박스 진행률** — `- [x] / - [ ]`의 비율을 상단 progress bar로.
- **`.md` 상대 링크 자동 재작성** — `[design](./design.md)` → `[design](./design.html)`.
- **사용자 인자 처리** — 동명 파일 `-2`/`-3` 접미사로 회피, BOM 자동 제거.

## CLI 사용법

```bash
md2html <file.md> [<file.md> ...]
```

| 옵션 | 의미 |
|---|---|
| `--no-open` | 브라우저 자동 오픈 끄기 |
| `--out-dir <path>` | 출력 디렉터리 변경 (기본 `/tmp/md2html`) |
| `--shell <path>` | 다른 셸 HTML 파일 사용 (실험·디버그) |
| `--no-link-rewrite` | `.md` → `.html` 상대 링크 재작성 끄기 |

동작:
- 출력은 `/tmp/md2html/<basename>.html`
- 첫 파일은 `open` 명령으로 자동 오픈 (macOS)
- 일부 파일 실패해도 나머지는 계속 변환, 마지막에 `exit 1`

## Claude Code 스킬

`make install`이 만든 symlink (`~/.claude/skills/md2html/`)가 SKILL.md를 가리키고 있어
Claude Code 가 자동으로 스킬을 인식한다.

```text
> 이 md 파일 html로 보여줘: docs/plan.md

(Claude가 절대 경로화 → md2html 호출 → 브라우저 오픈)
```

자세한 트리거 패턴과 옵션은 [`SKILL.md`](SKILL.md) 참고.

## Makefile 타겟

| 타겟 | 동작 |
|---|---|
| `make build` | `src/` 컴파일 → `bin/md2html` |
| `make test` | `go test ./...` (29개 테스트) |
| `make install` | build + symlink을 `~/.claude/skills/md2html/`에 |
| `make uninstall` | symlink 제거 (소스는 그대로) |
| `make clean` | `bin/md2html` 삭제 |

## 변환 규칙 요약

- 한글 헤딩 슬러그 (`## API 설계` → `id="api-설계"`), 충돌 시 `-2`, `-3` 접미사
- Mermaid 펜스 → `<pre class="mermaid">`
- 일반 코드 펜스 → `<pre><code class="language-X">` (lang 누락 시 `language-plaintext`)
- Task list → `<ul class="contains-task-list"><li class="task-list-item">…`
- `.md` 상대 링크 → `.html`로 재작성 (`--no-link-rewrite`로 끄기)
- 첫 `h1` 텍스트 또는 파일명을 `{{TITLE}}`로 치환, `h2/h3`만 `{{TOC}}`에 포함

전체 사양은 [`docs/design.md`](docs/design.md), 구현 작업 분해는 [`docs/plan.md`](docs/plan.md) 참고.

## 셸 갱신

`src/shell.html`이 브라우저에 표시되는 셸. 자동 강화 JS/CSS가 모두 들어있다.
교체 → `make install` 한 번 더 → 바이너리에 다시 임베드된다 (`//go:embed`).

## 개발

```bash
make test               # 단위 + golden 통합 테스트
cd src && go vet ./...  # lint
```

`src/testdata/sample.md`와 `sample.html`이 golden file 회귀 테스트의 기준이다.
변환 규칙을 바꿨다면 golden을 재생성해야 한다:

```bash
cat > /tmp/_test_shell.html <<EOF
<<<TITLE>>>{{TITLE}}<<</TITLE>>>
<<<TOC>>>{{TOC}}<<</TOC>>>
<<<CONTENT>>>{{CONTENT}}<<</CONTENT>>>
EOF
./bin/md2html --no-open --shell /tmp/_test_shell.html \
              --out-dir src/testdata src/testdata/sample.md
```

## 라이선스

[Apache License 2.0](LICENSE).
번들된 `src/shell.html`의 출처와 attribution은 [NOTICE](NOTICE) 참고.
