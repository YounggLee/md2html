---
name: md2html
description: 로컬 .md 파일을 인터랙티브 HTML로 변환해 브라우저로 연다. 사용자가 마크다운을 "보고 싶다 / 미리보기 / 렌더링 / html로 띄워줘 / preview" 류로 요청하면 — 도구명을 명시하지 않더라도 — 호출. 출력은 /tmp/md2html/<원본명>.html, 첫 파일 자동 오픈.
---

# md2html — MD를 HTML로 보기

기존 `.md` 파일 **변환 전용**. HTML 신규 생성·디자인은 다른 스킬의 영역.

## 호출 전 Claude가 할 일

- 모든 인자를 **절대 경로**로 변환 (도구는 상대 경로를 안 받음)
- 폴더/glob 요청(`docs/*.md`, "이 폴더 전체")이면 Claude가 먼저 확장해서 절대 경로 리스트로 풀어준다
- 결과 파일들을 **한 번에 호출** (파일당 한 번씩 호출 ❌ — 첫 파일만 오픈되는 동작이 깨진다)
- 누락 파일은 도구가 stderr + exit 1로 보고 — 그대로 사용자에게 전달

## 호출

```bash
~/.claude/skills/md2html/bin/md2html <abs-path-1> [<abs-path-2> ...]
```

### 트리거 예시
- "이 md 파일 html로 보여줘 / 미리보기 / 렌더링해줘"
- "/md2html foo.md"
- "spec.md, design.md를 html로 변환해줘"
- "이 폴더의 모든 md를 보고 싶어"

### 동작
- 출력: `/tmp/md2html/<basename>.html`
- 첫 파일은 macOS `open`으로 자동 오픈 (macOS 전용)
- 동명 파일 충돌 시 `-2`, `-3` 접미사로 자동 회피

### 옵션
- `--no-open` — 자동 오픈 끄기
- `--out-dir <path>` — 출력 디렉터리 변경 (기본 `/tmp/md2html`)
- `--shell <path>` — 다른 셸 사용 (실험·디버그)
- `--no-link-rewrite` — `.md` → `.html` 상대 링크 재작성 끄기
