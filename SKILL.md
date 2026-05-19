---
name: md2html
description: 로컬 MD 파일을 aac-core PR #99의 인터랙티브 셸로 HTML 변환해 브라우저에서 본다. /tmp/md2html/<원본명>.html 생성 후 자동 오픈.
---

# md2html — MD를 HTML로 보기

aac-core PR #99의 인터랙티브 셸(상태 칩, API 배지, 검색, sticky 칩 바 등 19종 자동 강화)을 임의의 로컬 MD 파일에 적용해 브라우저로 열어준다.

## 사용

인자: $ARGUMENTS

사용자가 다음과 같이 요청할 때:
- "이 md 파일 html로 보여줘"
- "/md2html foo.md"
- "spec.md, design.md를 html로 변환해줘"
- "이 폴더의 모든 md를 html로 보고싶어" (glob 후 인자로 풀어 호출)

### 실행

대상 MD 파일을 절대 경로로 변환한 뒤 한 번에 호출:

```bash
~/.claude/skills/md2html/bin/md2html <abs-path-1> [<abs-path-2> ...]
```

### 동작
- 출력: `/tmp/md2html/<basename>.html`
- 첫 파일은 macOS `open`으로 브라우저 자동 오픈
- 동명 파일 충돌 시 `-2`, `-3` 접미사로 자동 회피

### 옵션 (필요할 때만 전달)
- `--no-open` — 브라우저 자동 오픈 끄기
- `--out-dir <path>` — 출력 디렉터리 변경 (기본 `/tmp/md2html`)
- `--shell <path>` — 다른 셸 사용 (실험·디버그용)
- `--no-link-rewrite` — `.md` → `.html` 상대 링크 재작성 끄기

### 빌드 / 재설치

바이너리가 없거나 소스를 변경했다면:

```bash
cd ~/.claude/skills/md2html/src && go build -o ../bin/md2html .
```

`go`가 없으면: `brew install go`
