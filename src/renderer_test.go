package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func renderHTML(t *testing.T, src string) string {
	t.Helper()
	out, _, _, err := convertMarkdown([]byte(src), ConvertOptions{LinkRewrite: true})
	if err != nil {
		t.Fatalf("convertMarkdown error: %v", err)
	}
	return out
}

func TestHeading_KoreanSlugID(t *testing.T) {
	html := renderHTML(t, "## API 설계\n### 1. 개요")
	if !strings.Contains(html, `id="api-설계"`) {
		t.Errorf("missing korean slug id on h2:\n%s", html)
	}
	if !strings.Contains(html, `id="1-개요"`) {
		t.Errorf("missing slug id on h3:\n%s", html)
	}
}

func TestHeading_CollisionSuffix(t *testing.T) {
	html := renderHTML(t, "## 개요\n## 개요\n## 개요")
	if !strings.Contains(html, `id="개요"`) ||
		!strings.Contains(html, `id="개요-2"`) ||
		!strings.Contains(html, `id="개요-3"`) {
		t.Errorf("collision suffix not applied:\n%s", html)
	}
}

func TestHeading_H1H2SameTextCollision(t *testing.T) {
	// h1 and h2 share text; h1 should win the bare slug, h2 should get -2.
	html := renderHTML(t, "# 개요\n## 개요")
	if !strings.Contains(html, `<h1 id="개요">`) {
		t.Errorf("h1 should get bare slug:\n%s", html)
	}
	if !strings.Contains(html, `<h2 id="개요-2">`) {
		t.Errorf("h2 should get -2 suffix on text collision with h1:\n%s", html)
	}
}

func TestHeading_InsideBlockquote(t *testing.T) {
	html := renderHTML(t, "> ## 개요\n> body")
	if !strings.Contains(html, `<h2 id="개요">`) {
		t.Errorf("heading inside blockquote should still get slug id:\n%s", html)
	}
}

func TestCodeBlock_MermaidBranch(t *testing.T) {
	html := renderHTML(t, "```mermaid\nflowchart LR\n  A --> B\n```")
	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Errorf("mermaid block must use <pre class=\"mermaid\">:\n%s", html)
	}
	if strings.Contains(html, `language-mermaid`) {
		t.Errorf("mermaid block must not have language-* class:\n%s", html)
	}
	if !strings.Contains(html, "flowchart LR") {
		t.Errorf("mermaid content missing:\n%s", html)
	}
}

func TestCodeBlock_RegularLanguage(t *testing.T) {
	html := renderHTML(t, "```go\nfmt.Println(\"hi\")\n```")
	if !strings.Contains(html, `<pre><code class="language-go">`) {
		t.Errorf("regular code block must use <pre><code class=\"language-X\">:\n%s", html)
	}
	if !strings.Contains(html, `fmt.Println(&#34;hi&#34;)`) && !strings.Contains(html, `fmt.Println(&quot;hi&quot;)`) {
		t.Errorf("content must be HTML-escaped:\n%s", html)
	}
}

func TestCodeBlock_NoLanguage(t *testing.T) {
	html := renderHTML(t, "```\nplain text\n```")
	if !strings.Contains(html, `<pre><code class="language-plaintext">`) {
		t.Errorf("missing lang must default to plaintext:\n%s", html)
	}
}

func TestCodeBlock_LangSanitizeMixedCase(t *testing.T) {
	html := renderHTML(t, "```Go\nfmt.Println(\"hi\")\n```")
	if !strings.Contains(html, `class="language-go"`) {
		t.Errorf("lang should be lowercased to 'go':\n%s", html)
	}
}

