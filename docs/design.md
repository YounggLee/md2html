# md2html Skill — 설계 문서

- **작성일**: 2026-05-19
- **작성자**: yj.l@navercorp.com (Claude brainstorming 협업)
- **상태**: Draft (사용자 리뷰 대기)

## 목표

aac-core PR #99에서 도입된 "MD에 인터랙티브 HTML 셸을 적용하는" 기능을 임의의 로컬 MD 파일에도 적용할 수 있도록 한다. CLI 도구와 Claude Code 스킬을 함께 만들어, 사용자가 자연어 또는 `/md2html` 호출로 즉시 변환·확인할 수 있게 한다.

## 컨텍스트

### PR #99의 셸

aac-core PR #99(`feat/docs_html` → develop)는 `spec/templates/html-shell.html` (~30KB)을 도입했다. 이 셸은:

- 자기완결형 인터랙티브 셸. CDN(github-markdown-css, highlight.js, Mermaid) + 내장 CSS·JS.
- `{{TITLE}}`, `{{TOC}}`, `{{CONTENT}}` 세 개의 placeholder만 치환해 사용.
- 19종 자동 강화를 런타임에 적용: 상태 칩, API 메서드 배지, 체크박스 진행률, 헤딩 호버 앵커, 코드 복사 버튼, sticky 섹션 칩 바, 긴 코드 블록 자동 접기, 본문 검색(`/` 키), 요구사항 필터 알약, 인터페이스 병렬 비교 등.

### 기존 자동화 없음

PR #99~#102 분석 결과, 변환 자동화 코드는 어디에도 없다. PR #102가 49개 spec MD를 일괄 HTML로 만들 때도 Claude가 가이드(`html-rendering-guide.md`)를 보고 수작업으로 변환한 결과를 commit했다. 이 도구가 사실상 첫 자동화다.

### 사용 시나리오

- aac-core spec 폴더 외부의 임의 MD 파일을 PR #99 셸로 빠르게 보고 싶다.
- 자동 강화(검색·진행률·sticky 칩 바 등)를 활용해 긴 MD를 "읽는" 문서로 만든다.
- `/tmp` 영역에 휘발성으로 두고 빠르게 확인 → 영구 산출물은 아님.

## 비목표 (NON-GOALS)

- aac-core의 SDD 워크플로우 자동화. (이건 Claude의 CLAUDE.md 워크플로우 영역이고, 본 도구는 일반 사용자 도구)
- 셸 자체 개선. (셸은 PR #99 산출물을 그대로 사용)
- 자동 강화 19종의 트리거 로직 변경. (셸 JS가 런타임에 알아서 적용)
- aac-core 셸의 자동 동기화. (셸은 거의 변하지 않으므로 수동 갱신 후 재빌드로 충분)
- 폴더 재귀 변환. (단일 파일 인자만 받고, glob 확장은 Claude/shell이 담당)

## 아키텍처

```
사용자: Claude 대화에서 "/md2html foo.md" 또는 "이 md 파일 html로 보여줘"
   │
   ▼
md2html Skill (~/.claude/skills/md2html/SKILL.md)
   - Claude에게 호출 방법 지시
   │
   ▼ Bash 호출
md2html (Go 단일 바이너리, ~/.claude/skills/md2html/bin/md2html)
   - goldmark로 MD → HTML 변환
   - 셸을 //go:embed로 박은 상태
   - {{TITLE}}, {{TOC}}, {{CONTENT}} placeholder 치환
   │
   ▼ writes
/tmp/md2html/<basename>.html
   │
   ▼ macOS `open`
브라우저 자동 오픈 (첫 파일만)
```

### 산출물

1. **Go 소스 + 빌드 산출물** (`~/.claude/skills/md2html/src/`, `bin/`)
2. **셸 복사본** (`~/.claude/skills/md2html/src/shell.html`, `//go:embed`로 바이너리에 박힘)
3. **SKILL.md** (`~/.claude/skills/md2html/SKILL.md`, Claude 트리거 지시문)

## CLI 인터페이스

```bash
md2html foo.md                    # /tmp/md2html/foo.html, 브라우저 자동 오픈
md2html a.md b.md docs/c.md       # 여러 파일, 첫 파일만 오픈
md2html --no-open foo.md          # 오픈 끄기
md2html --out-dir /tmp/x foo.md   # 출력 디렉터리 변경
md2html --shell <path> foo.md     # 다른 셸로 변환 (실험·디버그)
md2html --no-link-rewrite foo.md  # .md → .html 상대 링크 재작성 끄기
md2html -h                        # 도움말
```

### 동작 세부

- **출력 위치**: `/tmp/md2html/<basename>.html`
- **파일명 충돌**: 같은 실행 내에서 동명 파일이 두 번 출력되면 두 번째부터 `-2`, `-3` suffix
- **브라우저 오픈**: macOS `open` 명령. 여러 파일이면 **첫 번째**만 (남발 방지)
- **종료 코드**:
  - 모든 변환 성공 → 0
  - 변환 단계에서 일부/전부 실패 → 1
  - 변환은 모두 성공했지만 브라우저 자동 오픈만 실패 → 0 (stderr 경고만). 부수효과 실패가 메인 작업 결과를 가리지 않는다는 Unix 관행에 부합.
  - `--help` → 0
- **stdout 한 줄/파일**: `foo.md → /tmp/md2html/foo.html`
- **에러는 stderr**: 파일 없음 등은 stderr로 보고, 나머지 파일은 계속 변환
- **인코딩**: UTF-8 가정, BOM 자동 제거

## MD → HTML 변환 규칙

가이드(`spec/templates/html-rendering-guide.md`)의 모든 규칙을 goldmark + custom NodeRenderers로 구현한다.

### goldmark 구성

```go
md := goldmark.New(
    goldmark.WithExtensions(extension.GFM),  // 표·취소선·task list·autolink
    goldmark.WithRendererOptions(html.WithUnsafe()),  // 인라인 HTML 보존
)
```

`parser.WithAutoHeadingID()`는 쓰지 않는다. 한글 슬러그 처리와 h1/h2/h3 사이 충돌 해결을 위해 직접 ID를 부여하기 때문이다. `buildTOC`가 한 번 walk 하면서 한글-친화 슬러그를 `__md2html_id` AST attribute에 박고, heading renderer가 그 값을 그대로 회수해 `<h_ id=...>`로 출력한다.

### 규칙 매핑

| 가이드 규칙 | 구현 방식 |
|---|---|
| `# 제목` → `<h1 id="slug">` | custom IDProvider — 한글 유지, 공백/`/`/`.`/`:` → `-`, 기타 비단어문자 제거, 충돌 시 `-2`/`-3` |
| `## ...`, `### ...` 동일 | 동일 IDProvider |
| 체크박스 → `<ul class="contains-task-list"><li class="task-list-item">...</li></ul>` | custom ListItem renderer (goldmark 기본은 클래스를 붙이지 않음). `<input type="checkbox" disabled>` 부착, 체크 시 `checked` 속성 추가, 텍스트는 `<p>` 래퍼 없이 평문 |
| Mermaid 펜스 → `<pre class="mermaid">` | custom FencedCodeBlock renderer — `info == "mermaid"`이면 `<pre class="mermaid">` + 이스케이프 본문 (code 래퍼·language-* 없음) |
| 일반 코드 펜스 → `<pre><code class="language-X">` | 동일 renderer의 else 분기. `lang`은 소문자로 정규화 후 `[a-z0-9-]`만 통과(strict sanitize), 비ASCII·breakout 시도·빈 값은 `plaintext` 폴백 |
| Loose task list의 `<p>` 래핑 제거 | paragraph renderer 추가 — 부모가 task-list-item이면 `<p>` 태그 skip, 그 외는 기본 `<p>...</p>` 동작 |
| 이메일 autolink (`<foo@bar>`, GFM linkify) | `mailto:` prefix 자동 prepend (이미 있으면 유지) |
| 모든 코드 블록 HTML 이스케이프 | goldmark 기본 + Mermaid 분기에서도 동일 처리 |
| `.md` → `.html` 상대 링크 재작성 | custom Link renderer — 프래그먼트 보존, http(s)/mailto/tel은 미변경, `--no-link-rewrite`로 끄기 가능 |
| `{{TITLE}}` | AST 1차 패스로 첫 h1 텍스트 추출, 없으면 파일명(확장자 제외) |
| `{{TOC}}` (h2/h3만) | AST 1차 패스로 h2/h3 수집 후 `<ul>` 빌드. 헤딩 없으면 `<p>(목차 없음)</p>` |
| 인라인 HTML 보존 | `html.WithUnsafe()` |

### 처리 흐름

```
1. MD 파일 읽기 (UTF-8, BOM 제거)
2. AST 파싱: md.Parser().Parse(text.NewReader(source))
3. AST 1차 순회: TITLE 추출, TOC 빌드
4. AST 렌더링 (custom renderers): HTML 본문 문자열
5. embedded shell 로드 (또는 --shell 경로)
6. placeholder 치환: {{TITLE}}, {{TOC}}, {{CONTENT}}
7. 잔류 placeholder 검증 (잔류 시 에러)
8. /tmp/md2html/<basename>.html 쓰기
9. (첫 파일만) macOS `open <path>`
```

### 상대 링크 재작성의 일반화

PR #99 가이드의 "제외 목록" (`tdd-guide.md`, `glossary.md`, `templates/**`, `README.md`, `CLAUDE.md`, `request.md`)은 aac-core 컨텍스트 전용이다. 본 도구는 임의 폴더에서도 동작해야 하므로:

- **기본 동작**: 모든 `.md` 상대 링크를 `.html`로 재작성
- **off 옵션**: `--no-link-rewrite` 플래그로 끌 수 있음

aac-core 셸의 정교한 제외 로직은 본 도구의 책임이 아니다.

### 자동 강화는 도구의 책임이 아님

상태 칩·API 배지·검색·sticky 칩 바 등 19종 강화는 셸 JS가 런타임에 자동 적용한다. 본 도구는 가이드의 MD→HTML 매핑까지만 책임진다.

## 셸 번들 정책

### 번들 방식: `//go:embed`

`src/shell.html`을 `//go:embed`로 바이너리에 박는다.

```go
//go:embed shell.html
var shellHTML string
```

- 산출물 = 단일 바이너리. 경로 못 찾는 사고 없음.
- 셸 갱신은 `src/shell.html` 갱신 + 재빌드 = 두 줄.

### 셸 출처

`/Users/user/aac/aac-core/spec/templates/html-shell.html` (PR #99 산출)을 초기에 1회 복사한다. PR 히스토리상 도입 후 1회만 수정됐으므로 자주 갱신할 필요 없다.

### Override

`--shell <path>` 플래그로 런타임에 다른 셸 파일을 지정할 수 있다 (실험·디버그용). 지정한 셸에 `{{TITLE}}`/`{{TOC}}`/`{{CONTENT}}` 세 placeholder가 모두 있어야 하며, 하나라도 누락이면 즉시 에러로 종료한다 (콘텐츠 침묵 손실 방지).

### placeholder 치환·검증

- **치환**: `strings.ReplaceAll`로 각 placeholder의 **모든 출현**을 치환한다. PR #99 셸은 각 placeholder가 1회만 등장하지만, 셸 갱신으로 `{{TITLE}}`이 `<title>`과 `og:title` meta 두 곳에 등장하는 등의 변경에도 안전.
- **부재 검증** (입력): 호출 시 셸에 세 placeholder가 모두 들어있는지 확인 (위 `--shell` Override 참고).
- **잔류 검증** (출력): 최종 HTML에 `{{...}}` 형태가 남아 있으면 에러. 미지의 placeholder가 추가됐다는 신호.

## Skill 구조

### 디렉터리

```
~/.claude/skills/md2html/
├── SKILL.md
├── bin/
│   └── md2html
└── src/
    ├── main.go
    ├── slugify.go
    ├── toc.go
    ├── renderer.go
    ├── shell.html
    ├── go.mod
    ├── go.sum
    └── testdata/
        ├── sample.md
        └── sample.html
```

### SKILL.md (요약)

```markdown
---
name: md2html
description: 로컬 MD 파일을 PR #99 셸로 HTML 변환해서 브라우저로 본다.
---

# md2html — MD를 HTML로 보기

인자: $ARGUMENTS

대상 MD 파일 절대 경로로 한 번에 호출:
```bash
~/.claude/skills/md2html/bin/md2html <abs-path-1> [<abs-path-2> ...]
```

옵션 (필요 시): --no-open, --out-dir <path>, --shell <path>, --no-link-rewrite

빌드: cd ~/.claude/skills/md2html/src && go build -o ../bin/md2html .
```

### Claude vs 도구의 책임

| | Claude | 도구 |
|---|---|---|
| 사용자 발화 해석 | ✅ | ❌ |
| 파일 경로 절대 경로화 | ✅ | ❌ |
| Glob 확장 (`./docs/*.md`) | ✅ | ❌ |
| MD → HTML 변환 | ❌ | ✅ |
| placeholder 치환 | ❌ | ✅ |
| 브라우저 오픈 | ❌ | ✅ |

도구는 단순하게 — 절대 경로 인자만 받고 변환만 한다.

## 빌드 / 설치 / 테스트

### 초기 설치 (사용자 1회 수행)

```bash
brew install go                                                   # Go 없으면
mkdir -p ~/.claude/skills/md2html/{src,bin}
# (소스 파일은 spec/plan 단계에서 작성)
cp /Users/user/aac/aac-core/spec/templates/html-shell.html \
   ~/.claude/skills/md2html/src/shell.html
cd ~/.claude/skills/md2html/src
go mod tidy
go build -o ../bin/md2html .
~/.claude/skills/md2html/bin/md2html --help                       # 검증
```

### 의존성

```
github.com/yuin/goldmark v1.7.x
```

`extension.GFM`이 task list·표·autolink·strikethrough를 모두 포함하므로 추가 의존성 없음.

### 테스트 전략

**단위 테스트** (Go 표준 `testing`):

| 테스트 대상 | 케이스 |
|---|---|
| 한글 슬러그 | `"API 설계"` → `"api-설계"`, `"1. 개요"` → `"1-개요"`, 충돌 시 `-2` suffix |
| Mermaid 분기 | mermaid 펜스 → `<pre class="mermaid">`, 그 외 → `<pre><code class="language-X">` |
| task-list-item | `- [ ] foo` → `<ul class="contains-task-list"><li class="task-list-item"><input type="checkbox" disabled> foo</li></ul>` (checked 시 `checked` 추가) |
| 상대 링크 재작성 | `./design.md` → `./design.html`, `./design.md#section` → `./design.html#section`, `http://...` 미변경, `--no-link-rewrite` 시 미변경 |
| TOC 빌드 | h2/h3 수집, h1 제외, 빈 문서 → `<p>(목차 없음)</p>` |
| placeholder 치환 + 잔류 검증 | 셸에서 placeholder 모두 치환됨 / 누락 시 에러 |

**통합 테스트** (golden file):

- `testdata/sample.md` — 체크박스·Mermaid·표·한글 헤딩·상대 링크 포함
- `testdata/sample.html` — 기대 결과
- `go test`에서 변환 결과와 비교

**수동 검증** — 빌드 후 aac-core spec 파일과 결과 비교:
```bash
~/.claude/skills/md2html/bin/md2html /Users/user/aac/aac-core/spec/features/20-logging-filter/spec.md
# 결과를 PR #102가 commit한 spec.html과 비교
```

## 향후 유지보수

- **셸 갱신**: `cp` 후 재빌드 (두 줄)
- **변환 규칙 변경**: 해당 NodeRenderer 수정 후 재빌드
- **golden file** 테스트가 리그레션 감지

## 결정 사항 요약

| # | 결정 | 대안 |
|---|---|---|
| 1 | Go + goldmark 자체 도구 | gm 그대로(가이드 호환 부족) / Rust(첫 빌드 느림) / Python(setup 부담) |
| 2 | `//go:embed`로 셸 박기 | 별도 파일 (경로 사고 위험) |
| 3 | `/tmp/md2html/<basename>.html` | 일자별 폴더(YAGNI) / 해시 폴더(과도) |
| 4 | 단일 파일 인자 다수 | 폴더 재귀(Claude/shell이 glob) |
| 5 | 첫 파일만 자동 오픈 | 전부 오픈(남발) / 오픈 안 함(불편) |
| 6 | `.md` → `.html` 재작성 기본 ON | aac-core 제외 목록 일반화(컨텍스트 의존) |
| 7 | 19종 강화는 셸 JS 담당 | 도구가 직접 강화(중복) |

## Open Questions

- 사용자 PATH 등록을 자동으로 할지 (`~/.local/bin` symlink) — 일단 수동 권장
- 비macOS 환경 지원 — 현재는 macOS `open` 가정. Linux는 `xdg-open`으로 폴백할지는 보류.
- 다중 파일 시 단일 index.html 생성 옵션 — YAGNI, 보류.