func TestCodeBlock_LangSanitizeRejectsBreakout(t *testing.T) {
	// Hostile fence info. Whatever the fence parser produces as `lang` must be
	// sanitized to [a-z0-9-] or fall back to plaintext.
	for _, evil := range []string{
		"```x\"><script>alert(1)</script><x\nbody\n```",
		"```语言\nbody\n```", // non-ASCII
		"```go;evil\nbody\n```",
	} {
		html := renderHTML(t, evil)
		if !strings.Contains(html, `class="language-plaintext"`) {
			t.Errorf("hostile lang must fall back to plaintext, got:\n%s", html)
		}
		if strings.Contains(html, "<script>") || strings.Contains(html, "onerror=") {
			t.Errorf("hostile lang must not leak attribute breakout:\n%s", html)
		}
	}
}

func TestCodeBlock_LangAfterSpaceIgnored(t *testing.T) {
	// CommonMark fence info: only the token before the first whitespace is the
	// language. Metadata after is dropped by goldmark before our renderer sees it.
	html := renderHTML(t, "```js onerror=alert(1)\nbody\n```")
	if !strings.Contains(html, `class="language-js"`) {
		t.Errorf("expected language-js, got:\n%s", html)
	}
	if strings.Contains(html, "onerror=") {
		t.Errorf("metadata after lang must not appear in output:\n%s", html)
	}
}

func TestCodeBlock_MermaidEscape(t *testing.T) {
	html := renderHTML(t, "```mermaid\nA-->>B: Mono<Foo>\n```")
	// inside <pre class="mermaid">, < and > must be HTML-escaped so the browser
	// returns the original source via textContent.
	if !strings.Contains(html, "Mono&lt;Foo&gt;") {
		t.Errorf("mermaid content must be HTML-escaped:\n%s", html)
	}
}

func TestTaskList_Classes(t *testing.T) {
	html := renderHTML(t, "- [ ] todo\n- [x] done\n")
	if !strings.Contains(html, `<ul class="contains-task-list">`) {
		t.Errorf("missing contains-task-list class on ul:\n%s", html)
	}
	occ := strings.Count(html, `<li class="task-list-item">`)
	if occ != 2 {
		t.Errorf("expected 2 task-list-item, got %d:\n%s", occ, html)
	}
	if !strings.Contains(html, `<input type="checkbox" disabled>`) {
		t.Errorf("missing disabled checkbox:\n%s", html)
	}
	if !strings.Contains(html, `<input type="checkbox" checked disabled>`) &&
		!strings.Contains(html, `<input type="checkbox" disabled checked>`) {
		t.Errorf("missing checked checkbox:\n%s", html)
	}
}

func TestTaskList_LooseListNoParagraphWrapper(t *testing.T) {
	// Loose list (blank lines between items) — goldmark turns the inner text
	// into a Paragraph. Our paragraph renderer must strip <p> for task items.
	src := "- [ ] one\n\n- [x] two\n"
	html := renderHTML(t, src)
	if strings.Contains(html, "<li class=\"task-list-item\"><p>") {
		t.Errorf("loose task-list-item must not have <p> wrapper:\n%s", html)
	}
	if !strings.Contains(html, `<li class="task-list-item"><input type="checkbox" disabled> one`) {
		t.Errorf("expected unwrapped checkbox+text for first item:\n%s", html)
	}
}

func TestTaskList_NonTaskItemKeepsParagraph(t *testing.T) {
	// In a loose plain list (no checkboxes), paragraphs are preserved.
	html := renderHTML(t, "- one\n\n- two\n")
	if !strings.Contains(html, "<p>one</p>") || !strings.Contains(html, "<p>two</p>") {
		t.Errorf("loose plain list items should keep <p> wrappers:\n%s", html)
	}
}

func TestTaskList_PlainList_NoClass(t *testing.T) {
	html := renderHTML(t, "- one\n- two\n")
	if strings.Contains(html, "contains-task-list") || strings.Contains(html, "task-list-item") {
		t.Errorf("plain list should not have task-list classes:\n%s", html)
	}
}

func TestLink_RelativeMdToHtml(t *testing.T) {
	html := renderHTML(t, "[design](./design.md)")
	if !strings.Contains(html, `href="./design.html"`) {
		t.Errorf("expected .md → .html rewrite:\n%s", html)
	}
}

func TestLink_FragmentPreserved(t *testing.T) {
	html := renderHTML(t, "[overview](../../overview.md#목표)")
	if !strings.Contains(html, `href="../../overview.html#목표"`) {
		t.Errorf("expected fragment preserved:\n%s", html)
	}
}

func TestLink_AbsoluteUnchanged(t *testing.T) {
	for _, link := range []string{
		"[x](https://example.com/a.md)",
		"[x](http://example.com/a.md)",
		"[x](mailto:foo@example.com)",
		"[x](tel:+1234)",
	} {
		html := renderHTML(t, link)
		if strings.Contains(html, ".html") {
			t.Errorf("absolute/protocol link must not be rewritten: %q →\n%s", link, html)
		}
	}
}

func TestLink_NonMdUnchanged(t *testing.T) {
	html := renderHTML(t, "[img](./pic.png)")
	if !strings.Contains(html, `href="./pic.png"`) {
		t.Errorf("non-md path must be unchanged:\n%s", html)
	}
}

func TestAutoLink_EmailGetsMailtoPrefix(t *testing.T) {
	html := renderHTML(t, "<foo@bar.com>")
	if !strings.Contains(html, `href="mailto:foo@bar.com"`) {
		t.Errorf("email autolink missing mailto: prefix:\n%s", html)
	}
}

func TestAutoLink_GfmEmailLinkifyGetsMailtoPrefix(t *testing.T) {
	// GFM Linkify converts bare emails to AutoLink nodes too.
	html := renderHTML(t, "contact foo@bar.com please")
	if !strings.Contains(html, `href="mailto:foo@bar.com"`) {
		t.Errorf("GFM-linkified email missing mailto: prefix:\n%s", html)
	}
}

func TestAutoLink_UrlUnchanged(t *testing.T) {
	html := renderHTML(t, "<https://example.com>")
	if !strings.Contains(html, `href="https://example.com"`) {
		t.Errorf("URL autolink missing or rewritten:\n%s", html)
	}
	if strings.Contains(html, "mailto:") {
		t.Errorf("URL autolink must not get mailto: prefix:\n%s", html)
	}
}

func TestLink_RewriteDisabled(t *testing.T) {
	out, _, _, err := convertMarkdown([]byte("[design](./design.md)"), ConvertOptions{LinkRewrite: false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `href="./design.md"`) {
		t.Errorf("rewrite disabled but .md still rewritten:\n%s", out)
	}
}

func TestLink_FragmentOnlyUnchanged(t *testing.T) {
	// Fragment-only links have no .md to rewrite.
	for _, opt := range []ConvertOptions{
		{LinkRewrite: true},
		{LinkRewrite: false},
	} {
		out, _, _, err := convertMarkdown([]byte("[top](#section)"), opt)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, `href="#section"`) {
			t.Errorf("fragment-only link must be unchanged (rewrite=%v):\n%s", opt.LinkRewrite, out)
		}
	}
}

func TestLink_InTableCellRewritten(t *testing.T) {
	// GFM table is rendered by the default renderer, but link nodes inside
	// table cells must still pass through our linkRenderer.
	src := "| col |\n|---|\n| [d](./design.md) |\n"
	html := renderHTML(t, src)
	if !strings.Contains(html, `href="./design.html"`) {
		t.Errorf("link inside table cell must be rewritten:\n%s", html)
	}
}

func TestRenderFullDocument_AllPlaceholdersReplaced(t *testing.T) {
	src := "# 제목\n## 섹션\n본문 [link](./other.md)"
	html, err := RenderDocument(RenderInput{
		Source:    []byte(src),
		Filename:  "test.md",
		ShellHTML: "<html>{{TITLE}}|{{TOC}}|{{CONTENT}}</html>",
		Options:   ConvertOptions{LinkRewrite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "{{TITLE}}") || strings.Contains(html, "{{TOC}}") || strings.Contains(html, "{{CONTENT}}") {
		t.Errorf("placeholders not all replaced:\n%s", html)
	}
	if !strings.Contains(html, "제목") {
		t.Errorf("title missing:\n%s", html)
	}
	if !strings.Contains(html, "섹션") {
		t.Errorf("TOC entry missing:\n%s", html)
	}
	if !strings.Contains(html, `href="./other.html"`) {
		t.Errorf("link rewrite missing:\n%s", html)
	}
}

func TestRenderFullDocument_LeftoverPlaceholderIsError(t *testing.T) {
	_, err := RenderDocument(RenderInput{
		Source:    []byte("# t"),
		Filename:  "test.md",
		ShellHTML: "<html>{{TITLE}}|{{TOC}}|{{CONTENT}}|{{UNKNOWN}}</html>",
		Options:   ConvertOptions{LinkRewrite: true},
	})
	if err == nil {
		t.Fatal("expected error for leftover placeholder")
	}
}

func TestRenderFullDocument_MissingPlaceholderIsError(t *testing.T) {
	for _, missing := range []string{
		"<html>{{TOC}}{{CONTENT}}</html>",  // no TITLE
		"<html>{{TITLE}}{{CONTENT}}</html>", // no TOC
		"<html>{{TITLE}}{{TOC}}</html>",     // no CONTENT
		"<html>no placeholders here</html>",
	} {
		_, err := RenderDocument(RenderInput{
			Source:    []byte("# t"),
			Filename:  "test.md",
			ShellHTML: missing,
			Options:   ConvertOptions{LinkRewrite: true},
		})
		if err == nil {
			t.Errorf("expected error for shell missing placeholder, got nil. shell=%q", missing)
		}
	}
}

func TestRenderFullDocument_MultiPlaceholdersAllReplaced(t *testing.T) {
	// PR #99 shell may have {{TITLE}} in both <title> and og:title meta.
	// All occurrences should be replaced; a single residual would slip past
	// leftoverPlaceholderRE.
	shell := "<title>{{TITLE}}</title>" +
		`<meta property="og:title" content="{{TITLE}}">` +
		"<body>{{TOC}}{{CONTENT}}</body>"
	out, err := RenderDocument(RenderInput{
		Source:    []byte("# Hello"),
		Filename:  "test.md",
		ShellHTML: shell,
		Options:   ConvertOptions{LinkRewrite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "{{TITLE}}") {
		t.Errorf("multiple {{TITLE}} not all replaced:\n%s", out)
	}
	if strings.Count(out, "Hello") < 2 {
		t.Errorf("expected title 'Hello' twice, got:\n%s", out)
	}
}

func TestRenderFullDocument_NoH1UsesFilenameTitle(t *testing.T) {
	html, err := RenderDocument(RenderInput{
		Source:    []byte("no heading"),
		Filename:  "foo.md",
		ShellHTML: "<html>TITLE={{TITLE}};{{TOC}};{{CONTENT}}</html>",
		Options:   ConvertOptions{LinkRewrite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "TITLE=foo;") {
		t.Errorf("expected filename-derived title 'foo', got:\n%s", html)
	}
}

func TestGoldenIntegration(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("testdata", "sample.md"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "sample.html"))
	if err != nil {
		t.Fatal(err)
	}
	shell := "<<<TITLE>>>{{TITLE}}<<</TITLE>>>\n<<<TOC>>>{{TOC}}<<</TOC>>>\n<<<CONTENT>>>{{CONTENT}}<<</CONTENT>>>\n"
	got, err := RenderDocument(RenderInput{
		Source:    src,
		Filename:  "sample.md",
		ShellHTML: shell,
		Options:   ConvertOptions{LinkRewrite: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		// helpful diff: show first 200 chars of each
		t.Errorf("output mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}
